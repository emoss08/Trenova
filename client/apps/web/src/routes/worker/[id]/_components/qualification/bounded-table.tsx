import { HoverGlideGroup } from "@/components/hover-glide";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

export type BoundedColumn = {
  key: string;
  label: ReactNode;
  align?: "start" | "end" | "center";
  /** Takes only the width its content needs (a menu, a mark). */
  shrink?: boolean;
};

const ALIGN = { start: "text-left", end: "text-right", center: "text-center" } as const;

/**
 * A table in a bounded region of its own: the column heads stay pinned while the rows scroll
 * under them, and one hover plate glides between rows. A row is a `tr` with `data-glide-target`
 * whose cells wear `BOUNDED_CELL_CLASS`.
 */
export function BoundedTable({
  columns,
  label,
  maxHeightClassName,
  children,
}: {
  columns: readonly BoundedColumn[];
  label: string;
  /** The region's height cap, as a Tailwind class (`max-h-152`). */
  maxHeightClassName: string;
  children: ReactNode;
}) {
  return (
    <ScrollArea
      className={cn("min-h-0", maxHeightClassName)}
      maskVariant="card"
      maskClassName="top-8"
    >
      <HoverGlideGroup className="bg-card" glideClassName="rounded-none">
        <table aria-label={label} className="w-full border-separate border-spacing-0 text-sm">
          <thead className="sticky top-0 z-10">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={cn(
                    "h-8 border-b border-border-subtle bg-sunken px-3 text-xs font-medium whitespace-nowrap text-foreground-subtle first:pl-4",
                    ALIGN[column.align ?? "start"],
                    column.shrink && "w-px",
                  )}
                >
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="[&>tr:last-child>td]:border-b-0">{children}</tbody>
        </table>
      </HoverGlideGroup>
    </ScrollArea>
  );
}

/** A group's head row inside a `BoundedTable` body ("Required 8"). */
export function BoundedGroupRow({
  span,
  label,
  count,
}: {
  span: number;
  label: string;
  count: number;
}) {
  return (
    <tr>
      <td
        colSpan={span}
        className="border-b border-border-subtle bg-card px-4 pt-2.5 pb-1.5 text-xs font-medium"
      >
        {label}
        <span className="ml-1 font-normal text-foreground-subtle tabular-nums">{count}</span>
      </td>
    </tr>
  );
}

/** The cell class every `BoundedTable` cell shares. */
export const BOUNDED_CELL_CLASS =
  "border-b border-border-subtle px-3 py-2 align-middle first:pl-4 last:pr-3";
