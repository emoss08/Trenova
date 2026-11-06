---
target: dispatch console
total_score: 23
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 4
target_identity: "file:/home/wolf/projects/Trenova/client/apps/web/src/routes/dispatch-console"
timestamp: 2026-11-05T13-42-49Z
slug: client-apps-web-src-routes-dispatch-console
---
Method: dual-agent (A: design review · B: detector)

Target: client/apps/web/src/routes/dispatch-console. Source-only review: the app was not running and the maintainer does not want a browser driven. Top findings were checked against source.

## Design Health Score

| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of System Status | 2 | No error state, no "as of" time, no connection state; a successful assign gives no feedback |
| 2 | Match System / Real World | 3 | Fluent trucking language; urgency words contradict the server's rules |
| 3 | User Control and Freedom | 2 | Undoing a reassign leaves the load uncovered instead of restoring the prior driver |
| 4 | Consistency and Standards | 2 | Drag gets a preflight, Assign does not; one bucket has four names |
| 5 | Error Prevention | 2 | Good guards exist, but the default assign path skips them |
| 6 | Recognition Rather Than Recall | 3 | Inspector drops hazmat, reefer and hold while choosing a driver |
| 7 | Flexibility and Efficiency | 3 | Strong keyboard and URL state; no move search, fleet filter or multi-select |
| 8 | Aesthetic and Minimalist Design | 3 | Disciplined; MoveCard can carry about 11 signals |
| 9 | Error Recovery | 1 | Failed reads appear as confident false statements |
| 10 | Help and Documentation | 2 | Guide is wrong about Assign; console keys missing from the shortcuts dialog |
| **Total** | | **23/40** | **Acceptable** |

## Design Specificity Verdict

Authored in its logic, generic in its shape. KPI strip, list rail, kanban and inspector is the standard operations template; the kanban is the least justified part. verdictLead and groupUncovered are excellent authored work. Missing: geography, HOS clocks on the timeline, and a visible agents-propose position on the board.

Deterministic scan: 0 findings across 39 files, exit 0. Not a clean result: the .tsx scan is regex-only, so rules needing a rendered page (tiny text, low contrast) never ran, and --json hides that notice. A seeded control fired 5 findings, so the regex path works; only 4 rules are proven on .tsx. No overlay: browser step skipped.

## Priority Issues

**[P0] Failed reads are drawn as a true, empty board.** page-content.tsx:78 reads only data and isLoading. On failure the strip shimmers forever, columns say "Nothing here", the inspector says "No eligible driver for this move." Fix: branch on isError in every read with describeError and a retry per pane; add one "as of {generatedAt}" line that warns when stale or disconnected; empty copy only after a successful read. Command: harden.

**[P1] The fastest ways to assign skip the safeguard.** Assign and digit keys commit directly; only drag opens preflight; console.md step 4 documents a preflight that does not happen. A digit reassigns a covered move silently. The digit listener checks only isTypingTarget, so a digit pressed inside a popover or some dialogs can assign. A failed preview leaves "Assign driver" enabled. Assign is not gated on ShipmentMove permissions. Fix: "2, Enter" through preflight, "Replaces {driver}", Undo restores the prior assignment, add guards and permission gates, advance selection. Command: harden.

**[P1] "Agents propose, a person approves" breaks at the seam.** Each Auto-assign click creates Pending proposals with a 7-day expiry. The console never uses proposalId, so Apply does not approve them and Discard only resets the client mutation. No reconciliation path was found. "Nothing is written until you apply" is not strictly true. No dry run or undo status; a person's click is stamped RunTriggerScheduled. Fix: make the proposal the one object; Apply approves, Discard rejects, a re-plan supersedes; show proposals on cards with the assist mark. Command: shape.

**[P1] Selection falls off the board, and the keyboard fails silently.** selectedMove resolves only against the board, so a Best fit match outside the window gives "Nothing selected". After "Nothing to plan", hasDialogOpen stays true while no dialog renders, so every hotkey dies. With an urgency focus on, j/k walks hidden moves. Fix: fetch off-board moves by ID and say why; pass the board window to driverMoves; clear empty plans; derive hasDialogOpen from what renders; walk visible moves only. Command: harden.

**[P1] The urgency language contradicts the server, and the board shape wastes the default view.** Client says Late is "Pickup window already open"; the server says Late means closed and open means Now (decorate.go). A move inside its window can show red "40m late". Tomorrow cannot populate at the default window (inferred); about 1.8 columns visible at 1440px, 3.6 at 1920px (inferred). Fix: one urgency definition from the server rule; replace the kanban default with a ranked queue or collapse empty columns. Commands: clarify, then layout.

## Persona Red Flags

**Alex (Power User):** no move or PRO search or fleet filter; selection lost after each assign and j resets to the top; console keys not in the shortcuts dialog; plan review has no select-none or sort.

**Sam (Accessibility-Dependent):** no aria-live, role="status", aria-selected or KeyboardSensor; j/k never moves DOM focus; MoveCard has no selected state or ui-focus-ring; FindingList severity is an aria-hidden icon; Planned and Tomorrow timeline text about 1.7:1 and 2.7:1 by calculation.

**Dispatcher mid-shift:** exceptions have no home; a move a colleague covers vanishes with no notice; "Free 14:30" has no date; a NeedsReview tender is the same 9px badge as everything else.

## Minor Observations

- Hard-coded English: coverage-kanban.tsx:137, preflight-dialog.tsx:278, tender-vocabulary.ts:79-81; server-side English uncovered reasons; tender-live-panel.tsx:426 compares against the English word.
- Partial data: bare comma from {originCity}, {originState}; boardLimit caps rows with no "showing N of M".
- Design drift: about 20 badges use h-4 rounded-md text-3xs instead of the 20px, 4px label; 28 opacity tints; hand-built toggles where SegmentedControl exists; timestamps not in mono.
- Tomorrow uses a category accent in an ordered set, breaking the Ordering Rule.
- Bulk apply partial failure shows only the first failure's text.

## Questions to Consider

- Why is the home view a board at all, rather than one ranked queue where each load arrives with its proposed driver?
- Why is an agent's plan a modal rather than a state of the board?
- Is this the console for the day, or only the first hour of a load's life?

## Run notes

Dropped: the claim that text-warning is 1.8:1 on panels; the theme maps it to the readable rung. Not verified: "no reconciliation path" is a found-none, not a proof; column widths and items marked inferred are reasoned from classes.
