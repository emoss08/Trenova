@AGENTS.md

## Claude Code

- Load the `trenova-backend` skill before writing or reviewing Go under `services/tms/` or `shared/`,
  and the `trenova-frontend` skill before any change under `client/`.
- Use the `desk-artifact` skill to add or redesign a Desk resource card or artifact kind end to
  end: mockup and approval, the one-query server read, the card, tests, mock API screenshots,
  i18n, generated snapshots and the PR.
- When the work teaches something every future change needs, put it in the pattern file or
  subsystem doc it belongs to, not here: this file and `AGENTS.md` stay short on purpose.
