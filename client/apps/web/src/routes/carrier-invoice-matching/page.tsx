import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { MatchingWorkspaceSkeleton } from "./_components/matching-workspace-skeleton";

const MatchingWorkspace = lazy(() => import("./_components/matching-workspace"));

export function CarrierInvoiceMatchingPage() {
  const t = useT();

  return (
    <PageLayout
      fill
      pageHeaderProps={{
        title: t("Carrier invoice matching"),
        description: t(
          "Reconcile inbound carrier freight invoices against negotiated buy rates — link carriers, create matches, and resolve variances.",
        ),
      }}
    >
      <LazyComponent fallback={<MatchingWorkspaceSkeleton />}>
        <MatchingWorkspace />
      </LazyComponent>
    </PageLayout>
  );
}
