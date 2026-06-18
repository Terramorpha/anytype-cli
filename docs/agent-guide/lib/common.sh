# Shared helpers for acting in an Anytype channel. Source this from your scripts:
#   source "$(dirname "$0")/../lib/common.sh"
# It loads config.sh (sibling of config.example.sh) and defines at_* functions.
# shellcheck shell=bash

set -euo pipefail

_here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
[ -f "$_here/config.sh" ] && source "$_here/config.sh" || {
  echo "missing $_here/config.sh — copy config.example.sh to config.sh and fill it in" >&2
  exit 1
}

_at() { "$AT_CLI" "$@" --no-update-check; }

# React to a message. Default emoji is the 👀 "I've seen it / I'm on it" claim.
# Usage: at_react <message_id> [emoji]
at_react() {
  local msg="$1" emoji="${2:-👀}"
  curl -s -m 8 -X POST \
    -H "Authorization: Bearer $AT_KEY" -H "Content-Type: application/json" \
    --data "{\"emoji\":\"$emoji\"}" \
    "$AT_API/v1/spaces/$AT_SPACE/chats/$AT_CHAT/messages/$msg/reactions" >/dev/null
}

# Reply IN-THREAD to a specific message (the "answer" function). PREFER this.
# Usage: at_reply <message_id> <text>
at_reply() { _at chat send --space "$AT_SPACE" --chat "$AT_CHAT" --reply-to "$1" "$2"; }

# Post a flat message (no thread). Use sparingly — only for standalone announcements.
# Usage: at_send <text>
at_send() { _at chat send --space "$AT_SPACE" --chat "$AT_CHAT" "$1"; }

# Block until at least one notification is pending, then print them as JSON and exit.
# This is push-based + level-triggered: it returns the authoritative pending set.
at_wait() { _at notify wait --state "$AT_STATE"; }

# Peek at pending notifications WITHOUT blocking (JSON array). Does not ack.
at_pending() { _at notify list --state "$AT_STATE"; }

# Acknowledge (consume) one or more messages so they don't resurface. ALWAYS do this
# after you've handled a message. Usage: at_done <id> [id...]
at_done() { _at notify done --state "$AT_STATE" "$@"; }
