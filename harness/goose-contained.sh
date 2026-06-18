#!/usr/bin/env bash
# Run `goose` inside a guix container, isolating Gemma's command execution from
# the raw host. Used as a drop-in for the `goose` binary by goose_bridge.py
# (pass --goose <this script>): the bridge plumbing (notify/chat) stays on the
# trusted host; only the goose process — where Gemma's shell tool actually runs
# commands — executes in here.
#
# guix shell -CFNW:
#   -C  --container   isolated mount/pid namespaces; host / and $HOME hidden
#   -F  --emulate-fhs standard /bin:/usr/bin layout (so tools + nested guix resolve)
#   -N  --network     share the host network namespace (reach llama-server on
#                     localhost:8080 and the anytype API)
#   -W  --nesting     make guix itself available INSIDE the container, so Gemma
#                     can `guix shell <pkg> -- ...` to fetch tools on demand
#
# Isolation verified: host $HOME (e.g. ~/.anytype-gemma) is invisible, /gnu/store
# is read-only, writes are confined to the shared sandbox + goose's own dirs.
set -euo pipefail

GOOSE_BIN="${GOOSE_BIN:-$HOME/.local/bin/goose}"
SANDBOX="${GEMMA_SANDBOX:-$HOME/gemma-sandbox}"          # Gemma's writable workdir
CFG="${XDG_CONFIG_HOME:-$HOME/.config-gemma-goose}"      # goose config (developer ext)
DAT="${XDG_DATA_HOME:-$HOME/.local/share-gemma-goose}"   # goose sessions = memory
mkdir -p "$SANDBOX" "$CFG" "$DAT"
cd "$SANDBOX"

# Base toolset baked into the container; Gemma can `guix shell` for anything else.
PKGS=(bash coreutils nss-certs git grep sed gawk findutils which less file curl jq python)

exec guix shell -C -F -N -W \
  --share="$SANDBOX" \
  --share="$CFG" \
  --share="$DAT" \
  --expose="$GOOSE_BIN" \
  --preserve='^(GOOSE_|OPENAI_|XDG_)' \
  "${PKGS[@]}" \
  -- "$GOOSE_BIN" "$@"
