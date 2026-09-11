import { useT } from "@trenova/shared/i18n/use-t";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import {
  useCreateReportDefinition,
  useDeleteReportDefinition,
  useReportDefinitionList,
  useResetCannedFork,
} from "@/hooks/use-reports";
import { usePermissions } from "@/hooks/use-permission";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import type { ReportDefinition } from "@/lib/graphql/reports";
import { cn } from "@trenova/shared/lib/utils";
import { Resource } from "@trenova/shared/types/permission";
import {
  parseReportIR,
  REPORT_DEFINITION_STATUS_LABELS,
  type ReportParameterDef,
} from "@/types/report";
import { formatDistanceToNowStrict } from "date-fns";
import {
  CalendarClockIcon,
  CopyIcon,
  GlobeIcon,
  LockIcon,
  MoreHorizontalIcon,
  PencilIcon,
  PlayIcon,
  PlusIcon,
  RotateCcwIcon,
  TableIcon,
  Trash2Icon,
} from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { irToInput } from "../builder/_components/builder-state";
import {
  compareReportsBySort,
  groupByReportCategory,
  type ReportSortOrder,
  type ReportStatusFilter,
} from "../reports-page-state";
import { CategoryGroupHeader, ReportCard, ReportGridEmpty } from "./report-card-chrome";
import { ReportSchedulesDialog } from "./report-schedules-dialog";
import { RunReportDialog, type RunReportTarget } from "./run-report-dialog";

type RunDialogState = {
  target: RunReportTarget;
  name: string;
  defaultFormat: string;
  parameters: ReportParameterDef[];
};

const STATUS_DOT: Record<string, string> = {
  draft: "bg-muted-foreground/50",
  active: "bg-emerald-500",
  archived: "bg-muted-foreground/30",
  needs_attention: "bg-amber-500",
};

function StatusDot({ status }: { status: string }) {
  return (
    <Tooltip>
      <TooltipTrigger>
        <span
          className={cn(
            "block size-1.5 rounded-full",
            STATUS_DOT[status] ?? "bg-muted-foreground/40",
          )}
        />
      </TooltipTrigger>
      <TooltipContent>{REPORT_DEFINITION_STATUS_LABELS[status] ?? status}</TooltipContent>
    </Tooltip>
  );
}

