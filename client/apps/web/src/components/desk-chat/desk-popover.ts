import { createContext, use } from "react";

/**
 * The surface every Desk popover floats on, spent on the shared
 * `PopoverContent`: the Desk's raised panel, lifted, with no padding or gap of
 * its own so each popover's rows keep the Desk's spacing. It is portalled out
 * of the room, so it carries the room's variables and resets (`dk-chat`) and
 * its type (`dk-float`, in desk-frame.css).
 */
export const deskPopoverClass =
  "dk-chat dk-float gap-0 overflow-hidden rounded-lg bg-dsk-raised p-0 text-dsk-fg ui-lift-float";

type ComposerPopoverFrame = { element: HTMLElement | null; dense: boolean };

/** The composer a menu opened from it hangs over, provided by `DeskComposer`. */
export const ComposerPopoverContext = createContext<ComposerPopoverFrame>({
  element: null,
  dense: false,
});

/**
 * Where a menu opened from the composer's toolbar hangs. In a dense Desk it
 * hangs above the whole composer, as wide as it is, instead of from its own
 * button; elsewhere it hangs from the button and `anchor` is left unset.
 */
export function useComposerPopover(): { anchor: HTMLElement | undefined; wide: boolean } {
  const { element, dense } = use(ComposerPopoverContext);
  return dense && element ? { anchor: element, wide: true } : { anchor: undefined, wide: false };
}
