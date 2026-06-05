import type { RowAction } from "@/types/data-table";
import type { Shipment } from "@/types/shipment";
import type { Row } from "@tanstack/react-table";
import { lazy, Suspense, type ReactNode } from "react";
import type { ShipmentDocumentUploadContext } from "./expanded-row/document-stack";
import { PanelSkeleton } from "./expanded-row/panel-skeletons";

const RouteTimelineBlock = lazy(() => import("./expanded-row/route-timeline-block"));
const FinancialsBlock = lazy(() => import("./expanded-row/financials-block"));
const DocumentsBlock = lazy(() =>
  import("./expanded-row/document-stack").then((m) => ({ default: m.DocumentsBlock })),
);
const QuickActionsBlock = lazy(() => import("./expanded-row/quick-actions-block"));

function PanelSection({
  title,
  fallback,
  children,
}: {
  title: string;
  fallback: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="min-w-0">
      <h4 className="cc-label mb-1.5">{title}</h4>
      <Suspense fallback={fallback}>{children}</Suspense>
    </section>
  );
}

export function ExpandedRow({
  row,
  shipment,
  rowActions,
  onUploadDocument,
}: {
  row: Row<Shipment>;
  shipment: Shipment;
  rowActions: RowAction<Shipment>[];
  onUploadDocument: (shipment: Shipment, context?: ShipmentDocumentUploadContext) => void;
}) {
  const stops = shipment.moves?.flatMap((m) => m.stops ?? []) ?? [];

  return (
    <div className="grid grid-cols-1 gap-5 px-4 py-3 md:grid-cols-[2fr_1.4fr_1fr_1fr]">
      <PanelSection title="Route timeline" fallback={<PanelSkeleton />}>
        <RouteTimelineBlock stops={stops} />
      </PanelSection>
      <PanelSection title="Financials" fallback={<PanelSkeleton />}>
        <FinancialsBlock shipment={shipment} />
      </PanelSection>
      <PanelSection title="Documents" fallback={<PanelSkeleton />}>
        <DocumentsBlock shipment={shipment} onUpload={onUploadDocument} />
      </PanelSection>
      <PanelSection title="Quick actions" fallback={<PanelSkeleton />}>
        <QuickActionsBlock row={row} actions={rowActions} />
      </PanelSection>
    </div>
  );
}

export default ExpandedRow;
