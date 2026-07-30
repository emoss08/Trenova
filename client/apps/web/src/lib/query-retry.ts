import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { retryableProblemTypes } from "@trenova/shared/types/errors";

// Retry policy for the QueryClient. Kept out of providers.tsx so that module exports only
// its component — a non-component export there breaks fast refresh for the provider tree,
// and the policy is not a provider concern.
const MAX_ATTEMPTS = 3;
const RETRY_BASE_DELAY_MS = 500;
const RETRY_MAX_DELAY_MS = 8_000;

// Status alone decides most cases. A request that never reached a resolver carries no
// extensions at all — the auth middleware, the rate limiter, the CSRF check, the
// body-size limit and any proxy in front of the API all reject before gqlgen runs — so
// keying only on the server's problem type would leave the most common transient class
// unretryable. 5xx means the server owns the fault; 4xx means the request does, and
// replaying it cannot change the answer.
function isRetryableStatus(status: number | undefined): boolean | null {
  if (status === undefined || status === 0) {
    return null;
  }
  if (status >= 500) {
    return true;
  }
  if (status >= 400) {
    return false;
  }

  return null;
}

function isRetryableError(error: unknown): boolean {
  if (error instanceof GraphQLRequestError) {
    const byStatus = isRetryableStatus(error.status);
    if (byStatus !== null) {
      return byStatus;
    }

    // No usable status: either a resolver failure inside a 200 or a connection that
    // never produced a response. The former is classified by the server, the latter is
    // transport and always worth one more try.
    if (error.isTransportError()) {
      return true;
    }

    return error.getProblemTypes().some((problemType) => retryableProblemTypes.has(problemType));
  }

  if (error instanceof ApiRequestError) {
    const byStatus = isRetryableStatus(error.status);
    if (byStatus !== null) {
      return byStatus;
    }

    const problemType = error.getProblemType();
    return problemType !== null && retryableProblemTypes.has(problemType);
  }

  // A rejected fetch never becomes an ApiRequestError or a GraphQLRequestError — it
  // surfaces as the raw TypeError the platform throws when the connection fails.
  return isConnectionFailure(error);
}

function isConnectionFailure(error: unknown): boolean {
  if (error instanceof DOMException && error.name === "AbortError") {
    return false;
  }

  return error instanceof TypeError;
}

// retryDelay uses equal-jitter backoff: half the window is the deterministic ceiling and
// half is random, so a server recovering from a fault does not take the whole fleet's
// retries back in one synchronised burst the way pure exponential backoff would.
export function retryDelay(failureCount: number): number {
  const ceiling = Math.min(RETRY_BASE_DELAY_MS * 2 ** failureCount, RETRY_MAX_DELAY_MS);
  return Math.round(ceiling / 2 + (Math.random() * ceiling) / 2);
}

// TanStack reads failureCount before incrementing it, so the first decision arrives with
// 0. Stopping below MAX_ATTEMPTS - 1 therefore allows the initial call plus two replays.
export function shouldRetry(failureCount: number, error: unknown): boolean {
  return failureCount < MAX_ATTEMPTS - 1 && isRetryableError(error);
}
