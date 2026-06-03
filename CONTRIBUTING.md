# Contributing to Trenova

Thank you for your interest in Trenova.

Trenova is currently in active development. The source is public for transparency, community review, and feedback, but external code contributions are not generally accepted at this stage. Maintainers may close unsolicited code pull requests so the core team can keep architecture, product direction, and release sequencing consistent.

The best ways to contribute right now are:

- Report reproducible bugs.
- Suggest well-scoped product improvements.
- Point out unclear documentation.
- Share deployment, compatibility, or local development findings.
- Participate constructively in GitHub Discussions or Discord.

## Before Opening an Issue

Please check the existing issues, discussions, README, and [SECURITY.md](SECURITY.md) before opening a new issue.

Do not open public issues for vulnerabilities, leaked credentials, customer data exposure, authentication bypasses, authorization failures, remote code execution, SSRF, or denial of service concerns. Report those privately using [SECURITY.md](SECURITY.md).

## Reporting Bugs

Use the bug report issue form and include:

- The affected area, such as `services/tms`, `shared`, `client`, `deploy`, or local infrastructure.
- The exact behavior you expected.
- The behavior you observed.
- Minimal steps to reproduce the issue.
- Relevant versions, environment details, logs, screenshots, or request examples.
- Whether the issue reproduces on the latest development branch.

Good bug reports make it possible for maintainers to reproduce the issue without guessing about local configuration.

## Suggesting Features

Use the feature request issue form and describe:

- The operational problem or workflow gap.
- The users or roles affected.
- The current workaround, if one exists.
- The outcome that would make the feature successful.
- Any compliance, security, performance, integration, or data model constraints.

Feature requests are more useful when they explain the transportation management workflow rather than only describing a preferred UI or API shape.

## Pull Requests

External code pull requests are accepted only when requested by a maintainer or when a maintainer has agreed to the scope in advance.

If you are submitting an approved pull request:

- Keep the change focused and reviewable.
- Link the related issue or maintainer discussion.
- Follow the repository guidance in [AGENTS.md](AGENTS.md) and [CLAUDE.md](CLAUDE.md).
- Match the existing architecture and naming conventions.
- Include relevant tests or explain why tests are not applicable.
- Update documentation, examples, migrations, or configuration when behavior changes.
- Do not include unrelated formatting, refactors, dependency updates, or generated files.

## Development Commands

Run commands from the relevant module.

For the TMS service:

```bash
cd services/tms
task test
task lint
```

For the frontend client:

```bash
cd client
pnpm build
pnpm lint
```

For local infrastructure:

```bash
docker compose -f docker-compose-local.yml up -d
```

Use narrower test commands when a full suite is not needed, but include the commands you ran in the pull request description.

## Code Standards

All changes must follow the repository's existing style and architecture.

- Go code follows the Uber Go Style Guide, `gofmt`, `goimports`, and configured lint rules.
- TypeScript and React code follows the OxFmt and OxLint configuration in `client/`.
- Shared utilities belong in `shared/`, not in domain, service, or handler files.
- Function signatures with several parameters should use named parameter structs.
- Error paths must be handled explicitly.
- New dependencies require clear justification.
- Placeholder code, TODO comments, and incomplete implementations are not acceptable.

## Community Standards

Participation in this repository is covered by the [Code of Conduct](CODE_OF_CONDUCT.md).
