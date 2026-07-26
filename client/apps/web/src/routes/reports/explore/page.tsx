import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useCannedReports, useReportDefinition } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { CircleAlertIcon } from "lucide-react";
import { useSearchParams, useParams } from "react-router";
import { ExploreView } from "./_components/explore-view";

function ExploreSkeleton() {
  return (
    <div className="flex h-full flex-col gap-3 p-4">
      <Skeleton className="h-10" />
      <Skeleton className="flex-1" />
    </div>
  );
}

function ExploreError({ error }: { error: unknown }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2">
      <CircleAlertIcon className="size-8 text-destructive" />
      <p className="text-sm text-muted-foreground">
        {graphQLErrorMessage(error, "Failed to load this report")}
      </p>
    </div>
  );
}

function NotFound({ what }: { what: string }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2">
      <CircleAlertIcon className="size-8 text-muted-foreground" />
      <p className="text-sm text-muted-foreground">{what}</p>
    </div>
  );
}

/**
 * Explore opens a saved report, or a canned one via ?canned=<key>, as a live
 * result you can sort, filter, chart, and drill into before exporting.
 */
export function ReportExplorePage() {
  const { definitionId } = useParams<{ definitionId: string }>();
  const [searchParams] = useSearchParams();
  const cannedKey = searchParams.get("canned") ?? undefined;

  const definition = useReportDefinition(definitionId);
  const canned = useCannedReports(!definitionId && Boolean(cannedKey));

  if (definitionId) {
    if (definition.isLoading) return <ExploreSkeleton />;
    if (definition.isError) return <ExploreError error={definition.error} />;
    if (!definition.data) return <NotFound what="This report no longer exists." />;

    return (
      <ExploreView
        key={definition.data.id}
        name={definition.data.name}
        description={definition.data.description}
        definition={definition.data.definition}
        target={{ definitionId: definition.data.id }}
        defaultFormat={definition.data.defaultFormat}
        editHref={`/reports/builder/${definition.data.id}`}
      />
    );
  }

  if (!cannedKey) return <NotFound what="Choose a report to explore." />;
  if (canned.isLoading) return <ExploreSkeleton />;
  if (canned.isError) return <ExploreError error={canned.error} />;

  const entry = canned.data?.find((report) => report.key === cannedKey);
  if (!entry) return <NotFound what="This gallery report no longer exists." />;

  return (
    <ExploreView
      key={entry.key}
      name={entry.name}
      description={entry.description}
      definition={entry.definition}
      target={{ cannedKey: entry.key }}
      defaultFormat={entry.defaultFormat}
    />
  );
}
