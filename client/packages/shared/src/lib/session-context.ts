import { markExplained } from "@trenova/shared/lib/explained-errors";
import { isRecord } from "@trenova/shared/lib/utils";
import { parseProblemType } from "@trenova/shared/types/errors";

export const USER_HEADER = "X-Trenova-User-Id";
export const ORGANIZATION_HEADER = "X-Trenova-Organization-Id";
export const BUSINESS_UNIT_HEADER = "X-Trenova-Business-Unit-Id";

const STALE_PROBLEM_TYPE = "session-context-stale";
const CHANNEL_NAME = "trenova-session-context";

/** The person, organization and business unit a tab loaded its records for. */
export type SessionContext = { userId: string; organizationId: string; businessUnitId: string };

/**
 * What the shared session cookie now means for this tab. Every tab shares it, so a change in
 * one moves them all: switching organization leaves the others on the wrong one, signing out
 * ends them, and signing in as someone else hands them to that person. `current` is the
 * session acting for this tab again.
 */
export type SessionContextStatus =
  | "current"
  | "organization-changed"
  | "user-changed"
  | "signed-out";

export type SessionContextStatusHandler = (status: SessionContextStatus) => void;

type SessionUser = {
  id?: string | null;
  currentOrganizationId?: string | null;
  businessUnitId?: string | null;
};

type ChannelMessage =
  | { kind: "confirmed"; session: SessionContext }
  | { kind: "signed-in" }
  | { kind: "signed-out" };

let confirmed: SessionContext | null = null;
let statusHandler: SessionContextStatusHandler | null = null;
let channel: BroadcastChannel | null = null;
let announcedSignIn: SessionContext | null = null;
const signInListeners = new Set<() => void>();

export function sessionContextOf(user: SessionUser | null | undefined): SessionContext | null {
  const userId = user?.id?.trim();
  const organizationId = user?.currentOrganizationId?.trim();
  const businessUnitId = user?.businessUnitId?.trim();
  return userId && organizationId && businessUnitId
    ? { userId, organizationId, businessUnitId }
    : null;
}

function sameSession(a: SessionContext | null, b: SessionContext | null): boolean {
  return (
    a?.userId === b?.userId &&
    a?.organizationId === b?.organizationId &&
    a?.businessUnitId === b?.businessUnitId
  );
}

function parseSessionContext(value: unknown): SessionContext | null {
  if (!isRecord(value)) {
    return null;
  }
  const { userId, organizationId, businessUnitId } = value;
  return typeof userId === "string" &&
    typeof organizationId === "string" &&
    typeof businessUnitId === "string"
    ? sessionContextOf({ id: userId, currentOrganizationId: organizationId, businessUnitId })
    : null;
}

function parseMessage(value: unknown): ChannelMessage | null {
  if (!isRecord(value)) {
    return null;
  }
  if (value.kind === "signed-out" || value.kind === "signed-in") {
    return { kind: value.kind };
  }
  const session = value.kind === "confirmed" ? parseSessionContext(value.session) : null;
  return session ? { kind: "confirmed", session } : null;
}

/**
 * Where this tab stands against the session the server or another tab reports. A refusal
 * that does not say which session it found is treated as an organization change: the tab
 * cannot tell, and the reload prompt is right for either.
 */
function verdict(current: SessionContext, session: SessionContext | null): SessionContextStatus {
  if (!session) {
    return "organization-changed";
  }
  if (session.userId !== current.userId) {
    return "user-changed";
  }
  return sameSession(current, session) ? "current" : "organization-changed";
}

function notify(status: SessionContextStatus): void {
  try {
    statusHandler?.(status);
  } catch (error) {
    console.error("[session-context] handler failed", error);
  }
}

function receive(message: ChannelMessage): void {
  if (message.kind === "signed-in") {
    if (!confirmed) {
      for (const listener of signInListeners) {
        try {
          listener();
        } catch (error) {
          console.error("[session-context] sign-in listener failed", error);
        }
      }
    }
    return;
  }
  if (!confirmed) {
    return;
  }
  notify(message.kind === "signed-out" ? "signed-out" : verdict(confirmed, message.session));
}

function sessionChannel(): BroadcastChannel | null {
  if (channel || typeof BroadcastChannel === "undefined") {
    return channel;
  }
  channel = new BroadcastChannel(CHANNEL_NAME);
  channel.addEventListener("message", (event: MessageEvent<unknown>) => {
    const message = parseMessage(event.data);
    if (message) {
      receive(message);
    }
  });
  return channel;
}

