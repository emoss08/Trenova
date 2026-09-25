import { DataTable } from "@/components/data-table/data-table";
import { useAccountingSyncLabels } from "@/hooks/use-accounting-sync-labels";
import {
  ACCOUNTING_SYNC_LEDGER_KEY,
  createAccountingSyncLedgerTableGraphQLConfig,
  type AccountingSyncLedgerRow,
} from "@/lib/graphql/accounting-sync-ledger-table";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useCallback, useMemo } from "react";
import { getLedgerColumns } from "./ledger-columns";
import { LedgerRecordPanel } from "./ledger-record-panel";

export default function LedgerTable({
  system,
  providerName,
}: {
  system: AccountingSystem;
  providerName: string;
}) {
  const t = useT();
  const labels = useAccountingSyncLabels();
  const columns = useMemo(() => getLedgerColumns(t, labels), [t, labels]);
  const graphql = useMemo(() => createAccountingSyncLedgerTableGraphQLConfig(system), [system]);
  const Panel = useCallback(
    (props: DataTablePanelProps<AccountingSyncLedgerRow>) => (
      <LedgerRecordPanel {...props} system={system} providerName={providerName} />
    ),
    [system, providerName],
  );

  return (
    <DataTable<AccountingSyncLedgerRow>
      name="Sync record"
      queryKey={ACCOUNTING_SYNC_LEDGER_KEY}
      graphql={graphql}
      resource={Resource.AccountingSync}
      columns={columns}
      TablePanel={Panel}
      enableCreateAction={false}
      enableReadOnlyPanel
    />
  );
}
