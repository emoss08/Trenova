# Session context: tabs a sign-in, sign-out or organization switch left behind

Every tab in a browser shares one session cookie, and the session holds the person,
organization and business unit it acts for. A change in one tab moves them all:
`SwitchOrganization` rewrites the organization on the session, signing out ends it, and signing
in replaces it, perhaps with someone else's. Without a guard, a tab still showing organization X
writes into Y (a create carries no tenant, `authctx.Bind` stamps the session's), a tab still
showing one dispatcher's work writes as the next one to sign in, and a signed-out tab keeps the
last person's records on screen for whoever sits down next.

## How it works

- **The client asserts.** Every first-party request carries the tab's confirmed session in
  `X-Trenova-User-Id`, `X-Trenova-Organization-Id` and `X-Trenova-Business-Unit-Id`
  (`client/packages/shared/src/lib/session-context.ts`). `withSessionHeaders` in `lib/api.ts`
  adds them with the CSRF token; a raw `fetch` to the API calls `withSessionHeaders` or
  `withSessionContextHeaders`, never builds the headers by hand.
- **Only the server confirms.** The tab's session comes from server answers (sign-in, a switch,
  `/users/me`) through the auth store's subscription, which reacts only to a new user. The user
  restored from storage at startup is not a confirmation, so requests assert nothing until the
  startup session check answers. Every handler that returns the caller's own user goes through
  `userhandler.respondWithSelf`, which reports the session's tenant: the `users` row records the
  organization last chosen on any device, which another device's session may not share.
- **The server refuses.** `middleware.SessionContextMiddleware` (protected group, after
  authentication and before the CSRF check, the read-only guard and idempotency: the CSRF
  token belongs to the session, so a tab someone else's sign-in left behind would otherwise
  get a CSRF failure for its writes instead of this answer) compares the headers with the auth context and answers
  `409 session-context-stale` (`errortypes.SessionContextStaleError`), whose `params` name the
  session the cookie acts for now. A request without the headers (API keys, webhooks, curl) is
  not checked. The router always allows the headers through CORS
  (`middleware.WithSessionContextHeaders`), whatever the config lists.
- **Streams end with their session.** A live stream and an assistant turn stream are bound to
  the session that opened them (`sessionwatch.Watch`): when it ends anywhere, on any device,
  the server closes them within `sessionCheckInterval` (realtime.md, "Ending"), and the
  realtime client tells the app, which ends the session in the tab as it would for a 401.
- **The tabs tell each other.** A confirmed session, and the end of one (`announceSignedOut`:
  `logout`, and the session-expiry handler), go to every other tab on a `BroadcastChannel`.
  Clearing a tab's session tells no one, so a tab leaving a session it lost never signs out
  the one that replaced it.
- **The tab acts.** The REST, GraphQL and realtime transports report a refusal to
  `reportSessionContextStale`; it and the channel give the app one status
  (`lib/session-context-handler.ts`). An organization change puts the reload prompt
  (`OrganizationChangedDialogHost`) in front of the tab until it reloads, the person keeps the
  page open, or the session comes back to its organization. A sign-out, or someone else signing in, leaves at once through
  `leaveEndedSession`, which clears the auth state and cache before loading the sign-in page or
  the new person's session, past any unsaved work (`leaveRegardlessOfWork`: the session was
  ended on purpose, and the prompt would keep the last person's screen up). A session the
  server ended (the expiry handler) leaves through the browser's prompt instead, since the
  person still at the screen may want to copy what they typed. A refusal naming the tab's own session is a request it sent just before switching
  itself, and changes nothing.
- **The driver portal shares the cookie** (it is served from the same origin under `/dash/`),
  so it registers its own handler (`apps/dash/src/lib/session-context-handler.ts`): any change
  clears its auth state and loads the portal sign-in, or the portal for the session the cookie
  holds now. It keeps its records only in memory and has no unsaved-work guard, so a fresh
  load is all the teardown it needs.
- **The reload prompt keeps the person's work reachable.** "Keep this page open" swaps the
  prompt for a bar (`OrganizationChangedBanner`, fixed to the bottom of every page),
  and later refusals do not bring the prompt back. Both Reload buttons are the person's own
  choice, so they reload normally and the browser still warns about unsaved edits.
- **A sign-in page follows.** When a session passes the app's session gate (`requireSession`),
  the tab announces it once (`announceSignedIn`): any organization and role steps are behind
  it. A tab holding no session, as a sign-in page does, follows it into the app through
  `useFollowSignInElsewhere` (the web sign-in flow and the driver portal's login). A tab in
  the middle of its own sign-in already holds a session and never follows.

## The rule that bites

A new transport, stream or raw `fetch` to the API sends `withSessionHeaders`, and a transport
that reads its own failed responses passes them through `responseError` (`lib/api.ts`) so the
refusal reaches the handler. A new endpoint that returns the caller's user responds with
`respondWithSelf`, or every other device of that user is refused on every request. A new way
to end a session calls `announceSignedOut`.
