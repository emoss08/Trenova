import {
  DENSITY_COMPACT_CELLS,
  DENSITY_COMPACT_ROW,
  TABLE_BODY_CELL_CLASS,
  TABLE_CONTAINER_CLASS,
  TABLE_HEAD_CELL_CLASS,
  TABLE_PAGER_CLASS,
  TABLE_PANEL_CLASS,
  TABLE_PANEL_CONTENT_CLASS,
  TABLE_SHELL_CLASS,
  TABLE_TOOLBAR_CLASS,
  TABLE_VIEWPORT_CLASS,
} from "@/components/data-table/data-table-chrome";
import { DataTableGroupHeaderSkeleton } from "@/components/data-table/data-table-group-header";
import { SwitchVertical01Icon } from "@trenova/shared/components/icons";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { BOARD_CELL_SKELETONS } from "./board-cell-skeletons";
import { useShipmentBoardUrl } from "./url-state";
import { ShipmentTimelineSkeleton } from "./views/shipment-timeline-skeleton";

/** The table's first page; a row past the fold costs nothing and keeps the scrollbar honest. */
const SKELETON_ROWS = 25;
const SELECT_COLUMN_SIZE = 40;

export type BoardSkeletonColumn = {
  id: string;
  label: string;
  size: number;
  sortable: boolean;
};

/**
 * The columns a board opens with, in order. The columns themselves are too heavy to import
 * ahead of the board's chunk, so a test holds this list to them.
 */
export function boardSkeletonColumns(t: TranslateFn): BoardSkeletonColumn[] {
  return [
    { id: "lane", label: t("Lane"), size: 280, sortable: false },
    { id: "status", label: t("Status"), size: 160, sortable: true },
    { id: "tenderStatus", label: t("Tender"), size: 130, sortable: true },
    { id: "billing", label: t("Billing"), size: 140, sortable: true },
    { id: "proBol", label: t("PRO / BOL"), size: 160, sortable: true },
    { id: "order", label: t("Order"), size: 130, sortable: false },
    { id: "customer", label: t("Customer"), size: 220, sortable: true },
    { id: "driver", label: t("Coverage"), size: 200, sortable: false },
    { id: "eta", label: t("ETA"), size: 150, sortable: true },
    { id: "revenue", label: t("Revenue"), size: 180, sortable: true },
    { id: "margin", label: t("Margin"), size: 120, sortable: false },
    { id: "actions", label: "", size: 72, sortable: false },
  ];
}

/** The toolbar's controls at their sizes; labels give way to icons on a narrow board. */
function BoardToolbarSkeleton({ showGroupMenu }: { showGroupMenu: boolean }) {
  return (
    <div aria-hidden className={TABLE_TOOLBAR_CLASS}>
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <Skeleton className="h-7 w-48" />
        <Skeleton className="h-7 w-7 @[1180px]/board:w-16.75" />
        <Skeleton className="h-7 w-7 @[1180px]/board:w-15.5" />
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <Skeleton className="h-7 w-24 rounded-lg @[1180px]/board:w-56" />
        {showGroupMenu ? <Skeleton className="h-7 w-7 @[1180px]/board:w-18.75" /> : null}
        <Skeleton className="h-7 w-8.5" />
        <Skeleton className="hidden h-7 w-7 @[1000px]/board:block @[1180px]/board:w-19.75" />
        <Skeleton className="hidden h-7 w-7 @[1000px]/board:block @[1180px]/board:w-17.5" />
        <Skeleton className="h-7 w-8" />
      </div>
    </div>
  );
}

function BoardTableSkeleton({ grouped }: { grouped: boolean }) {
  const t = useT();
  const columns = boardSkeletonColumns(t);
  const tableWidth = columns.reduce((total, column) => total + column.size, SELECT_COLUMN_SIZE);

  return (
    <div className={cn(TABLE_VIEWPORT_CLASS, "[--dt-viewport-w:100cqw]")}>
      <Table
        data-density="compact"
        style={{ minWidth: tableWidth }}
        className={cn(
          "border-separate border-spacing-0",
          DENSITY_COMPACT_ROW,
          DENSITY_COMPACT_CELLS,
        )}
        containerClassName={TABLE_CONTAINER_CLASS}
      >
        <TableHeader className="sticky top-0 z-20">
          <TableRow className="hover:bg-transparent">
            <TableHead
              className={TABLE_HEAD_CELL_CLASS}
              style={{ width: SELECT_COLUMN_SIZE, maxWidth: SELECT_COLUMN_SIZE }}
            >
              <span className="border-border-strong block size-4 translate-y-0.5 rounded-sm border" />
            </TableHead>
            {columns.map((column) => (
              <TableHead
                key={column.id}
                className={TABLE_HEAD_CELL_CLASS}
                style={{ width: column.size, maxWidth: column.size }}
              >
                {column.sortable ? (
                  <span className="text-foreground-subtle flex items-center gap-1.5">
                    {column.label}
                    <SwitchVertical01Icon className="size-3.5" />
                  </span>
                ) : (
                  column.label
                )}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {grouped ? <DataTableGroupHeaderSkeleton colSpan={columns.length + 1} /> : null}
          {Array.from({ length: SKELETON_ROWS }, (_, rowIndex) => (
            <TableRow key={rowIndex} className="hover:bg-transparent">
              <TableCell
                className={TABLE_BODY_CELL_CLASS}
                style={{ width: SELECT_COLUMN_SIZE, maxWidth: SELECT_COLUMN_SIZE }}
              >
                <span className="border-border-strong block size-4 translate-y-0.5 rounded-sm border" />
              </TableCell>
              {columns.map((column) => (
                <TableCell
                  key={column.id}
                  className={TABLE_BODY_CELL_CLASS}
                  style={{ width: column.size, maxWidth: column.size }}
                >
                  {BOARD_CELL_SKELETONS[column.id]}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function BoardPagerSkeleton() {
  return (
    <div aria-hidden className={TABLE_PAGER_CLASS}>
      <div className="flex items-center gap-2">
        <Skeleton className="size-7" />
        <Skeleton className="h-3.5 w-60" />
      </div>
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <Skeleton className="h-3.5 w-20" />
          <Skeleton className="h-7 w-15" />
        </div>
        <div className="flex items-center gap-1">
          <Skeleton className="size-7" />
          <Skeleton className="mx-2 h-3.5 w-28.75" />
          <Skeleton className="size-7" />
        </div>
      </div>
    </div>
  );
}

/**
 * The board while its code loads: the real toolbar's shapes, then whichever view the address
 * names. The table draws the frame and cells the real one will, so its rows land in place.
 */
export function ShipmentBoardSkeleton() {
  const t = useT();
  const [{ view, group }] = useShipmentBoardUrl();

  return (
    <div className={TABLE_PANEL_CLASS} aria-busy aria-label={t("Loading data...")}>
      <div className={TABLE_PANEL_CONTENT_CLASS}>
        <div className={TABLE_SHELL_CLASS}>
          <BoardToolbarSkeleton showGroupMenu={view === "table"} />
          {view === "table" ? (
            <>
              <BoardTableSkeleton grouped={group !== "none"} />
              <BoardPagerSkeleton />
            </>
          ) : view === "timeline" ? (
            <div className={TABLE_VIEWPORT_CLASS}>
              <ShipmentTimelineSkeleton />
            </div>
          ) : (
            <Skeleton aria-hidden className={cn(TABLE_VIEWPORT_CLASS, "rounded-none")} />
          )}
        </div>
      </div>
    </div>
  );
}
