const explainedErrors = new WeakSet<object>();

/**
 * Remembers a request error that a dialog has already explained to the person (a plan
 * limit, a tab left behind by an organization switch), so the caller's own error reporting
 * stays quiet instead of repeating it as a generic toast.
 */
export function markExplained(error: object): void {
  explainedErrors.add(error);
}

/** Whether a dialog has already explained this error to the person. */
export function isExplainedError(error: unknown): boolean {
  return typeof error === "object" && error !== null && explainedErrors.has(error);
}
