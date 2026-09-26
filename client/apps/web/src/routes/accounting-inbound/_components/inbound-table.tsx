import { DataTable } from "@/components/data-table/data-table";
import { useAccountingInboundLabels } from "@/hooks/use-accounting-inbound-labels";
import {
  ACCOUNTING_INBOUND_TABLE_KEY,
  createAccountingInboundTableGraphQLConfig,
  type AccountingInboundRow,
} from "@/lib/graphql/accounting-inbound-table";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useCallback, useMemo } from "react";
import { getInboundColumns } from "./inbound-columns";
import { InboundChangePanel } from "./inbound-panel";

export default function InboundTable({
  system,
  providerName,
}: {
  system: AccountingSystem;
  providerName: string;
}) {
  const t = useT();
  const labels = useAccountingInboundLabels();
  const columns = useMemo(() => getInboundColumns(t, labels), [t, labels]);
  const graphql = useMemo(() => createAccountingInboundTableGraphQLConfig(system), [system]);
  const Panel = useCallback(
    (props: DataTablePanelProps<AccountingInboundRow>) => (
      <InboundChangePanel {...props} providerName={providerName} />
    ),
    [providerName],
  );

  return (
    <DataTable<AccountingInboundRow>
      name="Payment from the books"
      queryKey={ACCOUNTING_INBOUND_TABLE_KEY}
      graphql={graphql}
      resource={Resource.AccountingSync}
      columns={columns}
      TablePanel={Panel}
      enableCreateAction={false}
      enableReadOnlyPanel
    />
  );
}
