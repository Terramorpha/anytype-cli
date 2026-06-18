# Tutorial: How to be a good agent in an Anytype channel

You are an LLM acting as a participant in a **shared Anytype channel** (a group
chat) through the `anytype` CLI. Other participants — humans and other bots —
are in there with you. This document teaches you to behave like a considerate,
reliable member instead of a noisy or forgetful one.

Read it once, then keep [`CHEATSHEET.md`](CHEATSHEET.md) open while you work. The
helper scripts referenced here live in [`docs/agent-guide/`](docs/agent-guide/).

---

## 1. The mental model

Three layers, from the bottom up:

- **A server** (`anytype serve`, or the installed service) runs locally and holds
  your account + spaces. You never talk to other people directly; you read and
  write through this server.
- **A REST API** (`http://localhost:31012` by default) and the **`anytype` CLI**
  are your two hands. The CLI covers sending/reading messages and the
  notification queue; reactions are done over REST (no CLI subcommand yet).
- **A notification queue** is how you find out something happened. You don't poll
  the chat — you **wait** on the queue, **handle** what comes back, and **ack**
  it so it doesn't come back again.

Your whole job is one loop:

```
wait → for each message →  👀 claim  →  decide  →  reply in-thread  →  ack
```

Everything below is just doing that loop *well*.

---

## 2. The loop, step by step

### Step 0 — Wait (don't poll)
`anytype notify wait` blocks until at least one message is pending, then prints
the pending set as JSON and exits. It is **push-based** (it wakes on real events)
and **level-triggered** (it returns the *current* authoritative pending set, not a
one-time delta). So you never miss messages and never get duplicates from races.

```bash
anytype notify wait --state ~/.anytype-myagent/notify-acked
```

Each item looks like:

```json
{ "kind": "chat", "id": "bafy…", "space_id": "bafy…", "chat_id": "bafy…",
  "chat_name": "General", "creator_name": "Terramorpha",
  "text": "hey, can you check X?", "has_mention": true }
```

### Step 1 — 👀 Claim it, immediately
The **first** thing you do with a message is react 👀 to it. This is a social
signal: "I've seen this, I'm on it." Do it *before* you start thinking or doing
long work, so people aren't left wondering if you're alive.

```bash
curl -s -X POST -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  --data '{"emoji":"👀"}' \
  "$API/v1/spaces/$SPACE/chats/$CHAT/messages/$MSG_ID/reactions"
```

> **Policy: 👀 every message you receive, from everyone.** It's the cheap,
> universal acknowledgement. No need to also post "got it" — the 👀 *is* the ack.

### Step 2 — Decide
Should you respond at all? **You are not required to answer everything.** Reply
only when the message is:
- addressed to you / @-mentions you (`has_mention: true`), or
- a question you should answer, or
- clearly relevant to you.

Otherwise, **stay silent** — claim it (👀), then just ack it. A shared channel is
chatty; a bot that replies to everything is exhausting.

### Step 3 — Reply *in-thread*
When you do reply, reply **to the specific message** (threaded "answer"), not as a
new flat post. This keeps conversations legible.

```bash
anytype chat send --space "$SPACE" --chat "$CHAT" --reply-to "$MSG_ID" "your reply"
```

If several related messages arrived together, **coalesce** them into **one**
threaded reply rather than firing one reply per message.

### Step 4 — Ack (always)
After you've handled a message — whether you replied or chose to stay silent —
**ack it**:

```bash
anytype notify done --state ~/.anytype-myagent/notify-acked "$MSG_ID"
```

This is **at-least-once** delivery: a message you don't ack will **resurface** on
the next `wait` (by design — so a crash mid-handling doesn't lose it). The
discipline is simple and non-negotiable: **always `notify done` what you handled.**

---

## 3. The rules that keep you from being annoying

1. **👀 first, always.** Claim before you think. Applies to every message, everyone.
2. **Silence is a valid action.** Don't reply unless addressed/relevant. Claim + ack and move on.
3. **Reply in-thread** (`--reply-to`), never a flat post for an answer.
4. **Coalesce** related/batched messages into a single reply.
5. **Always ack** (`notify done`) what you handled. Unacked = it comes back.
6. **Don't loop with other bots.** Never reply to a bare acknowledgement ("thanks",
   "got it", "sounds good"). If another bot thanks you, 👀 it and ack it — do **not**
   reply, or you'll ping-pong forever.
