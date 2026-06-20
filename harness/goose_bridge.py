#!/usr/bin/env python3
"""Goose-backed bridge: let an existing agent runtime (Block's Goose, driving a
local llama-server) operate in an Anytype channel.

This is the "real harness" upgrade over agent_mvp.py. Same I/O loop —

  anytype notify wait   ->  pending notifications (JSON, drop-free)
  for each chat item:
    react 👀                       (claim signal)
    goose run --name <chat> --resume --system <SYSTEM> -t "<sender>: <text>"
    parse the <SAY>...</SAY> sentinel out of goose's stdout
    if not IGNORE:  anytype chat send --reply-to <id>   (threaded)
    anytype notify done <id>       (ack; at-least-once)

— but the brain is now Goose, which brings:
  * tool use / command execution (the bundled `developer` shell extension),
  * persistent per-chat memory (named session + --resume),
all against a LOCAL OpenAI-compatible endpoint (llama-server), fully offline.

Goose talks freeform (and emits tool chatter on stdout), so we make it wrap its
channel-facing message in <SAY>...</SAY> exactly once. That sentinel both (a)
lets us extract the reply cleanly past the tool sections and (b) gives Gemma an
explicit "stay silent" via <SAY>IGNORE</SAY> — the discriminated-union idea from
the MVP, expressed as an output contract instead of a JSON schema.

Run (model up first via run-llm):
  python3 harness/goose_bridge.py --space <id> \
    --goose /tmp/goose-musl/goose \
    --goose-config-home ~/.config-gemma-goose \
    --goose-data-home   ~/.local/share-gemma-goose
"""
import argparse, json, os, re, subprocess, sys, urllib.request

AP = argparse.ArgumentParser()
AP.add_argument("--anytype", default="./dist/anytype", help="path to the anytype CLI")
AP.add_argument("--goose", default="/tmp/goose-musl/goose", help="path to the goose binary")
AP.add_argument("--goose-config-home", default=os.path.expanduser("~/.config-gemma-goose"),
                help="XDG_CONFIG_HOME for goose (isolates Gemma's goose config)")
AP.add_argument("--goose-data-home", default=os.path.expanduser("~/.local/share-gemma-goose"),
                help="XDG_DATA_HOME for goose (session/memory store)")
AP.add_argument("--model", default="gemma-4-12b.gguf")
AP.add_argument("--openai-host", default="http://127.0.0.1:8080")
AP.add_argument("--key-file", default="/tmp/gemma-key.txt", help="REST bearer key (for the 👀 reaction)")
AP.add_argument("--api", default="http://localhost:31022")
AP.add_argument("--space", required=True)
AP.add_argument("--only-chat", default="", help="restrict to one chat id (else handle all)")
AP.add_argument("--state", default=os.path.expanduser("~/.anytype-gemma/notify-acked"),
                help="ack-set file (MUST differ from any other agent's)")
AP.add_argument("--timeout", type=int, default=600,
                help="seconds per goose run (high: a contained run may build a guix package)")
ARGS = AP.parse_args()
KEY = open(ARGS.key_file).read().strip()

SYSTEM = (
    "You are Gemma, a participant in a shared Anytype group chat with Terramorpha, Claude, and "
    "other people. Talk like a normal person in a group chat: natural, brief, to the point. "
    "Do NOT talk like a corporate AI assistant — skip filler like 'I'm ready to help', 'happy to "
    "assist', 'I understand', or restating the question back at people. Just say the thing. "
    "You have real tools — a shell and file editing, plus a nested guix so you can install "
    "programs with `guix shell <pkg> -- ...` — actually use them to answer or to do what's asked, "
    "then report what you found or did. Only respond when a message is addressed to you, mentions "
    "you, or clearly needs your input; the channel is shared and chatty, so stay quiet otherwise. "
    "Plain text only (the channel is not markdown). "
    "When finished, output your channel message wrapped EXACTLY once as <SAY>your message</SAY>, "
    "as the very last thing you emit. If no response is warranted, output exactly <SAY>IGNORE</SAY>."
)

ANSI = re.compile(r"\x1b\[[0-9;?]*[a-zA-Z]")
SAY = re.compile(r"<SAY>(.*?)</SAY>", re.DOTALL | re.IGNORECASE)

def sh(*args):
    return subprocess.run([ARGS.anytype, *args, "--no-update-check"],
                          capture_output=True, text=True)

