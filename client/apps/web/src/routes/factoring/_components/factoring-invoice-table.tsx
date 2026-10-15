import { DataTable } from "@/components/data-table/data-table";
import {
  createFactoringInvoiceStateTableConfig,
  FACTORING_INVOICE_STATE_KEY,
  type FactoringInvoiceStateRow,
} from "@/lib/graphql/factoring";
import { usePermission } from "@/hooks/use-permission";
import { Send01Icon } from "@trenova/shared/components/icons";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { factoringTabVerdicts, type FactoringQueueTab } from "../use-factoring-state";
import { getFactoringInvoiceColumns } from "./factoring-invoice-columns";
import { FactoringInvoicePanel } from "./factoring-invoice-panel";
import { SubmitToFactorDialog } from "./submit-to-factor-dialog";

const EMPTY_TITLES = defineLabels({
  ready: "No invoices are ready to factor",
  "not-ready": "No invoices are waiting on a fix",
  submitted: "No invoices are with a factor",
  funded: "No funded invoices are waiting on their reserve",
});

export default function FactoringInvoiceTable({ tab }: { tab: FactoringQueueTab }) {
  const t = useT();
  const { allowed: canSubmit } = usePermission(Resource.FactoringSubmission, Operation.Submit);
  const [selected, setSelected] = useState<FactoringInvoiceStateRow[] | null>(null);
  const columns = useMemo(() => getFactoringInvoiceColumns(t, tab), [t, tab]);
  const graphql = useMemo(
    () => createFactoringInvoiceStateTableConfig(factoringTabVerdicts(tab)),
    [tab],
  );
  const selectable = tab === "ready" && canSubmit;
  const dockActions = useMemo<DockAction<FactoringInvoiceStateRow>[]>(
    () =>
      selectable
        ? [
            {
              id: "submit-to-factor",
              label: t("Submit to factor"),
              icon: Send01Icon,
              onClick: (rows) => setSelected(rows),
            },
          ]
        : [],
    [selectable, t],
  );

  return (
    <>
      <DataTable<FactoringInvoiceStateRow>
        name="Factoring invoice"
        emptyTitle={EMPTY_TITLES[tab]}
        queryKey={FACTORING_INVOICE_STATE_KEY}
        graphql={graphql}
        resource={Resource.Invoice}
        columns={columns}
        TablePanel={FactoringInvoicePanel}
        enableCreateAction={false}
        enableReadOnlyPanel
        enableRowSelection={selectable}
        dockActions={dockActions}
      />
      {selected ? (
        <SubmitToFactorDialog
          open
          rows={selected}
          onOpenChange={(open) => {
            if (!open) setSelected(null);
          }}
        />
      ) : null}
    </>
  );
}
