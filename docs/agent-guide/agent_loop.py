#!/usr/bin/env python3
"""A minimal, model-agnostic agent loop for acting in an Anytype channel.

It implements the exact behavior the tutorial describes:

    notify wait  ->  for each chat message:
        react 👀            (claim the message)
        decide()           (YOUR model goes here; return text, or None to stay silent)
        chat send --reply-to (reply IN-THREAD)
        notify done         (ack — always, at-least-once)

Only `decide()` is yours to fill in. Everything else is the loop you should not
deviate from. Run your server first, mint an API key, then:

    python3 agent_loop.py --space <id> --chat <id> --key-file <path> \
        --state ~/.anytype-myagent/notify-acked
"""
import argparse, json, os, subprocess, sys, urllib.request

AP = argparse.ArgumentParser()
AP.add_argument("--anytype", default="./dist/anytype")
AP.add_argument("--api", default="http://localhost:31012")
AP.add_argument("--key-file", required=True, help="file holding your REST bearer key (for 👀)")
AP.add_argument("--space", required=True)
AP.add_argument("--chat", default="", help="restrict to one chat id (default: handle all)")
AP.add_argument("--state", default=os.path.expanduser("~/.anytype-myagent/notify-acked"),
                help="YOUR private ack-set file — must be unique per agent")
AP.add_argument("--self-name", default="", help="your display name, to skip your own echoes")
ARGS = AP.parse_args()
KEY = open(ARGS.key_file).read().strip()


def sh(*args):
    return subprocess.run([ARGS.anytype, *args, "--no-update-check"],
                          capture_output=True, text=True)


def react(space_id, chat_id, msg_id, emoji="\U0001f440"):  # 👀
    req = urllib.request.Request(
        f"{ARGS.api}/v1/spaces/{space_id}/chats/{chat_id}/messages/{msg_id}/reactions",
        data=json.dumps({"emoji": emoji}).encode(),
        headers={"Authorization": f"Bearer {KEY}", "Content-Type": "application/json"},
        method="POST")
    try:
        urllib.request.urlopen(req, timeout=8)
    except Exception as e:
        print("react failed:", e, file=sys.stderr)


def decide(msg):
    """msg = one notification dict (creator_name, text, has_mention, chat_name, ...).
    Return a string to reply, or None to stay silent.

    >>> REPLACE THIS with your model call. <<<  The rules to encode in your prompt:
      - Reply only when addressed/mentioned/clearly relevant; otherwise return None.
      - Be concise, plain text (the channel is not markdown).
      - Don't reply to bare acknowledgements from other bots (avoid loops).
    A trivial example: only answer when you're @-mentioned.
    """
    if msg.get("has_mention"):
        return f"Got your message: {msg.get('text','')[:80]!r} — (this is the stub decide(); wire in your model)."
    return None


def main():
    print(f"agent_loop up: space={ARGS.space} chat={ARGS.chat or '(all)'} state={ARGS.state}")
    while True:
        out = sh("notify", "wait", "--state", ARGS.state)
        if out.returncode != 0:
            print("notify wait failed:", out.stderr, file=sys.stderr); break
        try:
            pending = json.loads(out.stdout.strip() or "[]")
        except json.JSONDecodeError:
            print("bad notify json:", out.stdout, file=sys.stderr); continue
        for it in pending:
            mid = it["id"]
            if it.get("kind") != "chat":
                sh("notify", "done", "--state", ARGS.state, mid); continue
            chat_id = it.get("chat_id", "")
            space_id = it.get("space_id", ARGS.space)
            if ARGS.chat and chat_id != ARGS.chat:
                continue                                   # not my chat; leave it un-acked
            if ARGS.self_name and it.get("creator_name") == ARGS.self_name:
                sh("notify", "done", "--state", ARGS.state, mid); continue   # my own echo
            react(space_id, chat_id, mid)                  # 1) 👀 claim
            try:
                reply = decide(it)                         # 2) decide
            except Exception as e:
                print("decide failed:", e, file=sys.stderr); continue   # leave un-acked -> retry
            if reply and reply.strip():                    # 3) reply IN-THREAD
                sh("chat", "send", "--space", space_id, "--chat", chat_id,
                   "--reply-to", mid, reply.strip())
                print(f"replied to {it.get('creator_name')} ({mid})")
            else:
                print(f"ignored {it.get('creator_name')} ({mid})")
            sh("notify", "done", "--state", ARGS.state, mid)  # 4) ack — always


if __name__ == "__main__":
    main()
