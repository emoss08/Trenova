import { cn } from "@trenova/shared/lib/utils";
import type { ComponentProps } from "react";

/**
 * A list pane whose list has views: Needs you and its lenses. The page is
 * one frame that stays: the head, with the views on it, is drawn once, and a
 * view brings only what is its own, its list under the head and what the
 * chosen row reads beside it. Going from one view to another changes those
 * two cells and nothing else, so the head never leaves the page, the marker
 * under the views glides, and a view left keeps its place (Kept).
 *
 * The frame is a grid rather than a pane with a page beside it because a view
 * owns a cell in each column: its list and its detail are one component, with
 * one selection and one set of keys.
 *
 * In a narrow window the list and the detail take turns: a view with a row
 * open marks its detail `reading`, and the head and the lists step aside.
 */
export function DeskViews({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "group/views grid h-full min-h-0 grid-cols-[minmax(300px,400px)_minmax(0,1fr)] grid-rows-[auto_minmax(0,1fr)] border-t border-dsk-b-sub max-[900px]:grid-cols-1",
        className,
      )}
      {...props}
    />
  );
}

// i18n-ignore: class names, never shown to a person
const PANE_CELL_CLASS =
  "col-start-1 min-w-0 border-r border-dsk-b-sub bg-dsk-canvas max-[900px]:group-has-[[data-reading]]/views:hidden";

/** The classes that make a `DeskListHead` the frame's head: the top of the list's column. */
// i18n-ignore: class names, never shown to a person
export const DESK_VIEWS_HEAD_CLASS = `${PANE_CELL_CLASS} row-start-1`;

/** A view's list, under the frame's head: its rows, or what stands in their place. */
export function DeskViewList({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(PANE_CELL_CLASS, "relative row-start-2 flex min-h-0 flex-col", className)}
      {...props}
    />
  );
}

/**
 * What a view reads beside its list. `reading` is set while a row is open in
 * the view in front, which gives a narrow window to the detail.
 */
export function DeskViewDetail({
  reading,
  className,
  ...props
}: ComponentProps<"div"> & { reading: boolean }) {
  return (
    <div
      data-reading={reading ? "" : undefined}
      className={cn(
        "col-start-2 row-span-2 row-start-1 min-h-0 min-w-0 overflow-auto max-[900px]:col-start-1 max-[900px]:[&:not([data-reading])]:hidden",
        className,
      )}
      {...props}
    />
  );
}
