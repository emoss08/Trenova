# Fix agents: shared brief

You are one of several engineers fixing findings from a security and bug review of Trenova
(a transportation management system: Go services in `services/tms`, React client in
`client/`). The review's write-up is `REVIEW/report.md` in your checkout: read the rows for
your findings there for context. Each finding names a location at the review's baseline
commit (`36d426b9`); the code may have moved a little since, so find it by name.

## Your goal

Fix every finding in your brief at its root cause, with a regression test, so the change
can merge into one pull request alongside the other agents' work. The lead engineer will
review your diff line by line, so write it to survive a careful review.

## Before you write code

0. Make sure your checkout is based on commit `8d4e27d1` (it holds the earlier review
   fixes). Run `git merge-base --is-ancestor 8d4e27d1 HEAD && echo ok`. If that does not
   print ok and you have no commits yet, run `git reset --hard 8d4e27d1`; if you have
   commits, `git rebase 8d4e27d1`.
1. Read `AGENTS.md` and `CLAUDE.md` at the repository root. They are binding.
2. Go work: read `.claude/skills/trenova-backend/SKILL.md` and the pattern files it points to
   for the layers you touch (`docs/engineering/backend/*.md`, always `scale.md`).
   Client work: read `.claude/skills/trenova-frontend/SKILL.md` and `client/apps/web/CLAUDE.md`.
3. Read the callers, callees and existing tests of anything you change before changing it.

Key repo rules (full text in AGENTS.md): no comments in Go except a keel model's doc
comments; `sonic`, never `encoding/json`; `errortypes` for errors a caller sees; repository
methods run in `dbtx.Read`/`dbtx.Write` and query through `Keel(ctx)`; columns come from the
generated keel tables; new user-facing strings go through i18n; no backwards-compatibility
shims (change every caller); use a maintained package already in `go.mod` before
hand-rolling a mechanism.

## What "done" means for each finding

A finding is done only when all of these hold:

- The defect is fixed at its root cause, on every path that has it (search for sibling
  paths with the same bug; a fix on one route of three is not done).
- A regression test names the behaviour it protects and asserts exact output. You have
  shown it fails without the fix and passes with it: temporarily revert the fix, run the
  test, see it fail, restore. Repository code gets an integration test when the bug lives
  in SQL.
- No test was deleted, skipped, or loosened to pass, and no lint suppression was added. If a
  suppression is truly needed, justify it inline and list it in your report.
- The packages you touched pass their tests, `go vet`, and golangci-lint for new issues
  (commands below). Generated artifacts your change affects are regenerated and committed.
- The work is committed in small logical commits (one finding or one kind of fix each).

If a finding turns out to be a product decision (two reasonable behaviours and nothing in
the code or docs picks one), a schema redesign far wider than the finding, or something
you cannot fix safely, do not guess. Leave it unfixed and report what you found, the design
you recommend and what it would touch. That is an acceptable outcome; a guessed fix is not.

If you find another bug or security issue in passing: fix it if it is in your area and
small, with its own test and commit; otherwise report it with file, line and impact.

## Environment rules (this machine is shared with other agents)

- 4 CPUs, 15 GB RAM, limited disk. Never run `go test ./...`, `go build ./...` or
  golangci-lint over the whole module. Test only the packages you changed and their direct
  dependents. Always use `-p 1`.
- Run every `go build`, `go test`, `go vet`, `go run`, `go generate` and golangci-lint
  command through the shared build lock:
  `flock /tmp/claude-0/-home-user-Trenova/5435a298-3ef8-5a44-8dc3-65e470cf197e/scratchpad/fix-agents/go.lock <command>`.
  Only one Go build runs at a time on this machine, and the lead trims the build cache only
  while holding that lock. A build outside the lock can fail with missing cache files. Keep
  each locked command short: test a few packages, not a long list.
