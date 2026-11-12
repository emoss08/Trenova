import { VirtualRows, type VirtualRowsProps } from "@/components/virtual-rows";
import { cn } from "@trenova/shared/lib/utils";
import type { ComponentProps, ReactNode } from "react";

/**
 * The Desk's list pane: the column on the left of a page that lists things
 * and reads the chosen one beside it. Needs you (and so Watchtower, Decisions
 * and the cases they open) and Triage's queue are this one pane, so the frame,
 * the scroll area, the group headings, the hover and the chosen row are drawn
 * once. A page brings what is its own: its head (title, filters, search) and
 * what a row says. A new page with a list in this place starts here.
 */
export function DeskListPane({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex min-h-0 min-w-0 flex-col border-r border-dsk-b-sub bg-dsk-canvas",
        className,
      )}
      {...props}
    />
  );
}

/** The gutter every list in the pane keeps, so rows, states and skeletons line up. */
// i18n-ignore: class names, never shown to a person
export const DESK_LIST_GUTTER_CLASS = "px-2 pb-4";

/**
 * Something that stands in the list's place and does not scroll as rows do:
 * its empty state, its error, its skeleton.
 */
export function DeskListStill({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("min-h-0 flex-1 overflow-auto", DESK_LIST_GUTTER_CLASS, className)}
      {...props}
    />
  );
}

export type DeskListProps = Omit<
  VirtualRowsProps,
  "mask" | "maskClassName" | "hoverGlide" | "glideClassName" | "contentClassName"
>;

/**
 * The pane's rows: only those near the viewport are mounted, in the Desk's
 * scroll area, which fades at an edge with more beyond it, under one hover
 * that glides from row to row.
 */
export function DeskList({ className, ...props }: DeskListProps) {
  return (
    <VirtualRows
      {...props}
      className={cn("min-h-0 flex-1", className)}
      contentClassName={DESK_LIST_GUTTER_CLASS}
      mask="background"
      maskClassName="before:from-dsk-canvas after:from-dsk-canvas"
      maskHeight={14}
      hoverGlide
      glideClassName="rounded-[8px] bg-dsk-hover"
    />
  );
}

/** A group's heading in the list: a lane of Needs you, a day or "Over target" in Triage. */
export function DeskListHeading({
  tone = "quiet",
  children,
}: {
  tone?: "quiet" | "danger";
  children: ReactNode;
}) {
  return (
    <div
      role="presentation"
      className={cn(
        "bg-dsk-canvas px-3 pt-3.5 pb-1.5 text-xs",
        tone === "danger" ? "text-dsk-error" : "text-dsk-faint",
      )}
    >
      {children}
    </div>
  );
}

/**
 * What makes an element one of the list's rows: the glide lights it on hover,
 * and the chosen one holds the Desk's active surface. Spread on the row's own
 * element, beside the classes that lay out what the row says.
 */
export function deskListRow(chosen: boolean): { "data-glide-target": true; className: string } {
  return {
    "data-glide-target": true,
    className: cn("relative rounded-[8px] text-left", chosen && "bg-(--dsk-active)"),
  };
}
