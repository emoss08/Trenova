# Held proposals in the Desk's approval box

Screenshots from a seeded development database (`trenova db seed --env development`), the API
and worker built from this branch, and the scripted mock model (`client/apps/web/e2e/mockllm`),
taken in headless Chromium. The agent called `home_add_widget`, a plan of two
`home_add_widget` calls, and `shipment_request_documents`.

| File | What it shows |
| --- | --- |
| `01-before-desk-paused-no-card.png` | `master`: the organization's agents are paused (the default), the reply says the change waits on approval, and the Desk docks nothing. |
| `02-after-desk-held-card.png` | The same thread with the fix: the box names the pause and links to "Resume agents"; there is no Approve and no ⌘↵ hint. |
| `03-after-ai-control-panel-held-card.png` | The same thread in the side panel on AI control, beside the overview that reads "16 proposals are held until you resume". |
| `04-after-resume-panel-approve.png` | After "Resume agents" on the overview, the panel's box takes its Approve button without a reload. |
| `05-desk-live-approve.png` | With the pause off, a new turn's proposal docks with Approve. |
| `06-desk-held-plan.png` | A held plan of two steps. |
| `07-desk-held-email.png` | A held email draft, with the draft open in the workspace. |
