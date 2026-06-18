#!/usr/bin/env python3
"""MVP harness: let a local LLM (llama-server, OpenAI-compatible) operate in an
Anytype chat through the anytype-cli primitives.

Loop (Event -> Action, single action type for the MVP = reply):
  anytype notify wait   ->  pending notifications (JSON, drop-free)
  for each item:
    react 👀            (claim signal; mechanics live in the harness)
    ask the model, FORCED via json_schema to emit {"reasoning", "reply"}
    anytype chat send --reply-to <id>   (post the reply, threaded)
    anytype notify done <id>            (ack; at-least-once)

The schema is a one-variant stub. Growing the protocol = adding variants to a
tagged union (reply / create_object / delegate / defer / noop, ...) and a
dispatch arm per variant. Keep it flat for llama.cpp's json-schema->GBNF.

Run: start the model first (`run-llm`), then:
  python3 harness/agent_mvp.py --space <id> [--only-chat <id>]
Ideally authenticate the `anytype` CLI as the local LLM's OWN account so it
posts as a distinct member.
"""
import argparse, json, os, subprocess, sys, urllib.request

AP = argparse.ArgumentParser()
AP.add_argument("--anytype", default="./dist/anytype", help="path to the anytype CLI")
AP.add_argument("--model-url", default="http://127.0.0.1:8080/v1/chat/completions")
AP.add_argument("--model", default="gemma-4-12b")
AP.add_argument("--key-file", default="/tmp/at-key.txt", help="REST bearer key (for the 👀 reaction)")
AP.add_argument("--api", default="http://localhost:31012")
AP.add_argument("--space", required=True)
AP.add_argument("--only-chat", default="", help="restrict to one chat id (else handle all)")
AP.add_argument("--state", default=os.path.expanduser("~/.anytype-gemma/notify-acked"),
                help="ack-set file (MUST differ from any other agent's, e.g. the default account's)")
AP.add_argument("--max-tokens", type=int, default=400)
ARGS = AP.parse_args()
KEY = open(ARGS.key_file).read().strip()

SYSTEM = (
    "You are Gemma, an AI assistant in a SHARED Anytype channel with Terramorpha, Claude, "
    "and others. You receive each new chat message. Decide whether to respond. Reply ONLY when "
    "the message is addressed to you, mentions you, asks a question you should answer, or is "
    "clearly relevant to you — otherwise IGNORE it. The channel is shared and chatty; do not "
    "reply to everything. Answer as JSON matching the schema: 'reasoning' (brief private thinking), "
    "then 'action': either {\"type\":\"reply\",\"text\":\"...\"} to post a message, or "
    "{\"type\":\"ignore\"} to stay silent."
)

# Tagged-union action schema. Grow the protocol by adding variants (create_object,
# delegate, react, ...). Keep it flat + discriminated for llama.cpp's GBNF.
SCHEMA = {
    "type": "object",
    "properties": {
        "reasoning": {"type": "string"},
        "action": {
            "oneOf": [
                {"type": "object",
                 "properties": {"type": {"const": "reply"}, "text": {"type": "string"}},
                 "required": ["type", "text"], "additionalProperties": False},
                {"type": "object",
                 "properties": {"type": {"const": "ignore"}},
                 "required": ["type"], "additionalProperties": False},
            ]
        },
    },
    "required": ["reasoning", "action"],
    "additionalProperties": False,
}

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

def infer(user_text):
    body = {
        "model": ARGS.model,
        "messages": [{"role": "system", "content": SYSTEM},
                     {"role": "user", "content": user_text}],
        "response_format": {"type": "json_schema",
                            "json_schema": {"name": "action", "schema": SCHEMA, "strict": True}},
        "temperature": 0.4,
        "max_tokens": ARGS.max_tokens,
    }
    req = urllib.request.Request(ARGS.model_url, data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"}, method="POST")
    resp = json.loads(urllib.request.urlopen(req, timeout=120).read())
    content = resp["choices"][0]["message"]["content"]
    obj = json.loads(content)  # schema-forced, so this parses
    return obj.get("action", {}), obj.get("reasoning", "")

def main():
    print(f"harness up: model={ARGS.model_url} space={ARGS.space} only_chat={ARGS.only_chat or '(all)'}")
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
                sh("notify", "done", "--state", ARGS.state, it["id"]); continue   # MVP: only chat messages
            chat_id = it.get("chat_id", "")
            space_id = it.get("space_id", ARGS.space)
            if ARGS.only_chat and chat_id != ARGS.only_chat:
                continue   # leave for someone else; don't ack
            react_eyes(space_id, chat_id, it["id"])
            user_text = f'[{it.get("chat_name","")}] {it.get("creator_name","?")}: {it.get("text","")}'
            try:
                action, reasoning = infer(user_text)
            except Exception as e:
                print("infer failed:", e, file=sys.stderr); continue  # leave un-acked -> retry
            kind = action.get("type")
            print(f"action={kind} reasoning={reasoning[:100]}")
            if kind == "reply":
                sh("chat", "send", "--space", space_id, "--chat", chat_id, "--reply-to", it["id"], action.get("text", ""))
            # "ignore" (or unknown): stay silent. Either way, consume it.
            sh("notify", "done", "--state", ARGS.state, it["id"])

if __name__ == "__main__":
    main()
