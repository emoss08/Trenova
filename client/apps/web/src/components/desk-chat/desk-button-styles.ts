/**
 * The Desk's own button looks, spent on a shared `Button` (`variant="bare"
 * size="bare"`, or `variant="quiet"` with an icon size) wherever the Desk, its
 * composer or the in-app assistant draws them, so each is written once.
 */

/** Underlined text that acts: "Send next", "Cancel", a status line's action. */
export const deskLinkClass =
  "text-sm font-medium whitespace-nowrap text-dsk-fg underline decoration-dsk-fg/30 underline-offset-2 hover:decoration-current";

/** The 30px icon command: subtle at rest, filled on hover. */
export const deskIconClass =
  "size-7.5 justify-center rounded-lg text-dsk-subtle transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg active:bg-dsk-hover";

/** The same command at 28px, in the assistant panel's header. */
export const deskPanelIconClass =
  "size-7 justify-center rounded-md text-dsk-subtle transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg disabled:opacity-40 aria-pressed:bg-dsk-hover aria-pressed:text-dsk-fg";

/** The same command at 24px, on a queued message or a wait. */
export const deskSmallIconClass =
  "size-6 justify-center rounded-md text-dsk-subtle transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg active:bg-dsk-hover disabled:opacity-35";

/** An icon command's open state, while the menu it opens is showing. */
export const deskIconOnClass = "bg-dsk-hover text-dsk-fg";
