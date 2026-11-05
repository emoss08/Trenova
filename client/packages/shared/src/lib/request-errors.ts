import type { ZodIssue } from "zod";

/**
 * How long a request may go unanswered. The server cannot answer later than its write
 * timeout (75s), so this only ever ends a connection that has died, never a slow answer.
 * An event stream, a download and an upload are long by nature and have no deadline.
 */
export const REQUEST_TIMEOUT_MS = 90_000;

/** The deadline passed. A write that timed out may still have been applied. */
export class RequestTimeoutError extends Error {
  public constructor() {
    super("The server did not answer in time");
    this.name = "RequestTimeoutError";
  }
}

/** A response that does not match what this build expects, so replaying it cannot help. */
export class ResponseContractError extends Error {
  public readonly label: string;
  public readonly issues: readonly ZodIssue[];

  public constructor(label: string, issues: readonly ZodIssue[] = []) {
    super(`Unexpected response: ${label}`);
    this.name = "ResponseContractError";
    this.label = label;
    this.issues = issues;
  }
}

/**
 * Runs a request under the deadline. The signal `run` receives covers reading the body as
 * well as the headers, and aborts with the caller's, so a cancelled request stays a
 * cancellation and only the deadline becomes a RequestTimeoutError.
 */
export async function withRequestDeadline<T>(
  signal: AbortSignal | null | undefined,
  run: (signal: AbortSignal) => Promise<T>,
): Promise<T> {
  const controller = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, REQUEST_TIMEOUT_MS);
  const forward = () => controller.abort(signal?.reason);

  if (signal?.aborted) {
    forward();
  } else {
    signal?.addEventListener("abort", forward, { once: true });
  }

  try {
    return await run(controller.signal);
  } catch (error) {
    if (timedOut) throw new RequestTimeoutError();
    throw error;
  } finally {
    clearTimeout(timer);
    signal?.removeEventListener("abort", forward);
  }
}
