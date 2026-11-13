# Notifications

How Trenova tells a person something, and what keeps the panel worth opening. A notification
is about one **subject**, and a person has one open row for it. That rule, and the three that
follow from it (it says why it reached you, it can be acted on where it is read, and it leaves
when the work is done), are what the rest of this doc protects.

## The shape of it

| Piece | Where |
|---|---|
| Entity, subject rules, lenses, reasons | `internal/modules/workspace/domain/notification/` (`notification.go`, `subject.go`, `enums.go`) |
| Create, collapse, lists, counts, the writes | `internal/modules/workspace/adapters/postgres/notificationrepository/` |
| Delivery (preferences, quiet hours, email, push), fan-out, resolve | `internal/modules/workspace/app/notificationservice/` |
| GraphQL | `internal/api/graphql/schema/notification.graphqls`, `resolver/notificationresolver` |
| Client reads and writes | `client/packages/shared/src/{services,hooks,lib/queries,types}/notification*` |
| Panel, row, registry | `client/apps/web/src/components/notification-center/` |
| Driver portal list | `client/apps/dash/src/routes/notifications.tsx` (the `open` lens, driver-scoped operations) |

## The row

| Column | Means |
|---|---|
| `subject_key` | what the notification is about, chosen by the producer; empty for one that never repeats |
| `reason` | `direct` (it is the person's own: assigned, mentioned, asked for, waits on them) or `team` (it went to everyone who can act) |
| `occurrence_count` | how many times the subject came up while the row was open |
| `surfaces_at` | when it shows: now, or later when the person set it aside |
| `dismissed_at` | set when it is done, by the person or by itself |
| `resolved_at`, `resolution` | set when it cleared itself, with a code for how (`ResolutionCleared`, `ResolutionResolved`, `ResolutionDecided`) |

A row is in exactly one part of the panel, worked out from the row (`lensFilter` on the server,
`lensOfNotification` on the client, which must agree):

| Lens | Rows |
|---|---|
| `forYou` | open, surfaced, `reason = direct` |
| `team` | open, surfaced, `reason = team` |
| `later` | open, `surfaces_at` in the future |
| `done` | `dismissed_at` set, newest finished first |
| `open` | `forYou` and `team` together: the driver portal's one list |

The bell counts `forYouUnread` only. A broadcast waits in Team without lighting the bell, so
the bell means "something is yours".

## Producing one

Call `notificationservice.Create` (one person) or `NotifyPermitted` (everyone who holds a
permission, which sets `reason = team`). Then decide three things.

1. **The subject.** Set `SubjectKey` with `notification.SubjectKey(parts...)` when the same
   thing can be said twice: a reminder, an escalation, a recount. A second notification with
   the key of an open row is **absorbed** into it (`Notification.Absorb`): the latest title,
   message and data win, the count goes up, the row becomes unread and moves to the top. It
   does not add a row. A broadcast that would otherwise be one row per record takes a key per
   kind and day (`DaySubjectKey`) and reports each record with `WithOccurrence`; the row keeps
   the newest three under `data.recent` and the panel lists them. Set `OccurrenceCount` to the
   number of new things the notice stands for, or to `NoNewOccurrence` for a reminder.
2. **The reason.** Leave it empty for the default (direct when it targets a person), or set
   `ReasonTeam` when it reaches the person only because they are one of many who can act.
3. **How it ends.** When the work the notification points at is finished, call
   `ResolveSubject` with the key and a resolution. Every open row for the subject moves to
   Done saying why, and one `notifications`/`resolved` event tells open panels. A producer
   that cannot tell when the work is finished leaves the row for the person to finish.

Titles say what happened ("Missed pickup"), not who it is for: the panel says why the person
got it. Data never carries a record id the row would show (`design-system.md`, "No record ids
on screen").

## Rules that bite

- **One open row per subject per person** is the unique partial index
  `uq_notifications_open_subject`. `createOrAbsorb` locks the open row and saves into it; two
  first notices racing each other hit the index, and the loser retries once outside its
  transaction and absorbs. Never insert around the repository.
- **A row set aside stays set aside** when its subject comes up again, unless the new notice
  is critical. Snoozing is `surfaces_at` in the future, at most 30 days; it comes back unread.
- **Every list and count is bounded.** Lists are keyset pages on `(surfaces_at, id)`, or
  `(dismissed_at, id)` for Done, each behind its own partial index. The tab counts are one
  batch of `CountUpTo` capped at 1,000, and the panel shows `999+` past that.
- **Marking a whole part done** is `DismissAll(lens)`, one statement. It is refused for Done
  and for anything that is not a part of the panel.
- **A new kind needs a registry entry** (`notification-registry.ts`) to get a look, a link, a
  line saying why, and actions. Without one it is drawn as a system notice with only its
  `data.link`.

## The panel

`notification-sheet.tsx` is the bell and the sheet, in the app header's chunk; everything
else loads when the bell is first pointed at. `notification-panel.tsx` reads the counts and
only the part that is open, and draws it with `VirtualRows`, so a long inbox mounts the rows
near the viewport. Its fades use `ScrollArea`'s `overlay` mask, the colour of the sheet it
sits in; a mask that does not match its surface draws a visible band.

A row (`notification-item.tsx`) takes what it offers from the registry:

- `why` — the line under the message. A team row with none shows its category.
- `actions` — `NotificationRowAction[]`. `open` goes somewhere and marks the row read, or
  done when it `settles` (reading a reply is all the notification asked for). The other kinds
  change the record through the operation its own page uses (`notification-actions.ts`), so
  the permission check and audit are the page's; the row moves to Done when the write
  succeeds, and stays, with the server's refusal, when it does not.
- `status` — a state worth a badge beyond priority ("Waiting 3 days").

Adding an action is a new member of `NotificationRowAction`, its case in
`runNotificationAction`, and the descriptor that returns it. Offer only what can be done
without a form: denying time off needs a reason, so the row opens the request instead.

Writes are optimistic across every cached part at once (`applyOptimistic` in
`use-notifications.ts`): the row leaves its list and both tab counts move in the same frame,
and the server's answer replaces all of it when the request settles. A row folds shut before
it leaves so the list closes over it (`useLeavingRows`); the write still runs if the panel
closes mid-fold.

The keyboard works the list from the moment the sheet opens: `J`/`K` move, `Enter` opens,
`E` finishes, `H` opens Later. The sheet element receives the keys and hands them to the
panel, because the sheet's own close button can hold the focus.

## Checking it

`node e2e/mock-api/notifications-screenshots.mjs` shoots every part of the panel against the
mock API (`handlers/notifications.mjs`, `fixtures/notifications.mjs`): the four parts, a
counted row expanded, both menus, an action taken, the empty state, light and phone width.
