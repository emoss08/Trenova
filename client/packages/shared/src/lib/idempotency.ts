export const IDEMPOTENCY_KEY_HEADER = "Idempotency-Key";
export const IDEMPOTENT_REPLAYED_HEADER = "Idempotent-Replayed";

export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}

// A key is kept while the server may still hold the request it named: the request
// got no answer at all, or the answer was that it is still running (409). Any other
// answer settles the request, so the next attempt is a new request with a new key.
export function keepsIdempotencyKey(error: unknown): boolean {
  if (typeof error !== "object" || error === null) {
    return true;
  }
  const status = (error as { status?: unknown }).status;
  if (typeof status !== "number") {
    return true;
  }
  return status === 409;
}

export function withIdempotencyKeyHeader(
  headers: Record<string, string>,
  idempotencyKey: string | undefined,
): Record<string, string> {
  if (!idempotencyKey) {
    return headers;
  }
  return { ...headers, [IDEMPOTENCY_KEY_HEADER]: idempotencyKey };
}
