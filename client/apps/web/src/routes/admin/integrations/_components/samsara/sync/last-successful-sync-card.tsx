import { useT } from "@trenova/shared/i18n/use-t";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { formatDurationFromSeconds, formatToUserTimezone } from "@trenova/shared/lib/date";
import { useSamsaraSyncStore } from "@/stores/samsara-sync";

export function LastSuccessfulSyncCard() {
  const t = useT();

  const lastSuccessfulSync = useSamsaraSyncStore.get("lastSuccessfulSync");

  if (!lastSuccessfulSync) {
    return null;
  }

  return (
    <DescriptionList columns={3} className="bg-card rounded-lg border p-3 lg:grid-cols-5">
      <DescriptionItem numeric label={t("Last Successful Sync")}>
        {formatToUserTimezone(lastSuccessfulSync.closedAt)}
      </DescriptionItem>
      <DescriptionItem numeric label={t("Duration")}>
        {formatDurationFromSeconds(lastSuccessfulSync.durationSeconds)}
      </DescriptionItem>
      <DescriptionItem numeric label={t("Workers")}>
        {lastSuccessfulSync.result.activeWorkers}/{lastSuccessfulSync.result.totalWorkers}
      </DescriptionItem>
      <DescriptionItem numeric label={t("Created Drivers")}>
        {lastSuccessfulSync.result.createdDrivers}
      </DescriptionItem>
      <DescriptionItem numeric label={t("Updated Mappings")}>
        {lastSuccessfulSync.result.updatedMappings}
      </DescriptionItem>
    </DescriptionList>
  );
}
