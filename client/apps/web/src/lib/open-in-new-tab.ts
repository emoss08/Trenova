/**
 * Opens a page in a new tab that has no way back to this one: the tab gets no
 * `window.opener`, and the address it was opened from is not sent. What opens
 * may be a stored file or a page Trenova doesn't control.
 */
export function openInNewTab(url: string): void {
  window.open(url, "_blank", "noopener,noreferrer");
}
