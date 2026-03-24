import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CardContent } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { usePermissions } from "@/hooks/use-permission";
import { ApiRequestError } from "@/lib/api";
import { formatToUserTimezone, generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useSamsaraSyncStore } from "@/stores/samsara-sync";
import { Resource } from "@/types/permission";
import type { WorkerSyncLogLevel, WorkerSyncSummary } from "@/types/samsara";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, CopyIcon, ExternalLinkIcon, RefreshCcwIcon } from "lucide-react";
import { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { z } from "zod";
import { LastSuccessfulSyncCard } from "./last-successful-sync-card";
import { RunConsoleLoadingState } from "./run-console-state";

const RunConsole = lazy(() => import("./run-console"));

type NormalizedWorkflowStatus =
  | "running"
  | "completed"
  | "failed"
  | "canceled"
  | "terminated"
  | "timed_out"
  | "continued_as_new"
  | "paused"
  | "unspecified"
  | "unknown";

type SyncProgressSnapshot = {
  syncedActiveWorkers: number;
  unsyncedActiveWorkers: number;
};

const workflowStatusPrefix = "WORKFLOW_EXECUTION_STATUS_";
const workerSyncSummaryStorageKey = "integration:samsara-worker-sync:last-success";
const conflictUsageStatsSchema = z
  .object({
    workflowId: z.string().optional(),
    runId: z.string().optional(),
  })
  .loose();
const workerSyncSummarySchema = z.object({
  workflowId: z.string(),
  runId: z.string(),
  startedAt: z.number().int(),
  closedAt: z.number().int(),
  durationSeconds: z.number().int(),
  result: z.object({
    totalWorkers: z.number().int(),
    activeWorkers: z.number().int(),
    createdDrivers: z.number().int(),
    updatedMappings: z.number().int(),
    failed: z.number().int(),
  }),
});

function normalizeWorkflowStatus(status?: string): NormalizedWorkflowStatus {
  if (!status) {
    return "unknown";
  }

  const normalized = status
    .replace(workflowStatusPrefix, "")
    .trim()
    .toLowerCase() as NormalizedWorkflowStatus;

  switch (normalized) {
    case "running":
    case "completed":
    case "failed":
    case "canceled":
    case "terminated":
    case "timed_out":
    case "continued_as_new":
    case "paused":
    case "unspecified":
      return normalized;
    default:
      return "unknown";
  }
}

function isWorkflowTerminal(status: NormalizedWorkflowStatus): boolean {
  switch (status) {
    case "completed":
    case "failed":
    case "canceled":
    case "terminated":
    case "timed_out":
      return true;
    default:
      return false;
  }
}

function getStatusLabel(status?: string): string {
  const normalized = normalizeWorkflowStatus(status);
  switch (normalized) {
    case "running":
      return "Running";
    case "completed":
      return "Completed";
    case "failed":
      return "Failed";
    case "canceled":
      return "Canceled";
    case "terminated":
      return "Terminated";
    case "timed_out":
      return "Timed Out";
    case "continued_as_new":
      return "Continued As New";
    case "paused":
      return "Paused";
    case "unspecified":
      return "Queued";
    default:
      return "Unknown";
  }
}

function getStatusVariant(
  status: NormalizedWorkflowStatus,
  failedCount: number,
): "active" | "inactive" | "info" | "warning" | "secondary" {
  if (status === "completed") {
    return failedCount > 0 ? "warning" : "active";
  }

  switch (status) {
    case "running":
      return "info";
    case "failed":
    case "terminated":
    case "timed_out":
      return "inactive";
    case "canceled":
      return "warning";
    default:
      return "secondary";
  }
}

function getStatusErrorMessage(error: unknown): string {
  if (!(error instanceof ApiRequestError)) {
    return "Unable to retrieve workflow status right now.";
  }

  return error.data.detail || error.data.title || "Unable to retrieve workflow status right now.";
}

function getDriftTypeLabel(type: string): string {
  switch (type) {
    case "missing_mapping":
      return "Missing Mapping";
    case "missing_remote_driver":
      return "Missing Remote Driver";
    case "mapping_mismatch":
      return "Mapping Mismatch";
    case "remote_deactivated":
      return "Remote Deactivated";
    default:
      return type
        .replaceAll("_", " ")
        .trim()
        .replace(/\b\w/g, (char) => char.toUpperCase());
  }
}

function getDriftTypeVariant(type: string): BadgeVariant {
  switch (type) {
    case "missing_mapping":
      return "warning";
    case "missing_remote_driver":
      return "inactive";
    case "mapping_mismatch":
      return "orange";
    case "remote_deactivated":
      return "secondary";
    default:
      return "secondary";
  }
}

export function SamsaraWorkerSyncCard({
  embedded = false,
  open = true,
}: {
  embedded?: boolean;
  open?: boolean;
}) {
  const queryClient = useQueryClient();
  const { canUpdate } = usePermissions(Resource.Integration);

  const [showAllFailures, setShowAllFailures] = useState(false);
  const [trackedWorkflowId, setTrackedWorkflowId] = useState<string | null>(null);
  const [trackedRunId, setTrackedRunId] = useState<string | null>(null);
  const [, setLogLines] = useSamsaraSyncStore.use("logLines");
  const [, setLastSuccessfulSync] = useSamsaraSyncStore.use("lastSuccessfulSync");
  const [syncProgressBaseline, setSyncProgressBaseline] = useState<SyncProgressSnapshot | null>(
    null,
  );
  const [retryingWorkerID, setRetryingWorkerID] = useState<string | null>(null);

  const notFoundErrorHandledAtRef = useRef(0);
  const terminalToastKeyRef = useRef("");
  const completionHandledKeyRef = useRef("");
  const previousStatusKeyRef = useRef("");

  const appendLog = useCallback(
    (level: WorkerSyncLogLevel, source: string, message: string) => {
      setLogLines((prev) => {
        const next = [
          ...prev,
          {
            id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
            ts: Date.now(),
            level,
            source,
            message,
          },
        ];
        return next.slice(-400);
      });
    },
    [setLogLines],
  );

  useEffect(() => {
    if (!open) {
      setTrackedWorkflowId(null);
      setTrackedRunId(null);
      setLogLines([]);
      setShowAllFailures(false);
      setSyncProgressBaseline(null);
      setRetryingWorkerID(null);
      previousStatusKeyRef.current = "";
      terminalToastKeyRef.current = "";
      completionHandledKeyRef.current = "";
    }
  }, [open, setLogLines]);

  useEffect(() => {
    if (!open || typeof window === "undefined") {
      return;
    }

    try {
      const raw = window.localStorage.getItem(workerSyncSummaryStorageKey);
      if (!raw) {
        return;
      }

      const parsed = workerSyncSummarySchema.safeParse(JSON.parse(raw));
      if (parsed.success) {
        setLastSuccessfulSync(parsed.data);
      }
    } catch {
      // Best effort read only; ignore malformed storage values.
    }
  }, [open, setLastSuccessfulSync]);

  const statusQuery = useQuery({
    ...queries.integration.samsaraWorkerSyncStatus(
      trackedWorkflowId ?? "",
      trackedRunId ?? undefined,
    ),
    enabled: open && Boolean(trackedWorkflowId),
    retry: 1,
    refetchInterval: (query) => {
      const statusResponse = query.state.data;
      if (!statusResponse) {
        return 3000;
      }

      const normalizedStatus = normalizeWorkflowStatus(statusResponse.status);
      return isWorkflowTerminal(normalizedStatus) ? false : 3000;
    },
  });

  const readinessQuery = useQuery({
    ...queries.integration.samsaraWorkerSyncReadiness(),
    enabled: open,
    refetchInterval: trackedWorkflowId ? 3000 : 30000,
  });
  const driftQuery = useQuery({
    ...queries.integration.samsaraWorkerSyncDrift(),
    enabled: open,
  });
  const samsaraConfigQuery = useQuery({
    ...queries.integration.config("Samsara"),
    enabled: open,
  });

  const startSyncMutation = useMutation({
    mutationFn: () => apiService.integrationService.startSamsaraWorkerSync(),
    onSuccess: (response) => {
      setTrackedWorkflowId(response.workflowId);
      setTrackedRunId(response.runId);
      setShowAllFailures(false);
      setSyncProgressBaseline(
        readinessQuery.data
          ? {
              syncedActiveWorkers: readinessQuery.data.syncedActiveWorkers,
              unsyncedActiveWorkers: readinessQuery.data.unsyncedActiveWorkers,
            }
          : null,
      );
      appendLog("success", "sync", `Workflow started: ${response.workflowId} (${response.runId})`);
      toast.success("Samsara worker sync started", {
        description: "Monitoring the workflow run now.",
      });
    },
    onError: (error) => {
      if (error instanceof ApiRequestError && error.isConflictError()) {
        const parsedUsageStats = conflictUsageStatsSchema.safeParse(error.getUsageStats());

        if (parsedUsageStats.success && parsedUsageStats.data.workflowId) {
          setTrackedWorkflowId(parsedUsageStats.data.workflowId);
          setTrackedRunId(parsedUsageStats.data.runId ?? null);
          setSyncProgressBaseline(
            readinessQuery.data
              ? {
                  syncedActiveWorkers: readinessQuery.data.syncedActiveWorkers,
                  unsyncedActiveWorkers: readinessQuery.data.unsyncedActiveWorkers,
                }
              : null,
          );
          appendLog(
            "warn",
            "sync",
            `Workflow already running, attached to ${parsedUsageStats.data.workflowId}`,
          );
          toast.info("Sync already running", {
            description: "Switched to monitoring the existing workflow run.",
          });
          return;
        }
      }

      if (error instanceof ApiRequestError) {
        appendLog("error", "sync", error.data.detail || error.data.title || "Failed to start sync");
        toast.error("Failed to start sync", {
          description: error.data.detail || error.data.title,
        });
        return;
      }

      appendLog("error", "sync", "Unable to submit the sync request right now.");
      toast.error("Failed to start sync", {
        description: "Unable to submit the sync request right now.",
      });
    },
  });

  const detectDriftMutation = useMutation({
    mutationFn: () => apiService.integrationService.detectSamsaraWorkerSyncDrift(),
    onSuccess: async (response) => {
      appendLog(
        "info",
        "drift",
        `Drift detection complete: ${response.totalDrifts} drift(s) found`,
      );
      toast.success("Worker drift detection complete");
      await queryClient.invalidateQueries({
        queryKey: queries.integration.samsaraWorkerSyncDrift().queryKey,
      });
    },
    onError: (error) => {
      if (error instanceof ApiRequestError) {
        appendLog(
          "error",
          "drift",
          error.data.detail || error.data.title || "Failed to detect worker drift",
        );
        toast.error("Failed to detect worker drift", {
          description: error.data.detail || error.data.title,
        });
        return;
      }
      appendLog("error", "drift", "Failed to detect worker drift");
      toast.error("Failed to detect worker drift");
    },
  });

  const repairDriftMutation = useMutation({
    mutationFn: () => apiService.integrationService.repairSamsaraWorkerSyncDrift(),
    onSuccess: async (response) => {
      appendLog(
        "success",
        "drift",
        `Drift repair completed: repaired ${response.repairedWorkers}, failed ${response.failedWorkers}`,
      );
      toast.success("Worker drift repair completed", {
        description: `Repaired ${response.repairedWorkers} worker(s).`,
      });
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ["worker-list"],
          refetchType: "all",
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.samsaraWorkerSyncReadiness().queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.samsaraWorkerSyncDrift().queryKey,
        }),
      ]);
    },
    onError: (error) => {
      if (error instanceof ApiRequestError) {
        appendLog(
          "error",
          "drift",
          error.data.detail || error.data.title || "Failed to repair worker drift",
        );
        toast.error("Failed to repair worker drift", {
          description: error.data.detail || error.data.title,
        });
        return;
      }
      appendLog("error", "drift", "Failed to repair worker drift");
      toast.error("Failed to repair worker drift");
    },
  });

  const retryWorkerMutation = useMutation({
    mutationFn: (workerID: string) =>
      apiService.integrationService.repairSamsaraWorkerSyncDrift([workerID]),
    onMutate: (workerID) => {
      setRetryingWorkerID(workerID);
    },
    onSuccess: async (response, workerID) => {
      if (response.failedWorkers > 0) {
        const firstFailure = response.failures?.[0];
        appendLog(
          "warn",
          "retry",
          firstFailure?.message || "Worker retry request completed with issues",
        );
        toast.error("Worker retry failed", {
          description: firstFailure?.message || "Unable to repair mapping for this worker.",
        });
      } else {
        appendLog("success", "retry", `Queued worker ${workerID} for resync`);
        toast.success("Worker retry queued", {
          description: "The worker mapping was repaired and will be picked up on next sync.",
        });
      }

      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.samsaraWorkerSyncDrift().queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.samsaraWorkerSyncReadiness().queryKey,
        }),
      ]);
    },
    onError: (error) => {
      if (error instanceof ApiRequestError) {
        appendLog("error", "retry", error.data.detail || error.data.title || "Worker retry failed");
        toast.error("Worker retry failed", {
          description: error.data.detail || error.data.title,
        });
        return;
      }

      appendLog("error", "retry", "Worker retry failed");
      toast.error("Worker retry failed");
    },
    onSettled: () => {
      setRetryingWorkerID(null);
    },
  });

  useEffect(() => {
    const error = statusQuery.error;

    if (!(error instanceof ApiRequestError) || !error.isNotFoundError()) {
      return;
    }

    if (statusQuery.errorUpdatedAt === notFoundErrorHandledAtRef.current) {
      return;
    }

    notFoundErrorHandledAtRef.current = statusQuery.errorUpdatedAt;
    setTrackedWorkflowId(null);
    setTrackedRunId(null);
    setShowAllFailures(false);
    appendLog("error", "workflow", "Tracked workflow not found. Cleared active tracking.");

    toast.error("Sync workflow not found", {
      description: "Tracking was cleared. Start a new sync to continue.",
    });
  }, [statusQuery.error, statusQuery.errorUpdatedAt, appendLog]);

  const statusResponse = statusQuery.data;
  const normalizedStatus = normalizeWorkflowStatus(statusResponse?.status);
  const syncResult = statusResponse?.result;
  const readiness = readinessQuery.data;
  const samsaraConfig = samsaraConfigQuery.data;
  const failures = syncResult?.failures ?? [];
  const drift = driftQuery.data;
  const driftRows = drift?.drifts ?? [];
  const visibleFailures = showAllFailures ? failures : failures.slice(0, 10);
  const hasHiddenFailures = failures.length > visibleFailures.length;

  const isTrackingWorkflow = Boolean(trackedWorkflowId);
  const isWorkflowRunning = isTrackingWorkflow && !isWorkflowTerminal(normalizedStatus);
  const areAllActiveWorkersSynced = readiness?.allActiveWorkersSynced ?? false;
  const hasToken = samsaraConfig?.fields?.some((f) => f.key === "token" && f.hasValue) ?? false;
  const isSamsaraConfigured = Boolean(samsaraConfig?.enabled && hasToken);
  const failedRecords = syncResult?.failed ?? 0;

  useEffect(() => {
    if (!isTrackingWorkflow || !statusResponse) {
      return;
    }

    const nextStatusKey = `${statusResponse.runId}:${statusResponse.status}:${statusResponse.closedAt ?? ""}`;
    if (previousStatusKeyRef.current !== nextStatusKey) {
      previousStatusKeyRef.current = nextStatusKey;
      appendLog(
        normalizedStatus === "failed"
          ? "error"
          : normalizedStatus === "completed"
            ? "success"
            : "debug",
        "workflow",
        `Status changed to ${getStatusLabel(statusResponse.status)}`,
      );
    }

    if (normalizedStatus === "running") {
      appendLog("debug", "workflow", "Heartbeat: sync workflow is still running");
    }
  }, [isTrackingWorkflow, normalizedStatus, statusResponse, appendLog]);

  useEffect(() => {
    if (!statusResponse || !trackedWorkflowId) {
      return;
    }

    const terminalStatus = normalizeWorkflowStatus(statusResponse.status);
    if (!isWorkflowTerminal(terminalStatus)) {
      return;
    }

    const key = `${trackedWorkflowId}:${statusResponse.runId}:${terminalStatus}:${statusResponse.closedAt ?? ""}`;
    if (terminalToastKeyRef.current !== key) {
      terminalToastKeyRef.current = key;

      if (terminalStatus === "completed") {
        if ((statusResponse.result?.failed ?? 0) > 0) {
          appendLog(
            "warn",
            "sync",
            `Completed with issues: ${statusResponse.result?.failed ?? 0} worker record(s) failed`,
          );
          toast.error("Samsara sync completed with issues", {
            description: `${statusResponse.result?.failed ?? 0} worker record(s) failed to sync.`,
          });
        } else {
          appendLog("success", "sync", "Completed successfully with no failures");
          toast.success("Samsara sync completed successfully");
        }
      } else {
        appendLog(
          "error",
          "sync",
          `Workflow ended in ${getStatusLabel(statusResponse.status)}: ${statusResponse.error || "no details"}`,
        );
        toast.error("Samsara sync did not complete", {
          description: statusResponse.error || getStatusLabel(statusResponse.status),
        });
      }
    }

    if (terminalStatus === "completed" && completionHandledKeyRef.current !== key) {
      completionHandledKeyRef.current = key;

      void queryClient.invalidateQueries({
        queryKey: ["worker-list"],
        refetchType: "all",
      });
      void queryClient.invalidateQueries({
        queryKey: queries.integration.samsaraWorkerSyncReadiness().queryKey,
      });
    }

    if (
      terminalStatus === "completed" &&
      (statusResponse.result?.failed ?? 0) === 0 &&
      statusResponse.result &&
      statusResponse.closedAt &&
      statusResponse.startedAt
    ) {
      const summary: WorkerSyncSummary = {
        workflowId: trackedWorkflowId,
        runId: statusResponse.runId,
        startedAt: statusResponse.startedAt,
        closedAt: statusResponse.closedAt,
        durationSeconds: Math.max(statusResponse.closedAt - statusResponse.startedAt, 0),
        result: {
          totalWorkers: statusResponse.result.totalWorkers,
          activeWorkers: statusResponse.result.activeWorkers,
          createdDrivers: statusResponse.result.createdDrivers,
          updatedMappings: statusResponse.result.updatedMappings,
          failed: statusResponse.result.failed,
        },
      };
      setLastSuccessfulSync(summary);

      if (typeof window !== "undefined") {
        window.localStorage.setItem(workerSyncSummaryStorageKey, JSON.stringify(summary));
      }
    }

    if (isWorkflowTerminal(terminalStatus)) {
      setSyncProgressBaseline(null);
    }
  }, [queryClient, statusResponse, trackedWorkflowId, setLastSuccessfulSync, appendLog]);

  useEffect(() => {
    if (!isTrackingWorkflow || !readiness || syncProgressBaseline) {
      return;
    }

    setSyncProgressBaseline({
      syncedActiveWorkers: readiness.syncedActiveWorkers,
      unsyncedActiveWorkers: readiness.unsyncedActiveWorkers,
    });
  }, [isTrackingWorkflow, readiness, syncProgressBaseline]);

  const handleStartSync = () => {
    startSyncMutation.mutate();
  };

  const handleClearTrackedRun = () => {
    setTrackedWorkflowId(null);
    setTrackedRunId(null);
    setShowAllFailures(false);
    setSyncProgressBaseline(null);
    appendLog("info", "workflow", "Cleared tracked run from UI");
  };

  const handleCopyFailure = async (message: string) => {
    try {
      await navigator.clipboard.writeText(message);
      toast.success("Failure message copied");
    } catch {
      toast.error("Failed to copy failure message");
    }
  };

  const handleOpenWorker = (workerID: string) => {
    if (typeof window === "undefined") {
      return;
    }

    window.open(
      `/workers?panelEntityId=${workerID}&panelType=edit`,
      "_blank",
      "noopener,noreferrer",
    );
  };

  const activeStatusVariant = getStatusVariant(normalizedStatus, failedRecords);
  const currentStatusLabel = isTrackingWorkflow ? getStatusLabel(statusResponse?.status) : "Idle";

  const content = (
    <div className="space-y-4">
      {embedded && (
        <div className="flex flex-col border-b border-border p-4 leading-tight">
          <div className="flex flex-row items-center gap-2">
            <p className="text-2xl font-semibold">Samsara Worker Sync</p>
            <Badge variant={isTrackingWorkflow ? activeStatusVariant : "secondary"}>
              {currentStatusLabel}
            </Badge>
          </div>
          <span className="text-sm text-muted-foreground">
            Sync Trenova worker records directly into Samsara.
          </span>
        </div>
      )}
      <div className="flex flex-wrap items-center gap-2 px-4">
        {!canUpdate && (
          <Alert variant="warning">
            <AlertTriangleIcon />
            <AlertTitle>Read-only access</AlertTitle>
            <AlertDescription>
              Worker update permission is required to start a new sync.
            </AlertDescription>
          </Alert>
        )}
        <Button
          size="sm"
          onClick={handleStartSync}
          isLoading={startSyncMutation.isPending}
          loadingText="Starting..."
          disabled={
            !canUpdate ||
            !isSamsaraConfigured ||
            isWorkflowRunning ||
            areAllActiveWorkersSynced ||
            detectDriftMutation.isPending ||
            repairDriftMutation.isPending ||
            retryWorkerMutation.isPending
          }
        >
          Start Sync
        </Button>

        {isTrackingWorkflow && (
          <Button
            size="sm"
            variant="outline"
            onClick={handleClearTrackedRun}
            disabled={startSyncMutation.isPending}
          >
            Clear Tracked Run
          </Button>
        )}
        <Button
          size="sm"
          variant="outline"
          onClick={() => detectDriftMutation.mutate()}
          isLoading={detectDriftMutation.isPending}
          loadingText="Detecting..."
          disabled={
            !canUpdate ||
            !isSamsaraConfigured ||
            repairDriftMutation.isPending ||
            retryWorkerMutation.isPending
          }
        >
          Detect Drift
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => repairDriftMutation.mutate()}
          isLoading={repairDriftMutation.isPending}
          loadingText="Repairing..."
          disabled={
            !canUpdate ||
            !isSamsaraConfigured ||
            detectDriftMutation.isPending ||
            driftRows.length === 0 ||
            retryWorkerMutation.isPending
          }
        >
          Repair Drift
        </Button>
      </div>
      <ScrollArea className="flex max-h-[calc(100vh-14rem)] flex-col px-4 [&_[data-slot=scroll-area-viewport]>div]:block!">
        <div className="space-y-3 pr-3">
          <div className="grid gap-2 rounded-md border border-border bg-muted/30 p-3 text-sm sm:grid-cols-2 xl:grid-cols-4">
            <div className="rounded-md border border-border bg-background p-3 text-xs text-muted-foreground">
              <p>Workflow</p>
              <div className="mt-1">
                <Badge variant={isTrackingWorkflow ? activeStatusVariant : "secondary"}>
                  {currentStatusLabel}
                </Badge>
              </div>
            </div>
            <div className="rounded-md border border-border bg-background p-3 text-xs text-muted-foreground">
              <p>Workflow ID</p>
              <p className="mt-1 truncate font-mono text-foreground">
                {trackedWorkflowId ?? "N/A"}
              </p>
            </div>
            <div className="rounded-md border border-border bg-background p-3 text-xs text-muted-foreground">
              <p>Run ID</p>
              <p className="mt-1 truncate font-mono text-foreground">
                {statusResponse?.runId || trackedRunId || "N/A"}
              </p>
            </div>
            <div className="rounded-md border border-border bg-background p-3 text-xs text-muted-foreground">
              <p>Last Updated</p>
              <p className="mt-1 text-foreground">
                {generateDateTimeStringFromUnixTimestamp(
                  statusResponse?.closedAt || statusResponse?.startedAt,
                )}
              </p>
            </div>
          </div>
          {readiness && (
            <div className="grid gap-2 rounded-md border border-border bg-muted/30 p-3 text-xs sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-md border border-border bg-background p-3">
                <p className="text-muted-foreground">Synced Active</p>
                <p className="font-semibold text-foreground">
                  {readiness.syncedActiveWorkers} / {readiness.activeWorkers}
                </p>
              </div>
              <div className="rounded-md border border-border bg-background p-3">
                <p className="text-muted-foreground">Unsynced Active</p>
                <p className="font-semibold text-foreground">{readiness.unsyncedActiveWorkers}</p>
              </div>
              <div className="rounded-md border border-border bg-background p-3">
                <p className="text-muted-foreground">Total Workers</p>
                <p className="font-semibold text-foreground">{readiness.totalWorkers}</p>
              </div>
              <div className="rounded-md border border-border bg-background p-3">
                <p className="text-muted-foreground">Readiness Scan</p>
                <p className="font-semibold text-foreground">
                  {formatToUserTimezone(readiness.lastCalculatedAt)}
                </p>
              </div>
            </div>
          )}
          <LastSuccessfulSyncCard />
          <Suspense fallback={<RunConsoleLoadingState description="Loading run console..." />}>
            <RunConsole isWorkflowRunning={isWorkflowRunning} />
          </Suspense>
          {statusQuery.isLoading && isTrackingWorkflow && (
            <div className="inline-flex items-center gap-2 text-xs text-muted-foreground">
              <Spinner className="size-3.5" />
              Loading workflow status...
            </div>
          )}
          {statusQuery.error &&
            !(
              statusQuery.error instanceof ApiRequestError && statusQuery.error.isNotFoundError()
            ) && (
              <Alert variant="destructive">
                <AlertTriangleIcon />
                <AlertTitle>Status lookup failed</AlertTitle>
                <AlertDescription>{getStatusErrorMessage(statusQuery.error)}</AlertDescription>
              </Alert>
            )}
          {driftRows.length > 0 && (
            <div className="space-y-2">
              <h3 className="text-sm font-medium">Detected Drift ({driftRows.length})</h3>
              <div className="rounded-md border border-border">
                <Table containerClassName="max-h-72 rounded-md">
                  <TableHeader>
                    <TableRow>
                      <TableHead>Worker</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Message</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {driftRows.slice(0, 20).map((item) => (
                      <TableRow key={`${item.workerId}-${item.driftType}-${item.detectedAt}`}>
                        <TableCell className="align-top">{item.workerName}</TableCell>
                        <TableCell className="align-top">
                          <Badge variant={getDriftTypeVariant(item.driftType)}>
                            {getDriftTypeLabel(item.driftType)}
                          </Badge>
                        </TableCell>
                        <TableCell className="align-top wrap-break-word whitespace-normal">
                          {item.message}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}

          {failures.length > 0 && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium">Sync Failures ({failures.length})</h3>
                {hasHiddenFailures && (
                  <Button size="sm" variant="outline" onClick={() => setShowAllFailures(true)}>
                    Show All
                  </Button>
                )}
              </div>

              <div className="rounded-md border border-border">
                <Table containerClassName="max-h-72 rounded-md">
                  <TableHeader>
                    <TableRow>
                      <TableHead>Worker</TableHead>
                      <TableHead>Operation</TableHead>
                      <TableHead>Message</TableHead>
                      <TableHead className="w-48 text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {visibleFailures.map((failure) => (
                      <TableRow key={`${failure.workerId}-${failure.operation}-${failure.message}`}>
                        <TableCell className="align-top">{failure.worker}</TableCell>
                        <TableCell className="align-top">{failure.operation}</TableCell>
                        <TableCell className="align-top wrap-break-word whitespace-normal">
                          {failure.message}
                        </TableCell>
                        <TableCell className="align-top">
                          <div className="flex justify-end gap-1">
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-7"
                              onClick={() => retryWorkerMutation.mutate(failure.workerId)}
                              disabled={
                                !canUpdate ||
                                retryWorkerMutation.isPending ||
                                detectDriftMutation.isPending ||
                                repairDriftMutation.isPending
                              }
                              isLoading={
                                retryWorkerMutation.isPending &&
                                retryingWorkerID === failure.workerId
                              }
                              loadingText="Retrying..."
                            >
                              <RefreshCcwIcon className="size-3.5" />
                              Retry
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-7"
                              onClick={() => handleCopyFailure(failure.message)}
                            >
                              <CopyIcon className="size-3.5" />
                              Copy
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-7"
                              onClick={() => handleOpenWorker(failure.workerId)}
                            >
                              <ExternalLinkIcon className="size-3.5" />
                              Open
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  );

  if (embedded) {
    return content;
  }

  return (
    <div>
      <div className="border-b border-border">
        <div>Samsara Worker Sync</div>
        <div>Start and monitor worker synchronization from TMS to Samsara.</div>
      </div>
      <CardContent className="space-y-4">{content}</CardContent>
    </div>
  );
}
