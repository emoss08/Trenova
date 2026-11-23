import { Link } from "@trenova/shared/components/link";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { PlanLockedNotice } from "@trenova/shared/components/plan-locked";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useCannedReports, useForkCannedReport } from "@/hooks/use-reports";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { usePermissions } from "@/hooks/use-permission";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import type { CannedReport } from "@/lib/graphql/reports";
import { Resource } from "@trenova/shared/types/permission";
import { parseReportIR, type ReportIR } from "@/types/report";
import { PlayIcon, RulerIcon, TableIcon } from "@trenova/shared/components/icons";
import { useId, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import {
  compareReportsBySort,
  groupByReportCategory,
  normalizeReportCategory,
  type ReportSortOrder,
} from "../reports-page-state";
import { CategoryGroupHeader, ReportCard, ReportGridEmpty } from "./report-card-chrome";
import { RunReportDialog, type RunReportTarget } from "./run-report-dialog";
import { useReportParameterOptions } from "@/hooks/use-report-parameter-options";

/**
 * Whether the person chose to see the gallery reports their plan's add-ons leave out. Off by
 * default: a report nobody can run is clutter, so it stays out of the way until asked for.
 */
export const SHOW_ADD_ON_REPORTS_STORAGE_KEY = "reports-gallery-show-add-on-reports";

type RunDialogState = {
  target: RunReportTarget;
  name: string;
  defaultFormat: string;
  report: ReportIR | null;
};

function CannedReportCard({
  report,
  canRun,
  canCustomize,
  onRun,
  onCustomize,
  customizing,
}: {
  report: CannedReport;
  canRun: boolean;
  canCustomize: boolean;
  onRun: () => void;
  onCustomize: () => void;
  customizing: boolean;
}) {
  const t = useT();

  return (
    <ReportCard>
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <h3 className="truncate text-sm font-medium">{report.name}</h3>
            <span className="bg-muted text-2xs text-muted-foreground rounded-sm px-1.5 py-0.5">
              {t("v{0}", report.version)}
            </span>
          </div>
          <p className="text-muted-foreground mt-0.5 line-clamp-2 min-h-8 text-xs">
            {t(report.description)}
          </p>
        </div>
      </div>

      <div className="border-border/60 mt-3 flex items-center justify-end border-t pt-3">
        <div className="flex items-center gap-1.5">
          <Button
            variant="ghost"
            size="sm"
            className="text-2xs h-6 gap-1 px-2"
            render={<Link to="/reports/explore" search={{ canned: report.key }} />}
          >
            <TableIcon className="size-3" />
            {t("Explore")}
          </Button>
          {canCustomize && (
            <Button
              variant="ghost"
              size="sm"
              className="text-2xs h-6 gap-1 px-2"
              onClick={onCustomize}
              disabled={customizing}
            >
              <RulerIcon className="size-3" />
              {customizing ? t("Copying...") : t("Customize")}
            </Button>
          )}
          {canRun && (
            <Button size="sm" variant="outline" className="text-2xs h-6 gap-1 px-2" onClick={onRun}>
              <PlayIcon className="size-3" />
              {t("Run")}
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
  onClearFilters,
}: {
  search: string;
  sortBy: ReportSortOrder;
  category: string;
  onClearFilters: () => void;
}) {
  const t = useT();

  const navigate = useNavigate();
  const { data: cannedReports, isLoading } = useCannedReports();
  const forkCanned = useForkCannedReport();
  const { canCreate, canExport } = usePermissions(Resource.Report);
  const [runDialog, setRunDialog] = useState<RunDialogState | null>(null);
  const runParameterOptions = useReportParameterOptions(runDialog?.report, runDialog !== null);
  const [customizingKey, setCustomizingKey] = useState<string | null>(null);
  const [showAddOnReports, setShowAddOnReports] = useLocalStorage(
    SHOW_ADD_ON_REPORTS_STORAGE_KEY,
    false,
  );
  const switchId = useId();

  const matching = useMemo(() => {
    const term = search.trim().toLowerCase();
    return (cannedReports ?? [])
      .filter((report) => {
        if (
          category !== "all" &&
          normalizeReportCategory(report.category) !== normalizeReportCategory(category)
        ) {
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
  // The control is offered only when the plan leaves some gallery report out; a plan that
  // includes them all has nothing to hide.
  const offPlanTotal = (cannedReports ?? []).filter((report) => !report.available).length;
  const hiddenCount = showAddOnReports ? 0 : matching.filter((report) => !report.available).length;
  const filtered = useMemo(
    () => (showAddOnReports ? matching : matching.filter((report) => report.available)),
    [matching, showAddOnReports],
  );
  const groups = useMemo(() => groupByReportCategory(filtered), [filtered]);
  const filtering = search.trim() !== "" || category !== "all";

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
          void navigate({ to: `/reports/builder/${definition.id}` });
        },
        onError: (error) =>
          toast.error(graphQLErrorMessage(error, t("Failed to customize the report"))),
        onSettled: () => setCustomizingKey(null),
      },
    );
  };

  return (
    <>
      {offPlanTotal > 0 ? (
        <div className="flex items-center justify-end gap-2 px-4 pt-3">
          {hiddenCount > 0 ? (
            <span className="text-xs text-foreground-subtle">
              {t("{0, plural, one {# hidden} other {# hidden}}", hiddenCount)}
            </span>
          ) : null}
          <Switch
            id={switchId}
            size="sm"
            checked={showAddOnReports}
            onCheckedChange={(checked) => setShowAddOnReports(checked)}
            aria-label={t("Show reports from add-ons")}
          />
          <label htmlFor={switchId} className="text-sm text-muted-foreground">
            {t("Show reports from add-ons")}
          </label>
        </div>
      ) : null}
      {filtered.length === 0 && hiddenCount > 0 ? (
        <div className="grid p-4">
          <ReportGridEmpty
            variant="reports"
            title={t("Only add-on reports match")}
            description={t(
              "Every gallery report that fits is part of an add-on your plan does not include. Show reports from add-ons to see what they cover.",
            )}
            action={
              <Button size="sm" onClick={() => setShowAddOnReports(true)}>
                {t("Show reports from add-ons")}
              </Button>
            }
          />
        </div>
      ) : filtered.length === 0 ? (
        <div className="grid p-4">
          <ReportGridEmpty
            variant="reports"
            title={filtering ? "Nothing matches" : "Nothing in the gallery"}
            description={
              filtering
                ? "No gallery report fits the search and filters. Widen them, or clear them to see the whole gallery."
                : "The gallery is where ready-made reports live. Once some are published, they appear here to run as they are or copy into your own."
            }
            onClearFilters={filtering ? onClearFilters : undefined}
          />
        </div>
      ) : (
        <div className="space-y-6 p-4">
          {groups.map((group) => (
            <section key={group.key} className="space-y-3">
              <CategoryGroupHeader label={t(group.label)} count={group.items.length} />
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
                {group.items.map((report) =>
                  report.available ? (
                    <CannedReportCard
                      key={report.key}
                      report={report}
                      canRun={canExport}
                      canCustomize={canCreate}
                      customizing={customizingKey === report.key}
                      onRun={() =>
                        setRunDialog({
                          target: { cannedKey: report.key },
                          name: report.name,
                          defaultFormat: report.defaultFormat,
                          report: parseReportIR(report.definition),
                        })
                      }
                      onCustomize={() => handleCustomize(report)}
                    />
                  ) : (
                    <PlanLockedNotice
                      key={report.key}
                      variant="card"
                      title={report.name}
                      featureName={report.planDenial?.featureName ?? ""}
                      addOnName={report.planDenial?.addOnName ?? ""}
                    />
                  ),
                )}
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
        parameters={runDialog?.report?.parameters ?? []}
        parameterOptions={runParameterOptions}
      />
    </>
  );
}
