#!/usr/bin/env bash
# Reply IN-THREAD to a message. Usage: reply.sh <message_id> <text...>
source "$(dirname "$0")/../lib/common.sh"
[ $# -ge 2 ] || { echo "usage: reply.sh <message_id> <text...>" >&2; exit 1; }
msg="$1"; shift
at_reply "$msg" "$*"
