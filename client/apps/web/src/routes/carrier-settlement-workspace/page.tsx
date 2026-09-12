import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Workspace = lazy(() => import("./_components/workspace"));

export function CarrierSettlementWorkspacePage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier Settlement Workspace"),
        description: t(
          "Everything needed to run a carrier pay period from one screen — review the queue, process settlements one at a time or in bulk, and keep the AP subledger current.",
        ),
      }}
    >
      <LazyComponent>
        <Workspace />
      </LazyComponent>
    </PageLayout>
  );
}
