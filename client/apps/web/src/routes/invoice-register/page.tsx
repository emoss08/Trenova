import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/invoice-register-table"));

/**
 * Every invoice the organization has issued, as a ledger rather than a work
 * queue: sortable and filterable by payer, date and settlement, with the
 * shipper named on invoices that bill someone else's freight.
 */
export function InvoiceRegisterPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Invoice register"),
        description: t(
          "Every invoice issued, with its bill-to, shipper, dates and settlement status.",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
