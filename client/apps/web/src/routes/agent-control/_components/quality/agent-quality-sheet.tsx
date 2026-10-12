import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  runAgentSuite,
  type AgentQualityDetail,
  type AgentQualityRow,
  type AgentSuiteRun,
} from "@/lib/graphql/agent-quality";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { toast } from "sonner";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { Callout } from "../edit/callout";
import { Ic } from "../kit/ic";
import { Tile } from "../kit/marks";
import { ReadSheet } from "../kit/read-sheet";
import { KV, Line, Pts } from "../kit/values";
import { SuiteRunStatusBadge } from "./quality-columns";
import {
  QUALITY_STALE_MS,
  SUITE_RUN_STATUS,
  formatShare,
  pointsChange,
  ratingReasonLabels,
  sparklineValues,
} from "./quality-model";

type AgentQualitySheetProps = {
  agentId: string | null;
  /** Shown while the agent's figures load. */
  agentName?: string;
  onClose: () => void;
};

/**
 * One agent's quality, read-only: its score over the recent runs with the change against
 * its median, what people thought, how the sweep treats it, and the answers rated lowest.
 * From here a suite run starts at once, the agent opens, or its runs are listed.
 */
export function AgentQualitySheet({ agentId, agentName, onClose }: AgentQualitySheetProps) {
  const t = useT();
  const detail = useQuery({
    ...queries.agentQuality.agent(agentId ?? ""),
    enabled: agentId !== null,
    staleTime: QUALITY_STALE_MS,
  });
  const name = detail.data?.agentName ?? agentName ?? t("Agent quality");

  return (
    <ReadSheet
      open={agentId !== null}
      onClose={onClose}
      label={name}
      head={
        <>
          <Tile agent={{ id: agentId, name }} s={36} />
          <div className="sh-t">
            <b>{name}</b>
            {detail.data && <span>{headline(t, detail.data)}</span>}
          </div>
        </>
      }
    >
      {detail.isError ? (
        <div className="sh-p">
          <Callout tone="d">{t("This agent's quality could not be loaded.")}</Callout>
        </div>
      ) : detail.data ? (
        <AgentQualityBody detail={detail.data} onClose={onClose} />
      ) : (
        <div className="sh-p">
          <p className="ad-h">{t("Loading…")}</p>
        </div>
      )}
    </ReadSheet>
  );
}

function headline(t: ReturnType<typeof useT>, detail: AgentQualityDetail): string {
  const cases = t("{0, plural, one {# golden case} other {# golden cases}}", detail.activeCases);
  const last = detail.lastSuiteRun;

  return last
    ? t("{0} · last run {1}", cases, t(SUITE_RUN_STATUS[last.status].text).toLowerCase())
    : t("{0} · never run", cases);
}

function AgentQualityBody({
  detail,
  onClose,
}: {
  detail: AgentQualityDetail;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useAIControlNavigation();
  const control = useQuery({ ...queries.agentQuality.control(), staleTime: QUALITY_STALE_MS });
  const { allowed: canRun } = usePermission(Resource.AgentEvalSuite, Operation.Create);
  const last = detail.lastSuiteRun;
  const values = sparklineValues(detail.qualityPoints);
  const change = pointsChange(last);

  const run = useApiMutation<AgentSuiteRun, string>({
    mutationFn: (agentDefinitionId) => runAgentSuite(agentDefinitionId),
    onSuccess: async () => {
      toast.success(t("Running {0}'s suite", detail.agentName), {
        description: t("The agent answers its cases with every write simulated."),
      });
      await queryClient.invalidateQueries({ queryKey: queries.agentQuality._def });
    },
    resourceName: t("Suite run"),
  });

  const settings = control.data;
  const satisfaction = !detail.ratingsVisible
    ? t("Needs access to ratings")
    : detail.satisfaction == null
      ? t("No ratings yet")
      : t("{0} of {1}", formatShare(detail.satisfaction), detail.ratings.toLocaleString());

  return (
    <>
      <div className="sh-p">
        <h4 className="sh-k">
          {t(
            "{0, plural, one {Quality score, last # run} other {Quality score, last # runs}}",
            values.length,
          )}
        </h4>
        {values.length === 0 ? (
          <p className="ad-h">{t("No suite run has scored this agent yet.")}</p>
        ) : (
          <div className="bigl">
            <b className="mono">{formatShare(last?.qualityScore ?? null)}</b>
            <Pts value={change} />
            <span className={change !== null && change < 0 ? "t-d" : "t-b"}>
              <Line
                values={values}
                w={240}
                h={48}
                label={t("Quality score over the recent runs")}
              />
            </span>
          </div>
        )}
        {last?.regression && (
          <Callout tone="w">
            {last.comments || t("The agent's score fell after it changed.")}
          </Callout>
        )}
      </div>
      <KV
        items={[
          [t("Satisfaction"), satisfaction],
          settings && [
            t("Regression threshold"),
            t("{0} pts", Math.round(settings.regressionThreshold * 100)),
          ],
          settings && [
            t("Judge"),
            settings.judgeEnabled
              ? t("{0} of answers", formatShare(settings.judgeSampleRate))
              : t("Off"),
          ],
          settings && [t("Cases per night"), t("up to {0}", settings.maxCasesPerAgent)],
          last && [t("Last run"), <SuiteRunStatusBadge key="status" status={last.status} />],
          last && [t("What changed"), last.changeSummary],
        ]}
      />
      {detail.ratingsVisible && (
        <div className="sh-p">
          <h4 className="sh-k">{t("Lowest rated")}</h4>
          {detail.worstRated.length === 0 ? (
            <p className="ad-h">{t("Nothing rated down.")}</p>
          ) : (
            detail.worstRated.map((answer) => {
              const snapshot = answer.sample?.turnSnapshot;
              const why = [
                ...ratingReasonLabels(t, answer.sample?.reasons ?? []),
                answer.sample?.comment ?? "",
              ]
                .filter(Boolean)
                .join(" · ");
              return (
                <div key={answer.id} className="wq">
                  <b>
                    {t("“{0}”", snapshot?.answer || snapshot?.question || t("Answer not kept"))}
                  </b>
                  <span>
                    {why ||
                      t("{0, plural, one {# thumbs down} other {# thumbs down}}", answer.negative)}
                  </span>
                </div>
              );
            })
          )}
        </div>
      )}
      <div className="ad-bar sh-f">
        {canRun && (
          <button
            type="button"
            className="xa"
            disabled={run.isPending || detail.activeCases === 0}
            title={detail.activeCases === 0 ? t("Add a golden case first") : undefined}
            onClick={() => run.mutate(detail.agentDefinitionId)}
          >
            <Ic n="refresh" s={13} />
            {t("Run suite now")}
          </button>
        )}
        <button
          type="button"
          className="xa"
          onClick={() =>
            navigate({
              tab: "agents",
              builder: { mode: "edit", agentId: detail.agentDefinitionId },
            })
          }
        >
          <Ic n="edit" s={13} />
          {t("Open agent")}
        </button>
        <span className="sp" />
        <button
          type="button"
          className="xa"
          onClick={() => {
            onClose();
            navigate({ tab: "quality", view: "runs", qualityAgent: detail.agentDefinitionId });
          }}
        >
          {t("All runs")}
        </button>
      </div>
    </>
  );
}

/** The agents table's sheet: the row's agent. */
export function AgentQualityPanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<AgentQualityRow>) {
  return (
    <AgentQualitySheet
      agentId={open && row ? row.agentDefinitionId : null}
      agentName={row?.name}
      onClose={() => onOpenChange(false)}
    />
  );
}
