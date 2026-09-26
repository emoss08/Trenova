import { DataTable } from "@/components/data-table/data-table";
import { useAccountingDriftLabels } from "@/hooks/use-accounting-drift-labels";
import { useAccountingSyncLabels } from "@/hooks/use-accounting-sync-labels";
import {
  ACCOUNTING_DRIFT_TABLE_KEY,
  createAccountingDriftTableGraphQLConfig,
  type AccountingDriftRow,
} from "@/lib/graphql/accounting-drift-table";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useCallback, useMemo } from "react";
import { getDriftColumns } from "./drift-columns";
import { DriftFindingPanel } from "./drift-panel";

export default function DriftTable({
  system,
  providerName,
  toleranceMinor,
}: {
  system: AccountingSystem;
  providerName: string;
  toleranceMinor: number;
}) {
  const t = useT();
  const labels = useAccountingDriftLabels();
  const syncLabels = useAccountingSyncLabels();
  const columns = useMemo(
    () => getDriftColumns(t, labels, syncLabels.objectType, toleranceMinor),
    [t, labels, syncLabels.objectType, toleranceMinor],
  );
  const graphql = useMemo(() => createAccountingDriftTableGraphQLConfig(system), [system]);
  const Panel = useCallback(
    (props: DataTablePanelProps<AccountingDriftRow>) => (
      <DriftFindingPanel {...props} providerName={providerName} toleranceMinor={toleranceMinor} />
    ),
    [providerName, toleranceMinor],
  );

  return (
    <DataTable<AccountingDriftRow>
      name="Drift finding"
      queryKey={ACCOUNTING_DRIFT_TABLE_KEY}
      graphql={graphql}
      resource={Resource.AccountingSync}
      columns={columns}
      TablePanel={Panel}
      enableCreateAction={false}
      enableReadOnlyPanel
    />
  );
}
