import { ChevronUpIcon } from "@trenova/shared/components/icons";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import type { Row } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useShipmentRecordActions } from "../record-actions";
import { DocPills } from "./doc-pills";
import { MoneyLine } from "./money-line";
import { MoneyLineSkeleton } from "./money-line-skeleton";
import { NextStep } from "./next-step";
import { QuickActions } from "./quick-actions";
import { RouteTrack } from "./route-track";

type ShipmentExpandedRowProps = {
  row: Row<Shipment>;
  onCollapse: () => void;
};

export function ShipmentExpandedRow({ row, onCollapse }: ShipmentExpandedRowProps) {
  const shipment = row.original;
  const { rowActions, addComment } = useShipmentRecordActions();

  return (
    <div className="flex flex-col">
      <div className="grid grid-cols-1 @[900px]/board:grid-cols-[minmax(0,1fr)_340px] border-b border-border">
        <div className="flex min-w-0 flex-col gap-4 px-5 py-4">
          <RouteTrack shipment={shipment} />
          <div className="border-border flex flex-col gap-3 border-t border-dashed pt-3 @[1100px]/board:flex-row @[1100px]/board:items-end @[1100px]/board:justify-between">
            <LazyComponent fallback={<MoneyLineSkeleton />}>
              <MoneyLine shipment={shipment} />
            </LazyComponent>
            <DocPills shipment={shipment} />
          </div>
        </div>
        <div className="border-border flex flex-col gap-4 border-t px-5 py-4 @[900px]/board:border-t-0 @[900px]/board:border-l">
          <NextStep shipment={shipment} />
          <QuickActions row={row} actions={rowActions} onAddComment={addComment} />
        </div>
      </div>
      <button
        type="button"
        onClick={onCollapse}
        className="ui-focus-ring text-muted-foreground hover:text-foreground items-center text-center gap-1.5 px-2 py-1 text-xs w-full transition-colors cursor-pointer"
      >
        <ChevronUpIcon className="size-3.5 inline-flex" />
      </button>
    </div>
  );
}
