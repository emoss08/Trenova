import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useCreateReportDefinition,
  useDeleteReportDefinition,
  useReportDefinitionList,
  useResetCannedFork,
} from "@/hooks/use-reports";
import { usePermissions } from "@/hooks/use-permission";
import { graphQLErrorMessage } from "@/lib/graphql";
import type { ReportDefinition } from "@/lib/graphql/reports";
import { cn } from "@/lib/utils";
import { Resource } from "@/types/permission";
import {
  parseReportIR,
  REPORT_DEFINITION_STATUS_LABELS,
  type ReportParameterDef,
} from "@/types/report";
import { formatDistanceToNowStrict } from "date-fns";
import {
  CalendarClockIcon,
  CopyIcon,
  FileChartColumnIcon,
  GlobeIcon,
  LockIcon,
  MoreHorizontalIcon,
  PencilIcon,
  PlayIcon,
  PlusIcon,
  RotateCcwIcon,
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
import { CategoryGroupHeader, ReportCard, ReportGridEmptyState } from "./report-card-chrome";
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
  const navigate = useNavigate();
  const resetFork = useResetCannedFork();
  const { canCreate, canUpdate, canExport } = usePermissions(Resource.Report);

  return (
    <ReportCard index={index} onClick={() => void navigate(`/reports/builder/${definition.id}`)}>
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <StatusDot status={definition.status} />
            <h3 className="truncate text-sm font-medium">{definition.name}</h3>
          </div>
          <p className="mt-0.5 line-clamp-2 min-h-8 text-xs text-muted-foreground">
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
                  aria-label="Report actions"
                >
                  <MoreHorizontalIcon className="size-3.5" />
                </Button>
              }
            />
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                title="Edit in Builder"
                startContent={<PencilIcon className="size-3.5" />}
                onClick={() => void navigate(`/reports/builder/${definition.id}`)}
              />
              {canCreate && (
                <DropdownMenuItem
                  title="Duplicate"
                  startContent={<CopyIcon className="size-3.5" />}
                  onClick={onDuplicate}
                />
              )}
              {canExport && (
                <DropdownMenuItem
                  title="Schedules"
                  startContent={<CalendarClockIcon className="size-3.5" />}
                  onClick={onSchedules}
                />
              )}
              {definition.kind === "canned_fork" && canUpdate && (
                <DropdownMenuItem
                  title="Reset to Default"
                  startContent={<RotateCcwIcon className="size-3.5" />}
                  onClick={() =>
                    resetFork.mutate(definition.id, {
                      onSuccess: () => toast.success("Report reset to its canned default"),
                      onError: (error) =>
                        toast.error(graphQLErrorMessage(error, "Failed to reset the report")),
                    })
                  }
                />
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title="Delete"
                color="danger"
                startContent={<Trash2Icon className="size-3.5" />}
                onClick={onDelete}
              />
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className="mt-3 flex items-center justify-between border-t border-border/60 pt-3">
        <div className="flex items-center gap-2 text-2xs text-muted-foreground">
          {definition.visibility === "shared" ? (
            <span className="flex items-center gap-1">
              <GlobeIcon className="size-3" /> Shared
            </span>
          ) : (
            <span className="flex items-center gap-1">
              <LockIcon className="size-3" /> Private
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
            className="h-6 gap-1 px-2 text-2xs opacity-0 transition-opacity group-hover:opacity-100"
            disabled={definition.status !== "active"}
            onClick={(event) => {
              event.stopPropagation();
              onRun();
            }}
          >
            <PlayIcon className="size-3" />
            Run
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
}: {
  search: string;
  sortBy: ReportSortOrder;
  category: string;
  status: ReportStatusFilter;
}) {
  const navigate = useNavigate();
  const { data: definitions, isLoading } = useReportDefinitionList(search);
  const deleteDefinition = useDeleteReportDefinition();
  const createDefinition = useCreateReportDefinition();
  const { canCreate } = usePermissions(Resource.Report);

  const [runDialog, setRunDialog] = useState<RunDialogState | null>(null);
  const [scheduleTarget, setScheduleTarget] = useState<ReportDefinition | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ReportDefinition | null>(null);

  const duplicateDefinition = (definition: ReportDefinition) => {
    const ir = parseReportIR(definition.definition);
    if (!ir) {
      toast.error("This report's definition could not be read");
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
          <ReportGridEmptyState
            icon={FileChartColumnIcon}
            title={hasFilters ? "No reports match your search and filters" : "No reports yet"}
            description={
              hasFilters
                ? "Try a different name, or clear the search and filters."
                : "Build a custom report or customize one from the gallery."
            }
            action={
              !hasFilters && canCreate ? (
                <Button size="sm" onClick={() => void navigate("/reports/builder")}>
                  <PlusIcon className="size-4" />
                  New Report
                </Button>
              ) : undefined
            }
          />
        </div>
      ) : (
        <div className="space-y-6 p-4">
          {groups.map((group) => (
            <section key={group.key} className="space-y-3">
              <CategoryGroupHeader label={group.label} count={group.items.length} noun="report" />
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
            <AlertDialogTitle>Delete {deleteTarget?.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              The report definition and its revision history will be permanently removed. Completed
              run artifacts are kept until they expire.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (!deleteTarget) return;
                deleteDefinition.mutate(deleteTarget.id, {
                  onSuccess: () => toast.success("Report deleted"),
                  onError: (error) =>
                    toast.error(graphQLErrorMessage(error, "Failed to delete the report")),
                });
                setDeleteTarget(null);
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
