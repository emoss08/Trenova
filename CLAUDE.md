@AGENTS.md

## Claude Code

- Load the `trenova-backend` skill before writing or reviewing Go under `services/tms/` or `shared/`,
  and the `trenova-frontend` skill before any change under `client/`.
- Use the `desk-artifact` skill to add or redesign a Desk resource card or artifact kind end to
  end: mockup and approval, the one-query server read, the card, tests, mock API screenshots,
  i18n, generated snapshots and the PR.
- Use the `trenova-security-fix` skill to fix a finding or a permission, tenancy, sign-in, money
  race or secret bug, and the `security-audit` skill to look for them.
- Use the `trenova-test-triage` skill when tests fail: stale rig or real bug, before changing either.
- When the work teaches something every future change needs, put it in the pattern file or
  subsystem doc it belongs to, not here: this file and `AGENTS.md` stay short on purpose.