function DefinitionCard({
  definition,
  index,
  onRun,
  onSchedules,
  onDuplicate,
  onDelete,
}: {
  definition: ReportDefinition;
  index: number;
  onRun: () => void;
  onSchedules: () => void;
  onDuplicate: () => void;
  onDelete: () => void;
}) {
  const t = useT();

  const navigate = useNavigate();
  const resetFork = useResetCannedFork();
  const { canCreate, canUpdate, canExport } = usePermissions(Resource.Report);

  return (
    <ReportCard index={index} onClick={() => void navigate(`/reports/explore/${definition.id}`)}>
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <StatusDot status={definition.status} />
            <h3 className="truncate text-sm font-medium">{definition.name}</h3>
          </div>
          <p className="text-muted-foreground mt-0.5 line-clamp-2 min-h-8 text-xs">
            {definition.description || "No description"}
          </p>
        </div>
        <div onClick={(event) => event.stopPropagation()}>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-6 opacity-0 transition-opacity group-hover:opacity-100 data-popup-open:opacity-100"
                  aria-label={t("Report actions")}
                >
                  <MoreHorizontalIcon className="size-3.5" />
                </Button>
              }
            />
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                title={t("Explore Results")}
                startContent={<TableIcon className="size-3.5" />}
                onClick={() => void navigate(`/reports/explore/${definition.id}`)}
              />
              <DropdownMenuItem
                title={t("Edit in Builder")}
                startContent={<PencilIcon className="size-3.5" />}
                onClick={() => void navigate(`/reports/builder/${definition.id}`)}
              />
              {canCreate && (
                <DropdownMenuItem
                  title={t("Duplicate")}
                  startContent={<CopyIcon className="size-3.5" />}
                  onClick={onDuplicate}
                />
              )}
              {canExport && (
                <DropdownMenuItem
                  title={t("Schedules")}
                  startContent={<CalendarClockIcon className="size-3.5" />}
                  onClick={onSchedules}
                />
              )}
              {definition.kind === "canned_fork" && canUpdate && (
                <DropdownMenuItem
                  title={t("Reset to Default")}
                  startContent={<RotateCcwIcon className="size-3.5" />}
                  onClick={() =>
                    resetFork.mutate(definition.id, {
                      onSuccess: () => toast.success(t("Report reset to its canned default")),
                      onError: (error) =>
                        toast.error(graphQLErrorMessage(error, "Failed to reset the report")),
                    })
                  }
                />
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title={t("Delete")}
                color="danger"
                startContent={<Trash2Icon className="size-3.5" />}
                onClick={onDelete}
              />
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className="border-border/60 mt-3 flex items-center justify-between border-t pt-3">
        <div className="text-2xs text-muted-foreground flex items-center gap-2">
          {definition.visibility === "shared" ? (
            <span className="flex items-center gap-1">
              <GlobeIcon className="size-3" /> {t("Shared")}
            </span>
          ) : (
            <span className="flex items-center gap-1">
              <LockIcon className="size-3" /> {t("Private")}
            </span>
          )}
          <span className="text-border">•</span>
          <span className="tabular-nums">
            {definition.lastRunAt
              ? `Ran ${formatDistanceToNowStrict(new Date(definition.lastRunAt * 1000), { addSuffix: true })}`
              : "Never run"}
          </span>
        </div>
        {canExport && (
          <Button
            size="sm"
            variant="outline"
            className="text-2xs h-6 gap-1 px-2 opacity-0 transition-opacity group-hover:opacity-100"
            disabled={definition.status !== "active"}
            onClick={(event) => {
              event.stopPropagation();
              onRun();
            }}
          >
            <PlayIcon className="size-3" />
            {t("Run")}
          </Button>
        )}
      </div>
    </ReportCard>
  );
}

export function ReportDefinitionGrid({
  search,
  sortBy,
  category,
  status,
  onClearFilters,
}: {
  search: string;
  sortBy: ReportSortOrder;
  category: string;
  status: ReportStatusFilter;
  onClearFilters: () => void;
}) {
  const t = useT();

  const navigate = useNavigate();
  const {
    data: definitions,
    isLoading,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useReportDefinitionList(search);
  const deleteDefinition = useDeleteReportDefinition();
  const createDefinition = useCreateReportDefinition();
  const { canCreate } = usePermissions(Resource.Report);

  const [runDialog, setRunDialog] = useState<RunDialogState | null>(null);
  const [scheduleTarget, setScheduleTarget] = useState<ReportDefinition | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ReportDefinition | null>(null);

  const duplicateDefinition = (definition: ReportDefinition) => {
    const ir = parseReportIR(definition.definition);
    if (!ir) {
      toast.error(t("This report's definition could not be read"));
      return;
    }
    createDefinition.mutate(
      {
        name: `${definition.name} (copy)`,
        description: definition.description || undefined,
        category: definition.category,
        tags: definition.tags,
        visibility: "private",
        status: definition.status === "active" ? "active" : "draft",
        defaultFormat: definition.defaultFormat,
        definition: irToInput(ir),
      },
      {
        onSuccess: (created) => {
          toast.success(`Duplicated as "${created.name}"`, {
            action: {
              label: "Open",
              onClick: () => void navigate(`/reports/builder/${created.id}`),
            },
          });
        },
        onError: (error) =>
          toast.error(graphQLErrorMessage(error, "Failed to duplicate the report")),
      },
    );
  };

  if (isLoading) {
    return (
      <div className="grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        {Array.from({ length: 8 }, (_, index) => (
          <Skeleton key={index} className="h-40 rounded-lg" />
        ))}
      </div>
    );
  }

  const filtered = (definitions ?? [])
    .filter(
      (definition) =>
        (category === "all" || definition.category === category) &&
        (status === "all" || definition.status === status),
    )
    .sort(compareReportsBySort(sortBy));
  const groups = groupByReportCategory(filtered);
  const hasFilters = Boolean(search) || category !== "all" || status !== "all";

  return (
    <>
      {filtered.length === 0 ? (
        <div className="grid p-4">
          <ReportGridEmpty
            variant="reports"
            title={hasFilters ? "Nothing matches" : "No reports yet"}
            description={
              hasNextPage
                ? "No report on the pages loaded so far fits these filters. Load the rest of the library, or widen them."
                : hasFilters
                  ? "No report fits the search and filters. Widen them, or clear them to see every report you own."
                  : "A report you build or customize from the gallery lands here, grouped by what it is about, ready to run or schedule."
            }
            onClearFilters={hasFilters ? onClearFilters : undefined}
            action={
              canCreate ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void navigate("/reports/builder")}
                >
                  <PlusIcon className="size-3.5" />
                  {t("New report")}
                </Button>
              ) : undefined
            }
          />
        </div>
      ) : (
        <div className="space-y-6 p-4">
          {groups.map((group) => (
            <section key={group.key} className="space-y-3">
              <CategoryGroupHeader label={t(group.label)} count={group.items.length} noun="report" />
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
                {group.items.map((definition, indexInGroup) => (
                  <DefinitionCard
                    key={definition.id}
                    definition={definition}
                    index={group.startIndex + indexInGroup}
                    onRun={() =>
                      setRunDialog({
                        target: { definitionId: definition.id },
                        name: definition.name,
                        defaultFormat: definition.defaultFormat,
                        parameters: parseReportIR(definition.definition)?.parameters ?? [],
                      })
                    }
                    onSchedules={() => setScheduleTarget(definition)}
                    onDuplicate={() => duplicateDefinition(definition)}
                    onDelete={() => setDeleteTarget(definition)}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
      {hasNextPage && (
        <div className="flex justify-center px-4 pb-4">
          <LoadMoreReports pending={isFetchingNextPage} onLoadMore={() => void fetchNextPage()} />
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
      <ReportSchedulesDialog
        open={scheduleTarget !== null}
        onOpenChange={(open) => {
          if (!open) setScheduleTarget(null);
        }}
        definition={scheduleTarget}
      />
      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Delete {0}?", deleteTarget?.name)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("The report definition and its revision history will be permanently removed. Completed run artifacts are kept until they expire.")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (!deleteTarget) return;
                deleteDefinition.mutate(deleteTarget.id, {
                  onSuccess: () => toast.success(t("Report deleted")),
                  onError: (error) =>
                    toast.error(graphQLErrorMessage(error, "Failed to delete the report")),
                });
                setDeleteTarget(null);
              }}
            >
              {t("Delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

/**
 * The rest of the library. The grid narrows by category and status over what it
 * holds, so without a way to reach the next page a filter could report an empty
 * library that is merely an unloaded one.
 */
function LoadMoreReports({ pending, onLoadMore }: { pending: boolean; onLoadMore: () => void }) {
  const t = useT();

  return (
    <Button variant="outline" size="sm" disabled={pending} onClick={onLoadMore}>
      {pending ? (
        <>
          <Spinner className="size-3.5" />
          {t("Loading")}
        </>
      ) : (
        "Load more reports"
      )}
    </Button>
  );
}
