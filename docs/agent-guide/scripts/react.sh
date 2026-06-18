#!/usr/bin/env bash
# 👀-react to a message (claim it). Usage: react.sh <message_id> [emoji]
source "$(dirname "$0")/../lib/common.sh"
[ $# -ge 1 ] || { echo "usage: react.sh <message_id> [emoji]" >&2; exit 1; }
at_react "$1" "${2:-👀}"
echo "reacted ${2:-👀} to $1"
