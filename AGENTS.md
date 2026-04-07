# Repository Guidelines

## Project Structure & Module Organization

This is a Go monorepo with a frontend client.

- `services/tms/`: main TMS backend (`cmd/`, `internal/`, `pkg/`, `config/`).
- `shared/`: shared Go utilities used by both services.
- `client/`: React + TypeScript app (Vite), UI routes/components in `client/src/`.
- `deploy/`, `docker-compose-local.yml`, `config/`: local infrastructure and deployment configs.
- `legacy/`: older code paths kept for reference/migration.
- `MIGRATION_GUIDE.md`: required playbook for legacy-to-new domain ports.
- `CLAUDE.md`: repository-specific conventions and architecture notes for agent-driven work.

## Build, Test, and Development Commands

Use service-local commands from each module.

- `cd services/tms && task run`: build and start TMS API via CLI.
- `cd services/tms && task test`, `task test-integration`, `task lint`: unit/integration/lint checks.
- `cd client && pnpm dev`: run frontend locally.
- `cd client && pnpm build && pnpm lint`: production build and lint.
- `docker compose -f docker-compose-local.yml up -d`: start local dependencies (Postgres, Redis, MinIO, Meilisearch, etc.).

## Coding Style & Naming Conventions

- **DRY & SOLID**: All code must follow DRY and SOLID principles. Do not duplicate logic — extract and reuse.
- **Uber Go Style Guide**: All Go code must follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).
- Go: format with `gofmt`/`goimports` (and `golines` where configured), lint with `golangci-lint`.
- TypeScript/React: OXFMT + OXLINT are enforced (`client/.oxfmtrc.json` and `client/.oxlintrc.json`).
- Naming: use descriptive domain-based names (`userservice`, `locationcategory`, `formula_template`); keep files/packages lower-case.
- Keep functions small and explicit; avoid introducing `encoding/json` in Go where project lint rules disallow it.
- Bun ORM: use the [docs](docs/bun/) for help.
- **Function signatures**: When a Go function has more than ~3-4 parameters, define a params struct instead of a long parameter list.
- **Utility functions go in `shared/`**: Do NOT place generic utility/helper functions inside domain, service, or handler files. The `shared/` directory (`shared/stringutils`, `shared/sliceutils`, `shared/intutils`, etc.) is the home for all reusable utilities. Create new sub-packages there if needed.
- **Performance**: Write efficient, allocation-conscious code. Preallocate slices/maps when sizes are known. Avoid unnecessary copies. Use appropriate data structures for the problem.
- **Code quality**: Prefer clarity and correctness. No dead code, no unused imports, no TODO placeholders left behind. Every function should have a single clear responsibility.

## Testing Guidelines

- Go tests use the standard `testing` package; many files follow `*_test.go`.
- Integration tests are tag-gated: run with `-tags=integration`.
- CI runs `go test -race` with coverage for `shared` and `services/tms`.
- No fixed coverage threshold is enforced in CI today; keep or improve coverage for touched areas.
- Frontend currently emphasizes `pnpm lint` and `pnpm build` as the baseline quality gate.

## Commit & Pull Request Guidelines

- Prefer concise, imperative commit messages with optional type prefixes, e.g. `fix: prevent goroutine leaks` or `add: fiscal period validation`.
- Keep commits focused; separate refactors from behavior changes.
- PRs should include: what changed and why, linked issue(s), test evidence (commands run/results), screenshots for `client/` UI changes, and migration/config notes.

## Agent-Specific Notes

- Treat `CLAUDE.md` as the operational supplement to this guide, especially for TMS architecture boundaries and Bun/validation conventions.

## Agent Self-Review

- After implementing changes, always review the work before concluding.
- Review for: bugs, regressions, security issues, unnecessary complexity, DRY violations, and SOLID/hexagonal architecture drift.
- For Go code, verify it follows Go best practices and the Uber Go Style Guide.
- Remove defensive nil checks for dependencies or request fields that are guaranteed by dependency injection, validation, or route binding unless a nil value is a real runtime possibility.
- Verify no utility functions were placed in domain/service/handler files — they belong in `shared/`.
- Verify function signatures are not bloated — use params structs for 4+ parameters.
- Check for performance: preallocated collections, no unnecessary allocations, efficient algorithms.
- If issues are found during self-review, fix them and re-run the relevant tests before finishing.

## Code Quality Non-Negotiables

- **No "v1" or placeholder code**: This is an enterprise TMS application. Implement features fully and completely on the first pass. No stubs, no TODO comments, no "we can improve this later" shortcuts, no simplified versions of what was asked. Handle all edge cases, error states, validation, and integration points. If the feature is complex, that's fine — implement the full complexity.
- Never deviate from the established code style in the repository. Match the patterns already in use.
- Do not introduce new dependencies without justification.
- Do not leave commented-out code, TODO comments, or placeholder implementations.
- Prefer the simplest correct solution. Do not over-engineer.
- All error paths must be handled explicitly — no swallowed errors.
