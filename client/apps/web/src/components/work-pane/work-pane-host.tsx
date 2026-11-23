import { SidePane } from "@/components/side-pane";
import { scrollParent } from "@/lib/dom";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, type ReactNode, type RefObject } from "react";
import { useWorkPaneApi, useWorkPaneState } from "./work-pane-context";

/**
 * Keeps a sticky element exactly as tall as the room left below its top in whatever scrolls it:
 * the rest of the viewport while the page above it is in view, all of it once that has scrolled
 * away. Written straight to the element, so scrolling never re-renders.
 */
function useFillsScroller(ref: RefObject<HTMLElement | null>, active: boolean) {
  useEffect(() => {
    const element = ref.current;
    const row = element?.parentElement;
    if (!active || !element || !row) return;
    const scroller = scrollParent(element);
    const pending = { frame: 0 };
    const place = () => {
      pending.frame = 0;
      const top = row.getBoundingClientRect().top;
      const scrollerTop = scroller ? scroller.getBoundingClientRect().top : 0;
      const room = scroller ? scroller.clientHeight : window.innerHeight;
      const height = Math.max(0, room - Math.max(0, top - scrollerTop));
      element.style.setProperty("--work-pane-height", `${Math.round(height)}px`);
    };
    const schedule = () => {
      if (pending.frame === 0) pending.frame = requestAnimationFrame(place);
    };

    place();
    const target: HTMLElement | Window = scroller ?? window;
    target.addEventListener("scroll", schedule, { passive: true });
    window.addEventListener("resize", schedule);
    const observer = new ResizeObserver(schedule);
    observer.observe(row);
    if (scroller) {
      observer.observe(scroller);
      if (scroller.firstElementChild) observer.observe(scroller.firstElementChild);
    }
    return () => {
      cancelAnimationFrame(pending.frame);
      target.removeEventListener("scroll", schedule);
      window.removeEventListener("resize", schedule);
      observer.disconnect();
    };
  }, [ref, active]);
}

/** Under this many pixels of row width the pane lies over the content instead of beside it. */
export const WORK_PANE_SHEET_BELOW = 1100;

const PANE_WIDTH = "26rem";

/**
 * The row a page's content and its work pane share. Put the content in it; the pane opens beside
 * it by width and narrows it, sticky under whatever has scrolled away and ending at the bottom of
 * the viewport. Under 1100px of row width, measured, the pane lies over the content as a sheet
 * with a scrim.
 *
 * Neither the row nor the box around the content may become a containing block for fixed
 * descendants (no `container-type`, `contain`, `transform`, `filter`, `perspective`,
 * `backdrop-filter` or `will-change` of them): a tab's save dock and the documents upload panel
 * are fixed to the viewport and would be pinned to the bottom of the tab instead. That is why
 * the narrow mode comes from the row's measured width and not from a container query.
 */
export function WorkPaneHost({ children, className }: { children: ReactNode; className?: string }) {
  const api = useWorkPaneApi();
  const { activeId } = useWorkPaneState();
  const open = activeId !== null;
  const paneRef = useRef<HTMLElement>(null);
  useFillsScroller(paneRef, open);
  const [rowRef, rowWidth] = useElementWidth<HTMLDivElement>();
  const sheet = rowWidth > 0 && rowWidth < WORK_PANE_SHEET_BELOW;

  return (
    <div
      ref={rowRef}
      data-slot="work-pane-row"
      data-sheet={sheet || undefined}
      className={cn("relative flex min-w-0 items-start overflow-x-clip", className)}
    >
      <div data-slot="work-pane-content" className="min-w-0 flex-1">
        {children}
      </div>
      {open && sheet ? (
        <div
          aria-hidden
          data-slot="work-pane-scrim"
          onClick={api.close}
          className="absolute inset-0 z-20 animate-in bg-scrim-subtle fade-in-0 motion-reduce:animate-none"
        />
      ) : null}
      <SidePane
        ref={paneRef}
        innerRef={api.setHost}
        open={open}
        width={sheet ? `min(${PANE_WIDTH}, ${rowWidth}px)` : PANE_WIDTH}
        data-slot="work-pane"
        className={cn(
          "sticky top-0 z-30 h-[var(--work-pane-height,100dvh)] self-start bg-card",
          sheet && [
            "ml-[calc(-1*var(--side-pane-width))] w-(--side-pane-width) border-border-strong opacity-100",
            "transition-[translate,visibility] duration-320 ease-settle",
            open ? "visible translate-x-0" : "invisible translate-x-full",
          ],
        )}
      />
    </div>
  );
}
