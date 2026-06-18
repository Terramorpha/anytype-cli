#!/usr/bin/env bash
# The core interaction loop. This is HOW to behave in the channel:
#
#   wait for messages  ->  for each one:  👀 (claim)  ->  decide  ->  reply in-thread  ->  ack
#
# The "decide" step is delegated to an external command ($DECIDE_CMD) so this loop
# is model-agnostic. That command receives ONE message as JSON on stdin and prints
# the reply text on stdout — or prints nothing / the literal "IGNORE" to stay silent.
# Plug your LLM in there (see ../agent_loop.py for a Python version with a model call).
#
# Usage:  DECIDE_CMD=./my-decider.sh ./watch-loop.sh
# Default decider stays silent (claims + acks but never replies) so it's safe to run.

source "$(dirname "$0")/../lib/common.sh"

DECIDE_CMD="${DECIDE_CMD:-}"            # e.g. ./decide.sh  or  python3 decide.py
SELF_NAME="${SELF_NAME:-}"             # optional: your own display name, to skip your echoes

decide() {                             # stdin: message JSON -> stdout: reply text or IGNORE
  if [ -n "$DECIDE_CMD" ]; then $DECIDE_CMD; else echo IGNORE; fi
}

echo "watch-loop up: space=$AT_SPACE chat=$AT_CHAT state=$AT_STATE decider=${DECIDE_CMD:-<silent>}"
while true; do
  pending="$(at_wait)" || { echo "notify wait failed" >&2; sleep 2; continue; }
  echo "$pending" | jq -c '.[]' 2>/dev/null | while read -r item; do
    kind=$(jq -r '.kind // "chat"' <<<"$item")
    id=$(jq -r '.id' <<<"$item")
    [ "$kind" = "chat" ] || { at_done "$id" >/dev/null; continue; }   # consume non-chat
    who=$(jq -r '.creator_name // "?"' <<<"$item")
    [ -n "$SELF_NAME" ] && [ "$who" = "$SELF_NAME" ] && { at_done "$id" >/dev/null; continue; }

    at_react "$id"                                # 1) 👀 claim it FIRST, always
    reply="$(decide <<<"$item")"                  # 2) decide what (if anything) to say
    reply="$(printf '%s' "$reply" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    if [ -n "$reply" ] && [ "$reply" != "IGNORE" ]; then
      at_reply "$id" "$reply" >/dev/null          # 3) reply IN-THREAD
      echo "replied to $who ($id)"
    else
      echo "ignored $who ($id)"
    fi
    at_done "$id" >/dev/null                       # 4) ACK — always, handled or ignored
  done
done
