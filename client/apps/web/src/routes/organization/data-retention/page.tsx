import { SettingsPage } from "@/components/settings/settings-page";
import { dataRetention } from "@/lib/queries/data-retention";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { DataRetentionSkeleton } from "./skeleton";

const DataRetentionForm = lazy(() => import("./_components/data-retention-form"));

export const prefetch: RoutePrefetch = () => [dataRetention.get()];

export function DataRetentionPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Data retention")}
      description={t(
        "How long each kind of record is kept. Anything older is removed for good by the nightly purges.",
      )}
      skeleton={<DataRetentionSkeleton />}
    >
      <DataRetentionForm />
    </SettingsPage>
  );
}
