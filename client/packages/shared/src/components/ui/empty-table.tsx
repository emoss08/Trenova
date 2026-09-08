import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { XIcon } from "lucide-react";
import type { ReactNode } from "react";

export type EmptyTableColumn = {
  label: string;
  /** Money and counts sit at the right, the way the real table sets them. */
  numeric?: boolean;
};

const GHOST_ROWS = 4;
const TEXT_WIDTHS = ["w-4/5", "w-3/5", "w-full", "w-1/2"] as const;
const NUMBER_WIDTHS = ["w-14", "w-10", "w-12", "w-16"] as const;

type EmptyTableProps = {
  title: string;
  description: string;
  /** The real table's own headings, so the sketch is that table with nothing in it. */
  columns: readonly EmptyTableColumn[];
  /** Offered when a filter is what emptied the table. */
  onClearFilters?: () => void;
  /** Another way forward, such as an add or import button. */
  action?: ReactNode;
  className?: string;
};

/**
 * A table drawn as what it is: the ruled page with its own column headings
 * and nothing written on the lines. The words underneath say what fills it.
 * The sketch is decoration for the eye only and is hidden from assistive
 * technology; the title, description and action carry the meaning.
 */
export function EmptyTable({
  title,
  description,
  columns,
  onClearFilters,
  action,
  className,
}: EmptyTableProps) {
  const template = columns
    .map((column) => (column.numeric ? "minmax(3.5rem,0.6fr)" : "minmax(0,1fr)"))
    .join(" ");
  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onClearFilters ? (
          <Button variant="outline" size="sm" onClick={onClearFilters}>
            <XIcon className="size-3.5" />
            Clear filters
          </Button>
        ) : (
          action
        )
      }
      sketch={
        <div className="border-border/70 bg-card rounded-md border text-left">
          <div
            className="text-muted-foreground text-2xs grid items-center gap-4 border-b px-3 py-1.5 leading-none"
            style={{ gridTemplateColumns: template }}
          >
            {columns.map((column) => (
              <span key={column.label} className={cn("truncate", column.numeric && "text-right")}>
                {column.label}
              </span>
            ))}
          </div>
          {Array.from({ length: GHOST_ROWS }, (_, row) => (
            <div
              key={row}
              className="border-border/60 grid items-center gap-4 border-b border-dashed px-3 py-2.5 last:border-0"
              style={{ gridTemplateColumns: template }}
            >
              {columns.map((column, index) => (
                <span
                  key={column.label}
                  className={cn("flex", column.numeric ? "justify-end" : "justify-start")}
                >
                  <GhostLine
                    className={
                      column.numeric
                        ? `h-2 ${NUMBER_WIDTHS[(row + index) % NUMBER_WIDTHS.length]}`
                        : index === 0
                          ? `h-2 ${TEXT_WIDTHS[row % TEXT_WIDTHS.length]}`
                          : TEXT_WIDTHS[(row + index) % TEXT_WIDTHS.length]
                    }
                  />
                </span>
              ))}
            </div>
          ))}
        </div>
      }
    />
  );
}
