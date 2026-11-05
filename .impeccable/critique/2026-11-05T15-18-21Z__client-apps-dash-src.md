---
target: driver portal
total_score: 19
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 4
target_identity: "file:/home/wolf/projects/Trenova/client/apps/dash/src"
timestamp: 2026-11-05T15-18-21Z
slug: client-apps-dash-src
---
Method: dual-agent (A: design review · B: detector). Target client/apps/dash/src. Source-only; claims checked against source.

| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of System Status | 1 | Failed requests read as false empty states; no offline status |
| 2 | Match System / Real World | 3 | Strong driver vocabulary; "Money" tab translates as an amount |
| 3 | User Control and Freedom | 2 | Drafts wiped on drawer close; idle sign-out reloads the page |
| 4 | Consistency and Standards | 2 | Error handling differs by screen; Depart green, Arrive ink |
| 5 | Error Prevention | 1 | Arrive/Depart a 24px chip 6px from Directions, no undo |
| 6 | Recognition Rather Than Recall | 3 | "Current load" is the list's first item |
| 7 | Flexibility and Efficiency | 2 | Arrive buried under documents, chat and stats |
| 8 | Aesthetic and Minimalist Design | 2 | Earned figure graphite on canvas; Profile ~16 sections |
| 9 | Error Recovery | 1 | retry: false globally; no retry on Loads, Pay, Home |
| 10 | Help and Documentation | 2 | "Call your dispatcher" four times, nothing to tap |
| **Total** | | **19/40** | **Poor** |

Specificity: gig-earnings template in a trucker's voice at office density. Cab Floor Rule has no tokens, utility or lint. Detector: 0 findings across 64 files, regex only; size, contrast and target rules never ran.

## Priority Issues
- [P0] Failed requests say something false; no offline story (retry off, no isError, SW has no fetch, no refetch on focus). harden.
- [P1] Controls sized for a mouse: Arrive/Depart ~24px, dispute 22px, POD 28px, back links ~17px, header 32px. adapt.
- [P1] Type, inputs, ink below floor: 90% of sizes 11-12px, inputs 13px (iOS zoom), 199 graphite facts. typeset.
- [P1] A glance doesn't show the next action; HOS violation alert below gauges. layout.
- [P1] Session and drafts don't survive interruption: 30-min idle, reload, drawers reset. harden.

## Persona Red Flags
Casey: top-placed small targets, swipe loses drafts, re-sign-in after 30 min. Sam: unlabelled textareas, chip pickers without radio semantics, no reduced-motion support. Driver at a dock: 24px Arrive by Directions, zooming inputs, graphite pay, false "No active loads".

## Minor Observations
Hard-coded English; no zone on appointment times; 48px .ico manifest and portrait lock; emoji in copy; opacity tints; nested 320px chat scroll.

## Questions to Consider
Idle timeout on a driver's phone? Active stop as home? Pay vs Money: driver's model or the schema's?
