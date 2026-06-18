#!/usr/bin/env bash
# Post a FLAT message (no thread). Use only for standalone announcements;
# normally you should reply.sh in-thread instead. Usage: send.sh <text...>
source "$(dirname "$0")/../lib/common.sh"
[ $# -ge 1 ] || { echo "usage: send.sh <text...>" >&2; exit 1; }
at_send "$*"
