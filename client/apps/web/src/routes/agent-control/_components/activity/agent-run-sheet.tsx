import type { AgentRunDetail, AgentRunRow } from "@/lib/graphql/agent-activity-tables";
import { AGENT_EVALUATION_LIST_KEY, replayAgentRun } from "@/lib/graphql/agent-evaluations";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { downloadAgentRunTranscript } from "@/services/agent-run";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDurationFromSeconds, formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { toast } from "sonner";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { Callout } from "../edit/fields";
import { Ic } from "../kit/ic";
import { ReadSheet } from "../kit/read-sheet";
import { KV } from "../kit/values";
import { RunStatusBadge, TriggerBadge, agentTypeLabel } from "./agent-badges";
import { runIsWorking } from "./activity-model";
import { RunTranscriptView } from "./run-transcript-view";

type AgentRunSheetProps = {
  /** The run to read; null closes the sheet. */
  runId: string | null;
  /** The run as a table already holds it, shown while its transcript loads. */
  row?: AgentRunRow | null;
  onClose: () => void;
};

/**
 * One run, read: how it ended and what it said, who started it and on what model, and
 * what it did on the way, step by step from its transcript. From here the trace opens,
 * the transcript downloads, a finished run replays against the agent as it is now, and
 * a run that failed leads to the providers.
 */
export function AgentRunSheet({ runId, row, onClose }: AgentRunSheetProps) {
  const t = useT();
  const detail = useQuery({
    ...queries.agentRun.detail(runId ?? ""),
    enabled: runId !== null,
  });
  const run = detail.data?.run ?? row ?? null;
  const title = run ? run.summary || agentTypeLabel(run.agentType, t) : t("Agent run");

  return (
    <ReadSheet
      open={runId !== null}
      onClose={onClose}
      label={title}
      head={
        <>
          <span className="src-i">
            <Ic n="timeline" s={15} />
          </span>
          <div className="sh-t">
            <b>{title}</b>
            {runId && <span className="mono">{runId}</span>}
          </div>
          {run && <RunStatusBadge value={run.status} t={t} />}
        </>
      }
    >
      {detail.isError ? (
        <div className="sh-p">
          <Callout tone="d">{t("This run could not be loaded. Try again shortly.")}</Callout>
        </div>
      ) : detail.isSuccess && detail.data === null ? (
        <div className="sh-p">
          <p className="ad-h">{t("This run no longer exists.")}</p>
        </div>
      ) : run ? (
        <RunBody run={run} detail={detail.data ?? null} onClose={onClose} />
      ) : (
        <div className="sh-p">
          <p className="ad-h">{t("Loading…")}</p>
        </div>
      )}
    </ReadSheet>
  );
}

function RunBody({
  run,
  detail,
  onClose,
}: {
  run: AgentRunRow;
  detail: AgentRunDetail | null;
  onClose: () => void;
}) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const replay = useReplayRun();
  const working = runIsWorking(run.status);
  const took =
    run.startedAt && run.completedAt
      ? formatDurationFromSeconds(Math.max(0, run.completedAt - run.startedAt))
      : working
        ? t("Still running")
        : null;

  return (
    <>
      {run.errorMessage && (
        <div className="sh-p">
          <code className="sh-code">{run.errorMessage}</code>
        </div>
      )}
      <KV
        items={[
          [t("Agent"), agentTypeLabel(run.agentType, t)],
          [
            t("Started by"),
            <span key="trigger">
              <TriggerBadge value={run.trigger} t={t} />
              {run.handedBy && ` · ${t("Handed by {0}", run.handedBy.name)}`}
            </span>,
          ],
          [
            t("Model"),
            run.modelIdentifier ? (
              <span key="model" className="mono">
                {run.modelIdentifier}
              </span>
            ) : (
              "—"
            ),
          ],
          took ? [t("Took"), took] : null,
          [t("Started"), formatUnixDateTimeMedium(run.startedAt ?? run.createdAt)],
        ]}
      />
      <div className="sh-p">
        <h4 className="sh-k">{t("What it did")}</h4>
        {detail === null ? (
          <p className="ad-h">{t("Loading…")}</p>
        ) : detail.transcript === null ? (
          <p className="ad-h">
            {t(
              "This run kept no transcript. Runs filed before transcripts were kept, and runs that said nothing, have only their summary.",
            )}
          </p>
        ) : (
          <RunTranscriptView runId={run.id} transcript={detail.transcript} />
        )}
      </div>
      <div className="ad-bar sh-f">
        {run.traceUrl && (
          <a className="xa" href={run.traceUrl} target="_blank" rel="noreferrer noopener">
            <Ic n="ext" s={13} />
            {t("Open trace")}
          </a>
        )}
        {detail?.transcript && (
          <button
            type="button"
            className="xa"
            aria-label={t("Download transcript")}
            onClick={() => downloadAgentRunTranscript(run.id)}
          >
            <Ic n="download" s={13} />
            {t("Download")}
          </button>
        )}
        {run.status === "Failed" && (
          <button
            type="button"
            className="xa"
            onClick={() => {
              onClose();
              navigate({ tab: "providers" });
            }}
          >
            <Ic n="plug" s={13} />
            {t("Check providers")}
          </button>
        )}
        <span className="sp" />
        {replay.allowed && run.agentDefinitionId && !working && (
          <button type="button" className="xa" onClick={() => void replay.run(run.id)}>
            <Ic n="refresh" s={13} />
            {t("Replay against the current agent")}
          </button>
        )}
      </div>
    </>
  );
}

/**
 * Replays a finished run against the agent as it is now. A replay spends model calls and
 * counts against the agent's budget, so it is a deliberate action on a finished run; the
 * outcome lands in Evaluations.
 */
export function useReplayRun() {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed } = usePermission(Resource.AgentRun, Operation.Create);

  return {
    allowed,
    run: async (runId: string) => {
      await replayAgentRun(runId);
      toast.success(t("Replay started"), {
        description: t("The comparison appears under Evaluations when it finishes."),
      });
      await queryClient.invalidateQueries({ queryKey: [AGENT_EVALUATION_LIST_KEY] });
    },
  };
}

/** The runs table's sheet: the row's run. */
export function AgentRunPanel({ open, onOpenChange, row }: DataTablePanelProps<AgentRunRow>) {
  return (
    <AgentRunSheet
      runId={open && row ? row.id : null}
      row={row}
      onClose={() => onOpenChange(false)}
    />
  );
}
