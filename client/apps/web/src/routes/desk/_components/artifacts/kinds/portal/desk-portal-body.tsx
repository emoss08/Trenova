import { PortalRunResult } from "@/components/assistant/portal-run-result";
import {
  isLivePortalRun,
  portalRunResultView,
  portalRunStatusAttrs,
  portalRunTitle,
} from "@/lib/portal-run";
import { agentExtension } from "@/lib/queries/agent-extension";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { portalTaskFrom } from "../../artifact-payloads";
import { PORTAL_TASK_POLL_MS, portalTaskResultView } from "./portal-task";
import { PortalTaskSkeleton } from "./portal-task-skeleton";

/**
 * A portal task the agent ran, followed as it goes: read again every few seconds while its
 * browser is open, so the person sees it wait on a sign-in code, enters the code or signs in
 * themselves in the live browser, and reads the answer the moment it lands. A run this person
 * cannot read shows only what the agent was told.
 */
export function DeskPortalBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const task = portalTaskFrom(artifact);
  const run = useQuery({
    ...agentExtension.portalRun(task.runId),
    enabled: task.runId !== "",
    retry: false,
    refetchInterval: (query) =>
      isLivePortalRun(query.state.data?.status ?? null) ? PORTAL_TASK_POLL_MS : false,
  });

  if (task.runId !== "" && run.isPending) {
    return <PortalTaskSkeleton />;
  }

  const current = run.data;
  const status = current ? portalRunStatusAttrs(t)[current.status] : null;
  const asked = task.mode === "Read" ? task.question : (current?.instruction ?? "");

  return (
    <div className="dk-ax-pad flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        {status && (
          <Badge variant={phaseTone(status.phase)} title={status.description}>
            {status.text}
          </Badge>
        )}
        <span className="min-w-0 truncate text-sm font-medium">
          {portalRunTitle(t, { mode: task.mode, siteName: current?.siteName ?? task.site })}
        </span>
        {current && (
          <span className="text-muted-foreground ml-auto text-xs">
            {formatUnixTime(current.startedAt)}
          </span>
        )}
      </div>
      {asked !== "" && (
        <p
          className={
            task.mode === "Read" ? "text-sm" : "text-muted-foreground line-clamp-4 text-xs"
          }
          title={task.mode === "Read" ? undefined : asked}
        >
          {asked}
        </p>
      )}
      {run.isError && (
        <Alert size="sm">
          <AlertDescription>
            {t("This task could not be read as it stands now. What the agent was told is shown.")}
          </AlertDescription>
        </Alert>
      )}
      <PortalRunResult
        run={current ? portalRunResultView(current) : portalTaskResultView(task)}
        canEnterCode={current !== undefined}
      />
    </div>
  );
}