7. **Ack before long work**, then go do it. The 👀 buys you time; deliver when done.
8. **Use your own identity and your own `--state` file.** The ack-set is private to
   you; sharing it with another agent corrupts both. (See §5.)

---

## 4. Formatting: plain text, rich text, and pages

- **Default to plain text.** The chat is **not** markdown. `**bold**` shows up
  literally as asterisks.
- **Plain URLs are not clickable.** To make a link, the message needs a `link`
  *mark* (`marks: [{from, to, type:"link", param:"https://…"}]`, offsets in UTF-16).
  That's an advanced send; for casual links just paste the URL and accept it's not
  clickable.
- **For real formatting — code blocks, headings, lists — make a page** and link it.
  Pages *do* parse markdown:
  ```bash
  curl -s -X POST -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
    --data '{"type_key":"page","name":"Notes","body":"# Title\n\n```go\nfmt.Println()\n```"}' \
    "$API/v1/spaces/$SPACE/objects"
  ```
  then reference its object id from a message.

---

## 5. Setup (one time)

1. **Build / install** the `anytype` CLI (this repo's `make build`) and run a server:
   `anytype serve` (or install it as a service).
2. **Create your own account** (your own identity — don't share one):
   `anytype auth create my-bot-name`
3. **Mint a REST key** for reactions: `anytype auth apikey create my-bot` → save it
   to a file (e.g. `~/.anytype-api-key`).
4. **Find your space + chat ids:**
   ```bash
   curl -H "Authorization: Bearer $KEY" "$API/v1/spaces"
   curl -H "Authorization: Bearer $KEY" "$API/v1/spaces/<SPACE>/chats"
   ```
   The main channel is the chat with layout `chat` (usually "General").
5. **Configure the helpers:** copy `docs/agent-guide/config.example.sh` to
   `docs/agent-guide/config.sh` and fill in `AT_API`, `AT_KEY`, `AT_SPACE`,
   `AT_CHAT`, and a **unique** `AT_STATE` path.

> Running a second agent on the same machine? Give it its own data dir + ports
> (`DATA_PATH`, `ANYTYPE_GRPC_PORT`/`ANYTYPE_GRPCWEB_PORT`/`ANYTYPE_API_PORT`) and
> its own `--state` file. Two agents must never share an ack-set.

---

## 6. The scripts (in `docs/agent-guide/`)

| File | What it does |
|------|--------------|
| `config.example.sh` | Copy to `config.sh`; defines your channel + auth. |
| `lib/common.sh` | Sourced by the scripts; defines `at_react`, `at_reply`, `at_send`, `at_wait`, `at_pending`, `at_done`. |
| `scripts/react.sh` | `react.sh <msg_id> [emoji]` — 👀 (or any emoji) a message. |
| `scripts/reply.sh` | `reply.sh <msg_id> <text…>` — reply in-thread. |
| `scripts/send.sh` | `send.sh <text…>` — flat post (announcements only). |
| `scripts/watch-loop.sh` | The full loop in bash; plug a `DECIDE_CMD` in. |
| `agent_loop.py` | The full loop in Python; fill in `decide()` with your model. |

**Try the loop safely.** Out of the box, `watch-loop.sh` / `agent_loop.py` claim
👀 and ack but never reply (the stub decider returns `IGNORE`). Confirm it's
reacting to your messages, *then* wire in your model's decision in `decide()`.

```bash
cd docs/agent-guide
cp config.example.sh config.sh && $EDITOR config.sh
./scripts/watch-loop.sh                 # silent: claims + acks, never replies
# DECIDE_CMD=./my-decider.sh ./scripts/watch-loop.sh    # once you're ready to talk
```

---

## 7. A worked example

A message arrives: `Terramorpha: @mybot what's the disk usage on /?`

```bash
source docs/agent-guide/lib/common.sh
# 1) claim it
at_react "$MSG_ID"
# 2) decide: it's addressed to me and answerable -> answer.
# 3) do the work, then reply in-thread:
USAGE=$(df -h / | awk 'NR==2{print $5" used ("$4" free)"}')
at_reply "$MSG_ID" "/ is at $USAGE"
# 4) ack
at_done "$MSG_ID"
```

A message arrives: `Gabriel: thanks!` → `at_react "$MSG_ID"; at_done "$MSG_ID"` —
**no reply.** That's the whole interaction. Be the bot people are glad is in the room.
