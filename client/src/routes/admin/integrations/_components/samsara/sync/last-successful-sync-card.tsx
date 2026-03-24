import { formatDurationFromSeconds, formatToUserTimezone } from "@/lib/date";
import { useSamsaraSyncStore } from "@/stores/samsara-sync";

export function LastSuccessfulSyncCard() {
  const lastSuccessfulSync = useSamsaraSyncStore.get("lastSuccessfulSync");

  if (!lastSuccessfulSync) {
    return null;
  }

  return (
    <div className="grid gap-2 rounded-md border border-emerald-500/30 bg-emerald-500/5 p-3 text-xs sm:grid-cols-2 lg:grid-cols-6">
      <ContentSection title="Last Successful Sync">
        {formatToUserTimezone(lastSuccessfulSync.closedAt)}
      </ContentSection>
      <ContentSection title="Duration">
        {formatDurationFromSeconds(lastSuccessfulSync.durationSeconds)}
      </ContentSection>
      <ContentSection title="Workers">
        {lastSuccessfulSync.result.activeWorkers}/{lastSuccessfulSync.result.totalWorkers}
      </ContentSection>
      <ContentSection title="Created Drivers">
        {lastSuccessfulSync.result.createdDrivers}
      </ContentSection>
      <ContentSection title="Updated Mappings">
        {lastSuccessfulSync.result.updatedMappings}
      </ContentSection>
    </div>
  );
}

function ContentSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-md border border-emerald-600/20 bg-background/70 p-3">
      <p className="text-muted-foreground">{title}</p>
      <p className="mt-1 font-semibold text-foreground">{children}</p>
    </div>
  );
}