def react_eyes(space_id, chat_id, msg_id):
    req = urllib.request.Request(
        f"{ARGS.api}/v1/spaces/{space_id}/chats/{chat_id}/messages/{msg_id}/reactions",
        data=b'{"emoji":"\xf0\x9f\x91\x80"}',
        headers={"Authorization": f"Bearer {KEY}", "Content-Type": "application/json"},
        method="POST")
    try:
        urllib.request.urlopen(req, timeout=8)
    except Exception as e:
        print("react failed:", e, file=sys.stderr)

def goose_env():
    env = dict(os.environ)
    env.update({
        "XDG_CONFIG_HOME": ARGS.goose_config_home,
        "XDG_DATA_HOME": ARGS.goose_data_home,
        "GOOSE_PROVIDER": "openai",
        "GOOSE_MODEL": ARGS.model,
        "OPENAI_API_KEY": "sk-dummy",
        "OPENAI_HOST": ARGS.openai_host,
        "OPENAI_BASE_PATH": "v1/chat/completions",
        "GOOSE_DISABLE_KEYRING": "1",
    })
    return env

def run_goose(session_name, user_text):
    """Run one goose turn, resuming the per-chat session. Returns the text inside
    the last <SAY>...</SAY> block, or None if goose emitted no sentinel."""
    cmd = [ARGS.goose, "run", "--name", session_name, "--resume",
           "--system", SYSTEM, "-t", user_text]
    try:
        out = subprocess.run(cmd, env=goose_env(), capture_output=True, text=True,
                             timeout=ARGS.timeout)
    except subprocess.TimeoutExpired:
        print(f"goose timeout ({ARGS.timeout}s) for {session_name}", file=sys.stderr)
        return None
    text = ANSI.sub("", out.stdout)
    matches = SAY.findall(text)
    if not matches:
        # First-ever run of a session can't --resume; retry fresh once.
        if "no session" in (out.stderr or "").lower() or "not found" in (out.stderr or "").lower():
            cmd2 = [ARGS.goose, "run", "--name", session_name,
                    "--system", SYSTEM, "-t", user_text]
            out = subprocess.run(cmd2, env=goose_env(), capture_output=True, text=True,
                                 timeout=ARGS.timeout)
            matches = SAY.findall(ANSI.sub("", out.stdout))
    if not matches:
        print("no <SAY> sentinel; stderr tail:", (out.stderr or "")[-300:], file=sys.stderr)
        return None
    return matches[-1].strip()

def main():
    print(f"goose-bridge up: goose={ARGS.goose} model={ARGS.model} "
          f"host={ARGS.openai_host} space={ARGS.space} only_chat={ARGS.only_chat or '(all)'}")
    while True:
        out = sh("notify", "wait", "--state", ARGS.state)
        if out.returncode != 0:
            print("notify wait failed:", out.stderr, file=sys.stderr); break
        try:
            pending = json.loads(out.stdout.strip() or "[]")
        except json.JSONDecodeError:
            print("bad notify json:", out.stdout, file=sys.stderr); continue
        for it in pending:
            if it.get("kind") != "chat":
                sh("notify", "done", "--state", ARGS.state, it["id"]); continue
            chat_id = it.get("chat_id", "")
            space_id = it.get("space_id", ARGS.space)
            if ARGS.only_chat and chat_id != ARGS.only_chat:
                continue
            # Don't eye every inbound message. Let Gemma decide whether the
            # message is for her (the agent returns IGNORE otherwise), and only
            # then claim it with 👀 — mirroring how senior Claude acks only the
            # messages it actually engages with, rather than the wrapper reacting
            # to everything automatically.
            user_text = f'{it.get("creator_name","?")}: {it.get("text","")}'
            session = "chat-" + chat_id          # per-chat memory
            try:
                reply = run_goose(session, user_text)
            except Exception as e:
                print("goose run failed:", e, file=sys.stderr); continue  # leave un-acked -> retry
            if reply and reply.strip().upper() != "IGNORE":
                # Gemma chose to engage: claim the message with 👀, then reply.
                react_eyes(space_id, chat_id, it["id"])
                sh("chat", "send", "--space", space_id, "--chat", chat_id,
                   "--reply-to", it["id"], reply)
                print(f"replied ({len(reply)} chars)")
            else:
                print(f"ignored (reply={reply!r})")
            sh("notify", "done", "--state", ARGS.state, it["id"])

if __name__ == "__main__":
    main()
