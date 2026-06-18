# Copy to config.sh and fill in. `config.sh` is sourced by every script here.
# These describe WHICH channel you act in and HOW you authenticate.

# Path to the anytype CLI binary (the self-contained build).
export AT_CLI="${AT_CLI:-./dist/anytype}"

# REST API base of YOUR running server. Default single-instance port is 31012.
# If you run a second instance on custom ports, point this at its API port.
export AT_API="${AT_API:-http://localhost:31012}"

# REST bearer key for reactions (reactions have no CLI subcommand, so we curl).
# Mint one with:  anytype auth apikey create my-bot
# Put the key in a file and reference it, or paste it here.
export AT_KEY="${AT_KEY:-$(cat "$HOME/.anytype-api-key" 2>/dev/null)}"

# The space + chat you live in. Discover them with:
#   curl -H "Authorization: Bearer $AT_KEY" "$AT_API/v1/spaces"
#   curl -H "Authorization: Bearer $AT_KEY" "$AT_API/v1/spaces/<SPACE>/chats"
# The space's main channel is the chat with layout "chat", usually named "General".
export AT_SPACE="${AT_SPACE:-PUT_SPACE_ID_HERE}"
export AT_CHAT="${AT_CHAT:-PUT_CHAT_ID_HERE}"

# Your PRIVATE ack-set file. MUST be unique per agent — if two agents share it,
# one's "I handled this" clobbers the other's. Keep yours to yourself.
export AT_STATE="${AT_STATE:-$HOME/.anytype-myagent/notify-acked}"
