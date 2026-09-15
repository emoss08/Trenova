import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { invoicePanelPath } from "@/lib/invoice-links";
import {
  invoiceRegisterTableGraphQLConfig,
  type InvoiceRegisterRow,
} from "@/lib/graphql/invoice-table";
import { InvoiceVoidDialog } from "@/routes/invoice/_components/invoice-void-dialog";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { BanIcon, ExternalLinkIcon, TruckIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { shipmentPanelPath } from "@/lib/shipment-utils";
import { getColumns } from "./invoice-register-columns";

export default function InvoiceRegisterTable() {
  const t = useT();
  const navigate = useNavigate();
  const columns = useMemo(() => getColumns(t), [t]);

  const [voiding, setVoiding] = useState<InvoiceRegisterRow | null>(null);

  const openInvoice = useCallback(
    (row: InvoiceRegisterRow) => {
      void navigate(invoicePanelPath(row.id));
    },
    [navigate],
  );

  const contextMenuActions = useMemo<RowAction<InvoiceRegisterRow>[]>(
    () => [
      {
        id: "open",
        label: t("Open in workspace"),
        icon: ExternalLinkIcon,
        onClick: (row) => openInvoice(row.original),
      },
      {
        id: "shipment",
        label: t("View shipment"),
        icon: TruckIcon,
        hidden: (row) => !row.original.shipmentId,
        onClick: (row) => {
          if (row.original.shipmentId) {
            window.open(shipmentPanelPath(row.original.shipmentId), "_blank");
          }
        },
      },
      {
        id: "void",
        label: t("Void"),
        icon: BanIcon,
        variant: "destructive",
        // An invoice with cash applied cannot be voided; the dialog explains
        // the rest once the AR context loads.
        hidden: (row) => row.original.status === "Voided",
        disabled: (row) => Number(row.original.appliedAmount ?? 0) > 0,
        onClick: (row) => setVoiding(row.original),
      },
    ],
    [t, openInvoice],
  );

  return (
    <>
      <DataTable<InvoiceRegisterRow>
        name="Invoice"
        queryKey="invoice-register"
        graphql={invoiceRegisterTableGraphQLConfig}
        resource={Resource.Invoice}
        columns={columns}
        enableCreateAction={false}
        onRowClick={openInvoice}
        contextMenuActions={contextMenuActions}
      />
      {voiding ? (
        <InvoiceVoidDialog
          invoice={voiding}
          open
          onOpenChange={(open) => {
            if (!open) setVoiding(null);
          }}
        />
      ) : null}
    </>
  );
}
