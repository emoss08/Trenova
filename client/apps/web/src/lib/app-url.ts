/** An address inside the app as a full URL on the app's own origin, for copying and sharing. */
export function absoluteAppUrl(path: string): string {
  return new URL(path, window.location.origin).toString();
}
