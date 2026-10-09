import { TableCell, TableRow } from "@trenova/shared/components/ui/table";
import type { ReactNode } from "react";

type DataTableExpandedRowProps = {
  rowId: string;
  colSpan: number;
  children: ReactNode;
  /** Set when the table draws only the rows in view, so the window can measure this one. */
  measureRef?: (element: Element | null) => void;
  virtualIndex?: number;
};

/**
 * The open row's panel. It is pinned to the left edge and sized to the
 * visible width of the table, so a wide table scrolls its columns sideways
 * while the panel stays put.
 */
export function DataTableExpandedRow({
  rowId,
  colSpan,
  children,
  measureRef,
  virtualIndex,
}: DataTableExpandedRowProps) {
  return (
    <TableRow
      ref={measureRef}
      data-index={virtualIndex}
      data-expanded-for={rowId}
      className="bg-field hover:bg-field h-auto"
    >
      <TableCell colSpan={colSpan} className="border-border border-b p-0 whitespace-normal">
        <div className="animate-expand-in sticky left-0 w-(--dt-viewport-w,100%) min-w-0">
          {children}
        </div>
      </TableCell>
    </TableRow>
  );
}
