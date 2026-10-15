/**
 * What `promise` settles to, or `fallback` when it has not settled within `ms`
 * or it rejects. For an answer worth waiting a moment for but never worth
 * holding the person up on: the work carries on in the background, and its
 * result, if it ever comes, is simply not read here.
 */
export function settleWithin<T>(promise: Promise<T>, ms: number, fallback: T): Promise<T> {
  return new Promise<T>((resolve) => {
    const timer = setTimeout(() => resolve(fallback), Math.max(0, ms));
    promise.then(
      (value) => {
        clearTimeout(timer);
        resolve(value);
      },
      () => {
        clearTimeout(timer);
        resolve(fallback);
      },
    );
  });
}
