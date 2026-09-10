import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/fuel-card-table"));

export function UnassignedFuelCardsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Unassigned Cards",
        description:
          "Cards a connected fuel card feed saw a transaction on before anybody had registered them. Until a card is tied to a tractor or driver, its purchases can only be matched by the unit number on the receipt.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <Table unassignedOnly />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}

export function FuelCardsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Fuel Cards",
        description:
          "The cards drivers fuel with — which provider issued each one, who carries it, and which unit it is tied to. Imported statements are matched to purchases through these cards.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
