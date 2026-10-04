# Desk end-to-end tests

Playwright tests for the Desk's core flows, driven by a scripted mock model so
every run gets the same replies, tool calls and proposals.

| Spec | Flow |
| --- | --- |
| `desk-conversation.spec.ts` | Ask from the home composer: the question, the reply streaming, the saved reply. Stop a streaming reply. |
| `desk-artifacts.spec.ts` | A reply that names an artifact opens it from the badge in its sentence; a lookup the reply did not name is never drawn, not even for a frame. |
| `desk-approvals.spec.ts` | A proposed change waits on the decision card and is approved. An email draft is edited, sent for approval and approved as modified. |
| `desk-shell.spec.ts` | The search palette (⌘K / Ctrl+K) and the settings dialog open and close from the keyboard. |

The tests do not start anything. They expect the stack below to be up.

## 1. The mock model

`mockllm/mock.py` is an OpenAI-compatible server (Python 3, no packages). It
answers from `mockllm/script.json`: the first rule whose `match` phrase is in
the latest user message calls its `tools` one per request and then streams its
`text`. Each test's question carries the tag its rule matches (`[e2e-basic]`,
`[e2e-stop]`, …).

```sh
cd client/apps/web/e2e/mockllm
MOCK_WORD_DELAY=0.05 python3 mock.py 5055
```

`MOCK_SCRIPT` points it at another rules file and `MOCK_LOG_DIR` says where
`requests.jsonl` and `offered.log` (the tools offered on each request) go. The
stop test needs replies to stream slowly enough to be stopped; keep the word
delay at 0.03 s or more.

In AI Control, point the providers the Desk uses at the mock: an
OpenAI-compatible provider with base URL `http://localhost:5055/v1` and any
model name, enabled for assistant chat and scope classification, with private
network access allowed. The mock answers the scope guard's structured-output
call itself.

## 2. The stack

From a development database seeded with `development` seeds (the agents
"Dispatch desk" and "Billing exceptions", the SEED-SHP shipments, John Smith):

```sh
# services/tms
./build/trenova-cli api run
./build/trenova-cli worker run
# client
pnpm --filter @trenova/web dev        # http://localhost:5173
```

## 3. The tests

```sh
cd client/apps/web
E2E_CHROMIUM_PATH=/opt/pw-browsers/chromium pnpm test:e2e
pnpm test:e2e desk-shell              # one spec
pnpm test:e2e --headed --debug        # step through
```

| Variable | Default | |
| --- | --- | --- |
| `E2E_BASE_URL` | `http://localhost:5173` | The web app. |
| `E2E_EMAIL` / `E2E_PASSWORD` | the development seed's admin | Who signs in. Global setup signs in once and saves the session to `e2e/.auth/user.json`. |
| `E2E_STORAGE_STATE` | | A saved, signed-in storage state to use instead of signing in. |
| `E2E_CHROMIUM_PATH` | | A Chromium binary to launch. Without it Playwright uses its own browser, which `playwright install chromium` downloads. |
| `E2E_COMMIT` | | `1` lets approvals go through. By default each approval is undone inside its undo window, so the seeded records stay as they were and the tests can run again. |

Reports land in `e2e/.report`, traces of failures in `e2e/.results`
(`pnpm exec playwright show-trace <trace.zip>`).

## When a test fails

- **The approval card says "would be refused" or "out of date".** The record
  the rule writes to has moved on (a run with `E2E_COMMIT=1` marks SEED-SHP-002's
  move In Transit for good). Reseed, or point the `e2e-approval` rule at a move
  that is still New.
- **The email draft is refused.** The draft needs an email profile: the rule
  calls `list_email_profiles` and passes the first `emlprof_` id it returns
  (`"{{id:emlprof_}}"`). With no profile, or a sender that cannot send, the
  preview refuses the draft.
- **A tool step never happens.** The runtime offers an agent only some of its
  tools per turn; each rule asks `find_tools` first, and the mock skips any
  step whose tool is still not offered. `offered.log` shows what each request
  offered.
- **The ids in `script.json` do not exist.** They are the development seed's
  records in one database. Look them up in yours and update the rules.
