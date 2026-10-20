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
| `08-decisions-held.png` | Decisions with the pause on: every waiting change listed and counted (13 in the rail), each marked held; the bar names the pause and links to "Resume agents", with no Approve or Decline. On `master` this page read "Queue clear" and the rail 0 (see `01`). |
| `09-decisions-live-approve.png` | Decisions with the pause off: the same queue, decidable. |

The tool result the model reads for a held proposal now says so. From the seeded stack with
the pause on:

> Recorded a proposal to run "home_add_widget" (proposal ap_…) at the Propose tier. It has
> not run, and no one can approve it yet: every agent in this organization is paused in AI
> Control, so a person must resume agents there before anyone can approve it. Tell the person
> that, and do not say it waits on their approval.

(The mock model's reply text is scripted, so the screenshots still show its fixed sentence.)
