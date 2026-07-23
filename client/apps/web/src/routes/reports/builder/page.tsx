import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useReportCatalog, useReportDefinition } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { CircleAlertIcon } from "lucide-react";
import { useParams } from "react-router";
import { ReportBuilder } from "./_components/report-builder";

function BuilderSkeleton() {
  return (
    <div className="flex h-full flex-col gap-3 p-4">
      <Skeleton className="h-10" />
      <div className="grid flex-1 grid-cols-[280px_1fr_360px] gap-3">
        <Skeleton />
        <Skeleton />
        <Skeleton />
      </div>
    </div>
  );
}

function BuilderError({ error }: { error: unknown }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2">
      <CircleAlertIcon className="size-8 text-destructive" />
      <p className="text-sm text-muted-foreground">
        {graphQLErrorMessage(error, "Failed to load the report builder")}
      </p>
    </div>
  );
}

export function ReportBuilderPage() {
  const { definitionId } = useParams<{ definitionId: string }>();
  const catalog = useReportCatalog();
  const definition = useReportDefinition(definitionId);

  if (catalog.isLoading || (definitionId && definition.isLoading)) {
    return <BuilderSkeleton />;
  }
  if (catalog.isError) {
    return <BuilderError error={catalog.error} />;
  }
  if (definitionId && definition.isError) {
    return <BuilderError error={definition.error} />;
  }
  if (!catalog.data) return null;

  return (
    <ReportBuilder
      key={definitionId ?? "new"}
      catalog={catalog.data}
      definition={definitionId ? definition.data : undefined}
    />
  );
}
