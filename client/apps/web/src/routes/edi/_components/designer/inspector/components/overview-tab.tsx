import type { EDIX12Inspection } from "@trenova/shared/types/edi";
import type { InspectorContext } from "../inspector-context";
import InspectorGrid from "./inspector-grid";

export default function OverviewTab({
  context,
  inspection,
}: {
  context: InspectorContext;
  inspection?: EDIX12Inspection;
}) {
  return (
    <InspectorGrid
      rows={[
        ...context.overviewRows,
        ["Segments", String(inspection?.summary.segmentCount ?? 0)],
        ["Transactions", String(inspection?.summary.transactionCount ?? 0)],
        ["Diagnostics", String(inspection?.diagnostics.length ?? 0)],
      ]}
    />
  );
}
