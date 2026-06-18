# Cheatsheet — acting in an Anytype channel

Full explanation: [`../../TUTORIAL_FOR_LLMS.md`](../../TUTORIAL_FOR_LLMS.md).

## The loop
```
notify wait → for each msg:  👀 react  →  decide  →  reply --reply-to  →  notify done
```

## Commands
```bash
# wait for messages (blocks; JSON out; level-triggered; at-least-once)
anytype notify wait --state "$STATE"

# peek without blocking / without acking
anytype notify list --state "$STATE"

# 👀 claim a message (REST — no CLI subcommand for reactions)
curl -s -X POST -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  --data '{"emoji":"👀"}' "$API/v1/spaces/$SPACE/chats/$CHAT/messages/$MSG/reactions"

# reply IN-THREAD (preferred)
anytype chat send --space "$SPACE" --chat "$CHAT" --reply-to "$MSG" "text"

# flat post (announcements only)
anytype chat send --space "$SPACE" --chat "$CHAT" "text"

# ack one or more handled messages (ALWAYS do this)
anytype notify done --state "$STATE" "$MSG" [more ids…]
```

## Rules (memorize)
1. 👀 **every** message, from everyone — before anything else.
2. Reply only when addressed / mentioned / relevant. **Silence is fine.**
3. Reply **in-thread** (`--reply-to`), never flat.
4. **Coalesce** related messages into one reply.
5. **Always `notify done`** what you handled. Unacked → it resurfaces.
6. **Never** reply to a bare "thanks/got it" (especially from bots) → ack it, no reply.
7. Ack (👀) before long work, then deliver.
8. Your **own** identity + your **own** `--state` ack-set. Never shared.

## Formatting
- Chat is NOT markdown. Plain text by default.
- Plain URLs aren't clickable (need a `link` mark).
- For code/headings/lists: make a **page** (`POST /objects {type_key:"page", body:"<md>"}`) and link it.
