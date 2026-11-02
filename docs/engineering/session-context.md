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
  itself, or a save sent with the cookie its session just replaced: it changes nothing in the
  app, and its caller still gets the error, so a refused save says so instead of vanishing.
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

## The stored session

- **The cookie is `ses_<ULID>.<secret>`** (`pkg/sessiontoken`). The ID finds the record, and
  the 256-bit secret proves the caller holds it. Redis stores only an HMAC of the secret, keyed
  by a key derived from `security.session.secret` and bound to the session and user IDs
  (`sessiontoken.Issuer`), so write access to Redis alone cannot mint a session or hand one
  to another person. Every stored session also carries a seal (`redis/authstore/seal.go`): an
  HMAC over what it acts for and how it was proved (person, organization, business unit,
  active roles, assurance, secrets, epoch, lifetime), written on create and update and checked
  on every read. A record whose seal does not match is discarded, so editing a session in
  Redis moves nobody to another tenant or role. Activity and display fields (`lastAccessedAt`,
  `accessConfirmedAt`, locale, user agent) stay outside the seal, so a touch is still one
  field write. Production refuses to start without `cache.password`; turn on `cache.tls`
  wherever the connection leaves the host. CSRF tokens use their own derived key (`api/csrf`);
  every derived key comes from `hashutils.DeriveKey` with its own label.
- **A change to a stored session goes through `SessionRepository.Update(ctx, id, mutate)`**
  (`redis/authstore/session.go`). It reads the latest record under `WATCH`, applies `mutate`
  and writes it back in `MULTI`. A write that lands in between makes it retry, and a session
  that has been deleted or has ended is `ErrSessionNotFound`, never recreated. A mutation that
  depends on what the caller read earlier checks it again inside `mutate`:
  `ActivateSessionRoles` refuses a session that switched organization since
  (`SessionContextStaleError`).
- **A rise in privilege gets a new secret.** Switching organization, activating roles and
  turning on two-factor rotate the secret in place through `sessionrotation.Rotator` and set
  a new cookie. The previous secret stays valid for `session.RotationGrace` (a minute), so
  reads already in flight with the old cookie are not refused; a password change keeps it
  only for `session.CredentialChangeGrace` (ten seconds). The previous secret is read-only:
  the auth middleware marks its requests (`authctx.UsesSupersededSessionSecret`), a REST write
  from one is refused, and so is a GraphQL mutation (`gqlext.ReadOnlyExtension`), both with
  `409 session-context-stale` rather than a sign-out. It cannot rotate the session either, so a
  copied cookie can neither change data nor take the session over inside the grace.
- **Streams follow the secret they opened with.** `sessionwatch.Watch` keys each stream by
  session and secret hash. A stream whose secret the session no longer accepts closes with
  `close: rotate` (the realtime stream) or ends quietly (a Desk turn or palette answer, which
  reattaches from its cursor), so the person's own tab reconnects with its new cookie and a
  stolen copy is refused. A stream that cannot be resumed (an agent dry run, which only the
  holder of the current secret can start) follows the session alone through
  `Watch.SessionContext`, so a rotation in another tab does not kill it. Every new stream opens
  through one of the two and checks `sessionwatch.Ended` when it stops. Signing out checks only
  the cookie's secret (`AuthService.SignOut`), so it ends the session even while the access
  check cannot run.
- **Changing a credential ends sessions.** A password change ends every other session of the
  person and gives the current one a new secret as its last step, falling back to ending every
  session when it cannot. A password reset, an administrator's two-factor reset, and losing
  sign-in access each end all of the person's sessions. Each of them also advances
  `users.session_epoch`; a session records the epoch it was issued under, and the access check
  ends a session whose epoch is behind. A sign-in re-reads the epoch right after it stores its
  session and discards the session if a change moved it in between, so a sign-in racing a
  password change never outlives it. A sign-in waiting on its second factor carries the epoch it started under
  and is refused if the epoch moved before the code arrived.
- **Access is re-checked on a clock that activity does not move.** A session issued by sign-in
  is checked against the person's membership, status and epoch once `accessConfirmedAt` is
  `lastAccessedResolution` (five minutes) old, however often keepalive moves
  `lastAccessedAt`; a confirmation time in the future forces the check. When the check
  cannot run, the session continues until it has gone `session.MaxUnconfirmed` (fifteen
  minutes) without a confirmation; after that its requests fail as a server error, not a
  sign-out, and keep the cookie, so access resumes when the database does. Neither a stale
  secret nor an unconfirmed session clears the cookie (`sessionports.KeepsCookie`), so a late
  response for an old secret cannot wipe a newly rotated one.

## The rule that bites

A new transport, stream or raw `fetch` to the API sends `withSessionHeaders`, and a transport
that reads its own failed responses passes them through `responseError` (`lib/api.ts`) so the
refusal reaches the handler. A new endpoint that returns the caller's user responds with
`respondWithSelf`, or every other device of that user is refused on every request. A new way
to end a session calls `announceSignedOut`. Never `Get` a session, edit it and write it back:
a session revoked in between comes back. Change it through `Update` with a `mutate`, and a
new privilege rise rotates the secret through `sessionrotation.Rotator`, whose handler sets
the cookie it returns.
