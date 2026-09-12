import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { PoliciesSkeleton } from "./_components/policies-skeleton";

const PoliciesConsole = lazy(() => import("./_components/policies-console"));

export function PoliciesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Policies"),
        description: t(
          "Handbooks and policies people sign from Dash. A signature records the version it was given for, so revising the words means giving them a new version — and everybody bound by it goes back to outstanding.",
        ),
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent fallback={<PoliciesSkeleton />}>
          <PoliciesConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