function post(message: ChannelMessage): void {
  sessionChannel()?.postMessage(message);
}

/**
 * Records the session the server just confirmed for this tab, and tells the other tabs the
 * cookie now acts for it. Only a server answer (sign-in, an organization switch, a session
 * check) is a confirmation: a user restored from storage at startup is not, because another
 * tab may have moved the session since it was written. Clearing it tells no one: a tab that
 * leaves a session it lost must not sign out the one that replaced it.
 */
export function setSessionContext(next: SessionContext | null): void {
  if (sameSession(confirmed, next)) {
    return;
  }
  confirmed = next;
  if (!next) {
    announcedSignIn = null;
    return;
  }
  notify("current");
  post({ kind: "confirmed", session: next });
}

/**
 * Tells every other tab that a signed-in session has reached the app, once per session: the
 * sign-in flow is finished (any organization and role steps are behind it), so a tab still on
 * a sign-in page can follow it in instead of asking the person to sign in again.
 */
export function announceSignedIn(): void {
  if (!confirmed || sameSession(announcedSignIn, confirmed)) {
    return;
  }
  announcedSignIn = confirmed;
  post({ kind: "signed-in" });
}

/**
 * Calls `listener` when another tab announces a finished sign-in while this tab holds no
 * session, as a sign-in page does. Returns the function that stops listening.
 */
export function onSignedInElsewhere(listener: () => void): () => void {
  signInListeners.add(listener);
  sessionChannel();
  return () => {
    signInListeners.delete(listener);
  };
}

/**
 * Tells every other tab the session has ended (the person signed out, or the server ended
 * it), so none is left showing the last person's records on a shared computer.
 */
export function announceSignedOut(): void {
  post({ kind: "signed-out" });
}

/**
 * Stamps a first-party request with the tab's confirmed session. The server refuses it when
 * the cookie has since moved to another person or organization, so a tab left behind cannot
 * write as someone else, or into an organization the person cannot see on its screen.
 */
export function withSessionContextHeaders(headers?: HeadersInit): Headers {
  const next = new Headers(headers);
  if (confirmed) {
    next.set(USER_HEADER, confirmed.userId);
    next.set(ORGANIZATION_HEADER, confirmed.organizationId);
    next.set(BUSINESS_UNIT_HEADER, confirmed.businessUnitId);
  }
  return next;
}

/**
 * The transports sit below the router and the dialogs, so they can only signal. The app
 * registers the one handler that acts on a changed session; without one a refusal still
 * throws to its caller and is reported the ordinary way.
 */
export function setSessionContextStatusHandler(handler: SessionContextStatusHandler | null): void {
  statusHandler = handler;
  if (handler) {
    sessionChannel();
  }
}

let sessionFoundEndedHandler: (() => void) | null = null;

/** What the app does when it learns on load that no session exists, however that came about. */
export function setSessionFoundEndedHandler(handler: (() => void) | null): void {
  sessionFoundEndedHandler = handler;
}

export function sessionFoundEnded(): void {
  sessionFoundEndedHandler?.();
}

type ProblemLike = { type?: unknown; params?: unknown };

/** The problems a thrown request error carries: a REST problem body or GraphQL errors. */
function problemsOf(error: object): ProblemLike[] {
  if ("graphQLErrors" in error && Array.isArray(error.graphQLErrors)) {
    return error.graphQLErrors.filter(isRecord);
  }
  if ("data" in error && isRecord(error.data)) {
    return [error.data];
  }
  return [];
}

function isStaleProblem(problem: ProblemLike): boolean {
  return typeof problem.type === "string" && parseProblemType(problem.type) === STALE_PROBLEM_TYPE;
}

/**
 * Called by a transport with the error it is about to throw. A refusal names the session the
 * cookie acts for now: a request this tab sent just before switching organization itself, or
 * a save sent with the cookie its session just replaced, is refused too, and must not mark the
 * tab left behind. Only a refusal the app answers with its own prompt is explained; one that
 * names this tab's session is left to its caller, so a refused save says so instead of
 * failing silently. Returns whether the error was explained.
 */
export function reportSessionContextStale(error: object): boolean {
  const refusal = problemsOf(error).find(isStaleProblem);
  if (!refusal || !confirmed || !statusHandler) {
    return false;
  }

  const status = verdict(confirmed, parseSessionContext(refusal.params));
  notify(status);
  if (status === "current") {
    return false;
  }

  markExplained(error);
  return true;
}
