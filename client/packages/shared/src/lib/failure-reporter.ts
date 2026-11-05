import {
  describeError,
  type ErrorDescription,
  type ErrorKind,
} from "@trenova/shared/lib/error-presentation";
import { isExplainedError } from "@trenova/shared/lib/explained-errors";

export type FailureContext = {
  source: "query" | "mutation" | "boundary" | "unhandled" | "background";
  operation?: string;
};

type FailureReporter = (description: ErrorDescription, context: FailureContext) => void;

/** The kinds that mean something is wrong with Trenova, not with the request or the network. */
const REPORTED_KINDS: ReadonlySet<ErrorKind> = new Set<ErrorKind>([
  "server",
  "contract",
  "unexpected",
]);

const logFailure: FailureReporter = (description, context) => {
  console.error(`[failure] ${context.source} ${context.operation ?? ""}`.trim(), {
    kind: description.kind,
    message: description.message,
    status: description.status,
    problemType: description.problemType,
    traceId: description.traceId,
  });
};

let reporter: FailureReporter = logFailure;
const reported = new WeakSet<object>();

/** Where failures go. The one place a telemetry service would plug in. */
export function setFailureReporter(next: FailureReporter): void {
  reporter = next;
}

function isCancellation(error: unknown): boolean {
  return error instanceof DOMException && error.name === "AbortError";
}

/**
 * Records a failure once, however many surfaces saw it: a query's retries end in one error
 * object that its cache and an error boundary can both hand here.
 */
export function reportFailure(error: unknown, context: FailureContext): void {
  if (isCancellation(error) || isExplainedError(error)) return;
  if (typeof error === "object" && error !== null) {
    if (reported.has(error)) return;
    reported.add(error);
  }

  const description = describeError(error);
  if (REPORTED_KINDS.has(description.kind)) reporter(description, context);
}

/** Catches what nothing else did: a rejected promise or a throw outside React. */
export function installFailureListeners(): void {
  window.addEventListener("unhandledrejection", (event) =>
    reportFailure(event.reason, { source: "unhandled" }),
  );
  window.addEventListener("error", (event) => {
    if (event.error) reportFailure(event.error, { source: "unhandled" });
  });
}
