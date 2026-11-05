---
target: the Desk
total_score: 23
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 3
target_identity: "file:/home/wolf/projects/Trenova/client/apps/web/src/routes/desk"
timestamp: 2026-11-05T15-18-21Z
slug: client-apps-web-src-routes-desk
---
Method: dual-agent (A: design review · B: detector). Target client/apps/web/src/routes/desk + components/desk-chat (the room). Source-only; top claims checked.

| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of System Status | 3 | Strong working line; rail row state only in a tooltip |
| 2 | Match System / Real World | 3 | Excellent failure copy; AI called "Desk", vendors in composer |
| 3 | User Control and Freedom | 2 | Stop vanishes once box has text; no Reject |
| 4 | Consistency and Standards | 1 | Parallel design system; ⌘ hard-coded on Windows in 10 places |
| 5 | Error Prevention | 2 | Strong server gates, undercut by the send-and-approve key |
| 6 | Recognition Rather Than Recall | 2 | Card says "1 shipment", not which |
| 7 | Flexibility and Efficiency | 3 | Rich keys; no stop key; ⌘N reserved by Chrome |
| 8 | Aesthetic and Minimalist Design | 2 | Dock can stack ~12 items |
| 9 | Error Recovery | 3 | Among the best in the product |
| 10 | Help and Documentation | 2 | No shortcut list |
| **Total** | | **23/40** | **Acceptable** |

Specificity: authored behaviour (preview digest, redraft, approve-some, steering, citations) in a category-standard look with AI tells (rainbow ring, comet, confetti default, logo rain, coloured kinds). ~27,800 lines of Desk-only CSS; documented room primitives missing or zeroed. Detector: 52 findings, regex only; 49 bounce-easing (37 break the Motion rule, 7 confirmations, 5 undecidable), 3 side-tab false positives.

## Priority Issues
- [P0] Ctrl/Cmd+Enter sends and approves in one keystroke when "Send with ⌘ Enter" is on (desk-thread.tsx:636-640 never checks defaultPrevented; composer inside at :1661). harden.
- [P1] Approval card buries the facts: no record name, faintest "can't be undone", no Reject, no 1 of N. clarify.
- [P1] Parallel design system filled with AI clichés. distill.
- [P1] Keyboard and screen-reader foundations broken in rail and thread. harden.
- [P2] Today doesn't open the day. shape.

## Persona Red Flags
Alex: ⌘N likely swallowed, no stop key, ⌘↵ on Windows, no decision keys. Sam: nested rail controls, unlabelled log, repeated alerts, faint focus ring. Rosa: card doesn't name 1048813, no Reject, confetti mid-shift.

## Minor Observations
Workspace ignores profile timezone; two "Today"s; zero counts; internal path in copy link; blurred stack; triple .dk-dcx; Esc in composer folds workspace.

## Questions to Consider
Room as design or memory? Type and approve in one place? Who is the AI? What does confetti teach?