- Always `export GOFLAGS="-tags=nomsgpack -trimpath"` before any go command (the trimpath
  lets the build cache be shared between agents' checkouts).
- Never delete or clean `/root/.cache/go-build`, `/tmp/go-build*`, or anything outside your
  checkout and your scratch dir. If `df -h /` shows under 3 GB free, stop building and say
  so in your report.
- Do not push, do not create branches other than the one you are on, do not open a PR.
  Commit in your checkout; the lead merges your branch.
- Never run a formatter over a directory; format the files you changed, by path.
- Never run mockery over the codebase. Mockery is not installed: edit the affected mock by
  hand to match the interface exactly (follow the generated style of the existing file).

Tools (absolute paths):

- golangci-lint v2.12.2: `/tmp/claude-0/-home-user-Trenova/5435a298-3ef8-5a44-8dc3-65e470cf197e/scratchpad/gl/golangci-lint`
  Run from `services/tms`: `golangci-lint run --new-from-rev=<your branch's starting commit> ./path/to/pkg/...`
  for each package you changed, and `golangci-lint fmt <files>` to apply the repo's
  golines/goimports formatting to files you changed.
- goimports, golines: `/tmp/claude-0/-home-user-Trenova/5435a298-3ef8-5a44-8dc3-65e470cf197e/scratchpad/bin/`
- Node 26 and pnpm: prefix `PATH` with `/tmp/claude-0/-home-user-Trenova/5435a298-3ef8-5a44-8dc3-65e470cf197e/scratchpad/node26/bin`.
- Your checkout has no `node_modules`. For the i18n tool, run once from your checkout root:
  `ln -s /home/user/Trenova/node_modules node_modules` (and for client work also
  `ln -s /home/user/Trenova/client/node_modules client/node_modules`, plus the same for each
  `client/apps/*/node_modules` and `client/packages/*/node_modules` that exists there).
  These links are git-ignored; never commit them.
- Integration tests (Postgres 18 with PostGIS, no pgvector; Redis 8) are already running:
  ```
  export TRENOVA_TEST_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/trenova_test?sslmode=disable"
  export TRENOVA_TEST_REDIS_ADDR=localhost:6379
  export TRENOVA_TEST_EXPECT_VECTOR=false
  go test -p 1 -tags=integration,nomsgpack -trimpath ./path/to/pkg/...
  ```
  Docker Hub rate-limits pulls; if a test needs another image, pull it as
  `mirror.gcr.io/library/<name>:<tag>` (or `mirror.gcr.io/<org>/<name>`) and `docker tag` it.
- Network: package registries and GitHub work; `vuln.go.dev` is blocked.

Generated artifacts (see `docs/engineering/generated-artifacts.md`): GraphQL schema change →
`go run ./internal/api/graphql/gqlexec/gen` and `go run ./internal/api/graphql/agentread/gen`
in services/tms, then `pnpm --filter @trenova/graphql codegen` in client; migration →
`task db-create-migration` conventions and `go run ./cmd/db migrate stamp`; keel model
change → `go tool keel gen` (flags in `.github/workflows/test-tms.yml`); write route or
mutation added/removed → `go generate ./internal/api/writecoverage/...`; agent tool or
prompt change → the agentdefinition prompt snapshots and agentevalgate catalog snapshot
(`-update`); user-facing text → i18n below.

i18n: the English string is the key. After adding or changing strings run, from the
checkout root, `node i18n/tools/i18n.mjs sync`, write translations for every new key for
`es`, `zh-CN` and `zh-TW` into batch files, `node i18n/tools/i18n.mjs merge <locale>
<batch.json>` for each, then `node i18n/tools/i18n.mjs emit` and `node i18n/tools/i18n.mjs
check` (must pass). Strings must be literal arguments inside functions to be extracted.
Save your three batch files under
`/tmp/claude-0/-home-user-Trenova/5435a298-3ef8-5a44-8dc3-65e470cf197e/scratchpad/fix-agents/i18n/<your-area>/`
(the lead re-merges them if catalogs conflict between agents).

Commit messages: concise imperative with an area prefix (`documents: …`), a body saying
what was wrong and what changed, and these exact trailer lines at the end:

```
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01L7Gd5v8t2y7WHswGwtgbrn
```

No model names anywhere else in commits or code.

## Your report back

End with a report in this shape (it is what the lead reviews against):

1. **Branch and commits**: branch name, checkout path, and each commit hash with its subject.
2. **Per finding**: ID, status (fixed / partly fixed / not fixed), what was wrong (one or
   two sentences), what you changed (files), the regression test name(s), and the
   fail-before / pass-after evidence (the command and the failing assertion you saw).
3. **Behaviour changes** a reviewer must know: API shape, permissions or auth behaviour, DB
   schema or migrations, config keys, user-facing text.
4. **Other bugs found**: fixed (with commit) or reported (file:line, impact).
5. **Checks run**: each command and its result (tests, vet, lint, generators, i18n check).
6. **Not done / needs a decision**: what, why, and your recommended design.
7. **i18n batch path** if you added strings.

Be exact. Do not report a check as passing unless you ran it and saw it pass.
