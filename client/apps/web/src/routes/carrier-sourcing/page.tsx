import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { CarrierSourcingWorkspace } from "./_components/carrier-sourcing-workspace";

export function CarrierSourcingPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier sourcing"),
        description: t("Find carriers, vet them against your rules and import the ones you want"),
      }}
    >
      <CarrierSourcingWorkspace />
    </PageLayout>
  );
}
