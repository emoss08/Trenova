import { ApiRequestError } from "@trenova/shared/lib/api";
import type {
  PublicCheckCallKind,
  PublicTrackingStop,
} from "@trenova/shared/types/carrier-tracking";

export type CheckCallPostFailure =
  | { kind: "invalid" }
  | { kind: "closed"; message: string }
  | { kind: "throttled" }
  | { kind: "unavailable" }
  | { kind: "rejected"; messages: string[] };

// What a failed carrier post means for the page. Only the server's answer about
// the token itself says the link is dead; a refused stop or a missing delay
// reason is the carrier's to fix, and a load that finished is read-only, not
// broken. A network failure or a 5xx is never presented as a dead link.
export function checkCallPostFailure(error: unknown): CheckCallPostFailure {
  if (!(error instanceof ApiRequestError)) {
    return { kind: "unavailable" };
  }
  if (error.status === 429) {
    return { kind: "throttled" };
  }
  if (error.status >= 500) {
    return { kind: "unavailable" };
  }
  const normalized = error.normalize();
  if (normalized.fieldErrors.some((fieldError) => fieldError.field === "token")) {
    return { kind: "invalid" };
  }
  if (error.isBusinessError()) {
    return { kind: "closed", message: normalized.message };
  }
  if (error.status >= 400) {
    const messages = normalized.fieldErrors.map((fieldError) => fieldError.message);
    return { kind: "rejected", messages: messages.length > 0 ? messages : [normalized.message] };
  }
  return { kind: "unavailable" };
}

export type NextStopAction = {
  stop: PublicTrackingStop;
  kind: Extract<PublicCheckCallKind, "Arrival" | "Departure">;
};

// The visit the carrier is most likely reporting next: the first stop on the
// route not yet left, arriving if nobody recorded an arrival there yet.
export function nextStopAction(stops: readonly PublicTrackingStop[]): NextStopAction | null {
  const ordered = [...stops].sort((a, b) => a.sequence - b.sequence);
  const stop = ordered.find((candidate) => candidate.departedAt == null);
  if (!stop) {
    return null;
  }
  return { stop, kind: stop.arrivedAt == null ? "Arrival" : "Departure" };
}

export function newClientEventId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
}
