# Paused fix work (unreviewed)

The findings that `REVIEW/report.md` lists as not fixed were handed to eight fix agents,
one per area, each in its own worktree based on `8d4e27d1`. The frontend agent finished; its
work was reviewed and is on this branch (`67025eee`..`95ce056b`). The other seven were cut
off by a usage limit partway through, then paused on purpose: `claude/small-packages`
restructures the Go service (9,242 files, more than 1,000 moves), and it lands on master
first. Their work is ported onto that structure afterwards rather than finished on this
one.

**Nothing in this folder has been reviewed.** Each patch is the agent's commits plus a final
`wip:` commit of whatever it had not committed yet when it stopped. Some fixes are complete
with tests, some are half-written, and none has been checked by the lead. Treat each patch
as a record of the approach, not as code to apply.

| Patch | Area | Findings in scope (IDs from report.md §2) | Where it stopped |
|---|---|---|---|
| `documents.patch` | Document access | H8 owner-record check on every read path, select-options authz; upload sessions not bound to their creator; F5 presigned URLs to sealed blobs; F6 empty content type; M21 capture print-job memory; orphan receipt documents | running client codegen after a GraphQL change |
| `integrations.patch` | Outbound HTTP, EDI | H10 SSRF through integration URLs; M20 unbounded `io.ReadAll`; M19 secrets kept on destination change; M14 AS2 unsigned inbound, async MDN, identifier squatting; EDI carrier invoice N+1; API keys logged; webhook token entropy and index; googlemaps route authz | writing an integration test |
| `money.patch` | Accounting, billing, rating | H11 foreign currency posted unconverted; M23 self-approval; M24 reversal of subledger journals; Locked periods accepting subledger postings; M25 idempotency of payment post and PayWorkerNow; M17 detention cap; M22 rate quote writes behind read; remittance `LIKE`; detention stats limit; collections paging | refactoring an invoice helper, worklist integration test |
| `identity.patch` | Auth, users, reports | M13 SSO enforcement on organization switch; SSO state not bound to the browser; forgot-password timing; actor-covers-target on MFA reset, reset email, deactivate, role unassign; last-administrator guard; cross-org memberships; emails in select options; private report definitions; organization cache before RLS | role service changes |
| `platform.patch` | API, realtime, config | H9 realtime events carry entities; M15 request body limit; M16 placeholder secrets and APP_ENV; SSE timeout bypass; client IP from X-Forwarded-For; a REST authz lint; self-scoped GraphQL roots | client `pnpm lint` |
| `agents.patch` | Agent module | M27: runs acting as deactivated users; portal live view behind read; agent definitions beyond the editor's access; proposals skipping private threads; triage taint laundering; watchtower hand-off checks | a test run |
| `operations.patch` | Dispatch, workforce | Compliance Block enforced on assignment with an audited override; M18 timesheet week in UTC; HOS team scope and limits; MyYtdPay; comment paging; future ClockOut; tracking links after the move closes | dispatch preflight dialog |

`AGENT-BRIEF.md` is the brief every agent worked to: the definition of done (root-cause fix
on every path, a regression test shown to fail before and pass after, generated artifacts,
small commits), the repository rules, and the report format. `i18n/<area>/` holds the
translation batches the agents wrote for new strings.

## Porting plan

Once `claude/small-packages` is on master:

1. Branch from master.
2. Port this branch's merged fixes (report.md §2 and §3, each with its commit) one at a
   time. Port the regression test first, watch it fail on the new structure, then port the
   fix.
3. Redo each area above against the new structure, using its patch and the brief as the
   spec, with the same definition of done. Review each area before merging it.
4. Re-run the checks in report.md §1, refresh the report, and open one PR.

The fixes in this branch were designed against the old module layout. Some may already be
fixed, moved or made moot by the restructure. Check each one against the new code before
porting it.
