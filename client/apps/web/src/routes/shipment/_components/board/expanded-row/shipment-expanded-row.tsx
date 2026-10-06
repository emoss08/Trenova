import { useT } from "@trenova/shared/i18n/use-t";
import { ChevronUpIcon } from "@trenova/shared/components/icons";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import type { Row } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { lazy, Suspense } from "react";
import { DocPills } from "./doc-pills";
import { MoneyLine } from "./money-line";
import { NextStep } from "./next-step";
import { QuickActions } from "./quick-actions";
import { RouteTrack } from "./route-track";
import { useShipmentRecordActions } from "../record-actions";

const TelematicsFormsBlock = lazy(() =>
  import("./telematics-forms-block").then((module) => ({ default: module.TelematicsFormsBlock })),
);

type ShipmentExpandedRowProps = {
  row: Row<Shipment>;
  onCollapse: () => void;
};

/**
 * The open row: where the load is and what it is worth on the left, the one
 * next step and the everyday actions on the right. No inner cards; the parts
 * are separated by hairlines on the row's own tint.
 */
export function ShipmentExpandedRow({ row, onCollapse }: ShipmentExpandedRowProps) {
  const t = useT();
  const shipment = row.original;
  const { rowActions, addComment } = useShipmentRecordActions();

  return (
    <div className="flex flex-col">
      <div className="grid grid-cols-1 @[900px]/board:grid-cols-[minmax(0,1fr)_340px]">
        <div className="flex min-w-0 flex-col gap-4 px-5 py-4">
          <RouteTrack shipment={shipment} />
          <div className="border-border flex flex-col gap-3 border-t border-dashed pt-3 @[1100px]/board:flex-row @[1100px]/board:items-end @[1100px]/board:justify-between">
            <MoneyLine shipment={shipment} />
            <DocPills shipment={shipment} />
          </div>
          {shipment.id ? (
            <Suspense fallback={null}>
              <TelematicsFormsBlock shipmentId={shipment.id} />
            </Suspense>
          ) : null}
        </div>
        <div className="border-border flex flex-col gap-4 border-t px-5 py-4 @[900px]/board:border-t-0 @[900px]/board:border-l">
          <NextStep shipment={shipment} />
          <QuickActions row={row} actions={rowActions} onAddComment={addComment} />
        </div>
      </div>
      <button
        type="button"
        onClick={onCollapse}
        className="ui-focus-ring text-muted-foreground hover:text-foreground mx-auto mb-2 inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs"
      >
        <ChevronUpIcon className="size-3" aria-hidden />
        {t("Collapse")}
        <Kbd>Esc</Kbd>
      </button>
    </div>
  );
}
