import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { GlobeIcon } from "lucide-react";
import { lazy } from "react";

const Table = lazy(() => import("./_components/ifta-tax-rate-table"));

export function IftaTaxRatesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "IFTA Tax Rates",
        description:
          "The per-gallon rates each jurisdiction publishes for a quarter and fuel type, as the IFTA rate matrix lists them. A return cannot be finalized while any of its member lines is missing a rate.",
      }}
    >
      <div className="flex flex-col gap-4">
        <Alert variant="info">
          <GlobeIcon className="size-4" />
          <AlertTitle>Rates are global</AlertTitle>
          <AlertDescription>
            Every organization is taxed at the rate published here, so a change or deletion moves
            the figures on every return for that quarter, not only yours. Enter rates as the matrix
            prints them, in USD per US gallon, and note the source.
          </AlertDescription>
        </Alert>
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
