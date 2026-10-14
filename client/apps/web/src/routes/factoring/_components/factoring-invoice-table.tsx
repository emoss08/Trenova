import { DataTable } from "@/components/data-table/data-table";
import {
  createFactoringInvoiceStateTableConfig,
  FACTORING_INVOICE_STATE_KEY,
  type FactoringInvoiceStateRow,
} from "@/lib/graphql/factoring";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { factoringTabVerdicts, type FactoringTab } from "../use-factoring-state";
import { getFactoringInvoiceColumns } from "./factoring-invoice-columns";
import { FactoringInvoicePanel } from "./factoring-invoice-panel";

export default function FactoringInvoiceTable({ tab }: { tab: FactoringTab }) {
  const t = useT();
  const columns = useMemo(() => getFactoringInvoiceColumns(t, tab), [t, tab]);
  const graphql = useMemo(
    () => createFactoringInvoiceStateTableConfig(factoringTabVerdicts(tab)),
    [tab],
  );

  return (
    <DataTable<FactoringInvoiceStateRow>
      name="Factoring invoice"
      emptyTitle={
        tab === "ready"
          ? t("No invoices are ready to factor")
          : t("No invoices are waiting on a fix")
      }
      queryKey={FACTORING_INVOICE_STATE_KEY}
      graphql={graphql}
      resource={Resource.Invoice}
      columns={columns}
      TablePanel={FactoringInvoicePanel}
      enableCreateAction={false}
      enableReadOnlyPanel
    />
  );
}
