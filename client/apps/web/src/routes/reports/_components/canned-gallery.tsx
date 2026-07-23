import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useCannedReports, useForkCannedReport } from "@/hooks/use-reports";
import { usePermissions } from "@/hooks/use-permission";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import type { CannedReport } from "@/lib/graphql/reports";
import { Resource } from "@trenova/shared/types/permission";
import { parseReportIR, type ReportParameterDef } from "@/types/report";
import { PackageIcon, PencilRulerIcon, PlayIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import {
  compareReportsBySort,
  groupByReportCategory,
  type ReportSortOrder,
} from "../reports-page-state";
import { CategoryGroupHeader, ReportCard, ReportGridEmptyState } from "./report-card-chrome";
import { RunReportDialog, type RunReportTarget } from "./run-report-dialog";

type RunDialogState = {
  target: RunReportTarget;
  name: string;
  defaultFormat: string;
  parameters: ReportParameterDef[];
};

function CannedReportCard({
  report,
  index,
  canRun,
  canCustomize,
  onRun,
  onCustomize,
  customizing,
}: {
  report: CannedReport;
  index: number;
  canRun: boolean;
  canCustomize: boolean;
  onRun: () => void;
  onCustomize: () => void;
  customizing: boolean;
}) {
  return (
    <ReportCard index={index}>
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <h3 className="truncate text-sm font-medium">{report.name}</h3>
            <span className="rounded-sm bg-muted px-1.5 py-0.5 text-2xs text-muted-foreground">
              v{report.version}
            </span>
          </div>
          <p className="mt-0.5 line-clamp-2 min-h-8 text-xs text-muted-foreground">
            {report.description}
          </p>
        </div>
      </div>

      <div className="mt-3 flex items-center justify-end border-t border-border/60 pt-3">
        <div className="flex items-center gap-1.5">
          {canCustomize && (
            <Button
              variant="ghost"
              size="sm"
              className="h-6 gap-1 px-2 text-2xs"
              onClick={onCustomize}
              disabled={customizing}
            >
              <PencilRulerIcon className="size-3" />
              {customizing ? "Copying..." : "Customize"}
            </Button>
          )}
          {canRun && (
            <Button size="sm" variant="outline" className="h-6 gap-1 px-2 text-2xs" onClick={onRun}>
              <PlayIcon className="size-3" />
              Run
            </Button>
          )}
        </div>
      </div>
    </ReportCard>
  );
}

export function CannedGallery({
  search,
  sortBy,
  category,
}: {
  search: string;
  sortBy: ReportSortOrder;
  category: string;
}) {
  const navigate = useNavigate();
  const { data: cannedReports, isLoading } = useCannedReports();
  const forkCanned = useForkCannedReport();
  const { canCreate, canExport } = usePermissions(Resource.Report);
  const [runDialog, setRunDialog] = useState<RunDialogState | null>(null);
  const [customizingKey, setCustomizingKey] = useState<string | null>(null);

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    return (cannedReports ?? [])
      .filter((report) => {
        if (category !== "all" && report.category !== category) {
          return false;
        }
        if (!term) {
          return true;
        }
        return (
          report.name.toLowerCase().includes(term) ||
          report.description.toLowerCase().includes(term) ||
          report.tags.some((tag) => tag.toLowerCase().includes(term))
        );
      })
      .sort(compareReportsBySort(sortBy));
  }, [cannedReports, search, sortBy, category]);
  const groups = useMemo(() => groupByReportCategory(filtered), [filtered]);

  if (isLoading) {
    return (
      <div className="grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => (
          <Skeleton key={index} className="h-40 rounded-lg" />
        ))}
      </div>
    );
  }

  const handleCustomize = (report: CannedReport) => {
    setCustomizingKey(report.key);
    forkCanned.mutate(
      { cannedKey: report.key },
      {
        onSuccess: (definition) => {
          toast.success(`"${report.name}" copied to your reports`);
          void navigate(`/reports/builder/${definition.id}`);
        },
        onError: (error) =>
          toast.error(graphQLErrorMessage(error, "Failed to customize the report")),
        onSettled: () => setCustomizingKey(null),
      },
    );
  };

  return (
    <>
      {filtered.length === 0 ? (
        <div className="grid p-4">
          <ReportGridEmptyState
            icon={PackageIcon}
            title="No gallery reports match"
            description="Try a different search term, or clear the filters."
          />
        </div>
      ) : (
        <div className="space-y-6 p-4">
          {groups.map((group) => (
            <section key={group.key} className="space-y-3">
              <CategoryGroupHeader label={group.label} count={group.items.length} noun="report" />
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
                {group.items.map((report, indexInGroup) => (
                  <CannedReportCard
                    key={report.key}
                    report={report}
                    index={group.startIndex + indexInGroup}
                    canRun={canExport}
                    canCustomize={canCreate}
                    customizing={customizingKey === report.key}
                    onRun={() =>
                      setRunDialog({
                        target: { cannedKey: report.key },
                        name: report.name,
                        defaultFormat: report.defaultFormat,
                        parameters: parseReportIR(report.definition)?.parameters ?? [],
                      })
                    }
                    onCustomize={() => handleCustomize(report)}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
      <RunReportDialog
        open={runDialog !== null}
        onOpenChange={(open) => {
          if (!open) setRunDialog(null);
        }}
        target={runDialog?.target ?? null}
        reportName={runDialog?.name ?? ""}
        defaultFormat={runDialog?.defaultFormat ?? "csv"}
        parameters={runDialog?.parameters ?? []}
      />
    </>
  );
}
