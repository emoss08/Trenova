import { PageLayout } from "@/components/navigation/sidebar-layout";
import { dataRetention } from "@/lib/queries/data-retention";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { DataRetentionSkeleton } from "./skeleton";

const DataRetentionForm = lazy(() => import("./_components/data-retention-form"));

export const prefetch: RoutePrefetch = () => [dataRetention.get()];

export function DataRetentionPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Data retention"),
        description: t(
          "How long each kind of record is kept before the nightly purge jobs remove it.",
        ),
      }}
    >
      <LazyComponent fallback={<DataRetentionSkeleton />}>
        <DataRetentionForm />
      </LazyComponent>
    </PageLayout>
  );
}
