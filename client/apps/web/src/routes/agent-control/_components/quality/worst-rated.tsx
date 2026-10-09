import { DataTable } from "@/components/data-table/data-table";
import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { AGENT_EVAL_CASE_LIST_KEY, createAgentEvalCase } from "@/lib/graphql/agent-eval-cases";
import {
  AGENT_WORST_RATED_LIST_KEY,
  createAgentWorstRatedTableGraphQLConfig,
  type AgentWorstRatedAnswer,
  type AgentWorstRatedRow,
} from "@/lib/graphql/agent-quality";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryState } from "nuqs";
import { useMemo } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { QUALITY_AGENT_PARAM, qualityAgentParser } from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { AgentScope } from "./agent-scope";
import { Ic } from "../kit/ic";
import { Tile } from "../kit/marks";
import { ReadSheet } from "../kit/read-sheet";
import { answerQuestion, getWorstRatedColumns } from "./quality-columns";
import { TARGET_TYPE_LABEL, ratingReasonLabels } from "./quality-model";
import { Button } from "@trenova/shared/components/ui/button";

/**
 * What the person saw when they rated an answer down, read-only: the question, the answer
 * and the tools it used as they were kept, and what the person said. Restricted values
 * were replaced before they were kept. From here the answer becomes a golden case, a
 * memory is written, or the conversation opens for its owner, who alone can read it.
 */
export function WorstRatedSheet({
  answer,
  onClose,
}: {
  answer: AgentWorstRatedAnswer | null;
  onClose: () => void;
}) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const queryClient = useQueryClient();
  const { allowed: canCapture } = usePermission(Resource.AgentEvalSuite, Operation.Create);
  const capture = useApiMutation<{ duplicate: boolean }, string>({
    mutationFn: (feedbackId) => createAgentEvalCase({ fromFeedback: { feedbackId } }),
    onSuccess: async ({ duplicate }) => {
      toast.success(
        duplicate
          ? t("Already in {0}'s golden set", answer?.agentName ?? "")
          : t("Added to {0}'s golden set", answer?.agentName ?? ""),
      );
      await queryClient.invalidateQueries({ queryKey: [AGENT_EVAL_CASE_LIST_KEY] });
      onClose();
    },
    resourceName: t("Golden case"),
  });
  const snapshot = answer?.sample?.turnSnapshot;
  const said = answer?.sample
    ? [...ratingReasonLabels(t, answer.sample.reasons), answer.sample.comment ?? ""]
        .filter(Boolean)
        .join(" · ")
    : "";

  return (
    <ReadSheet
      open={answer !== null}
      onClose={onClose}
      label={answer ? answerQuestion(answer) || t("Answer") : t("Answer")}
      head={
        answer && (
          <>
            <Tile agent={{ id: answer.agentDefinitionId, name: answer.agentName }} s={36} />
            <div className="sh-t">
              <b>{answer.agentName}</b>
              <span>
                {t(
                  "{0} · {1}",
                  t(TARGET_TYPE_LABEL[answer.targetType]),
                  formatUnixDateMedium(answer.lastRatedAt),
                )}
              </span>
            </div>
          </>
        )
      }
    >
      {answer && (
        <>
          <div className="sh-p">
            {snapshot?.question && (
              <>
                <h4 className="sh-k">{t("The question")}</h4>
                <p className="wq-a q">{snapshot.question}</p>
              </>
            )}
            <h4 className="sh-k mt">{t("The answer")}</h4>
            <p className="wq-a">{snapshot?.answer ? t("“{0}”", snapshot.answer) : "—"}</p>
            <h4 className="sh-k mt">{t("What they said")}</h4>
            <p className="wq-a q">
              {said ||
                t(
                  "{0, plural, one {# thumbs down} other {# thumbs down}}, {1, plural, one {# up} other {# up}}",
                  answer.negative,
                  answer.positive,
                )}
            </p>
            {snapshot && snapshot.tools.length > 0 && (
              <>
                <h4 className="sh-k mt">{t("Tools it used")}</h4>
                <ol className="trs">
                  {snapshot.tools.map((tool, index) => (
                    <li key={`${tool.name}-${index}`} className={tool.failed ? "f" : undefined}>
                      <i />
                      <div>
                        <b className="mono">{tool.name}</b>
                        <span>{tool.failed ? t("Failed · {0}", tool.summary) : tool.summary}</span>
                      </div>
                    </li>
                  ))}
                </ol>
                {snapshot.omittedTools > 0 && (
                  <p className="ad-h">
                    {t(
                      "{0, plural, one {# more tool} other {# more tools}}",
                      snapshot.omittedTools,
                    )}
                  </p>
                )}
              </>
            )}
            {snapshot?.redacted && (
              <p className="ad-h">{t("Restricted values were replaced before this was kept.")}</p>
            )}
          </div>
          <div className="ad-bar sh-f">
            {canCapture && answer.sample && (
              <Button
                type="button"
                variant="default" size="sm"
                disabled={capture.isPending}
                onClick={() => answer.sample && capture.mutate(answer.sample.id)}
              >
                <Ic n="plus" s={12} />
                {t("Add to golden set")}
              </Button>
            )}
            <button
              type="button"
              className="xa"
              onClick={() => {
                onClose();
                navigate({ tab: "memory", panel: { mode: "create" } });
              }}
            >
              <Ic n="brain" s={13} />
              {t("Write a memory")}
            </button>
            <span className="sp" />
            {answer.canOpenThread && answer.threadId && (
              <Link to={recordPath("assistant_thread", answer.threadId)} className="xa">
                {t("Open the conversation")}
              </Link>
            )}
          </div>
        </>
      )}
    </ReadSheet>
  );
}

function WorstRatedPanel({ open, onOpenChange, row }: DataTablePanelProps<AgentWorstRatedRow>) {
  return <WorstRatedSheet answer={open ? row : null} onClose={() => onOpenChange(false)} />;
}

/**
 * The answers people liked least in the last 30 days, most disliked first,
 * for every agent or the one a link narrowed it to. Everything shown is what
 * was kept when the person rated, never the live conversation.
 */
export default function WorstRatedTable() {
  const t = useT();
  const navigate = useAIControlNavigation();
  const [agentId] = useQueryState(QUALITY_AGENT_PARAM, qualityAgentParser);
  const columns = useMemo(() => getWorstRatedColumns(t), [t]);
  const graphql = useMemo(() => createAgentWorstRatedTableGraphQLConfig(agentId), [agentId]);

  const table = (
    <DataTable<AgentWorstRatedRow>
      name="Worst-Rated Answer"
      emptyTitle={t("No worst-rated answers yet")}
      queryKey={AGENT_WORST_RATED_LIST_KEY}
      graphql={graphql}
      resource={Resource.AgentFeedback}
      columns={columns}
      TablePanel={WorstRatedPanel}
      enableCreateAction={false}
      enableReadOnlyPanel
      initialColumnVisibility={{ targetType: false }}
    />
  );

  if (!agentId) {
    return table;
  }

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <AgentScope agentId={agentId} onClear={() => navigate({ tab: "quality", view: "ratings" })} />
      {table}
    </div>
  );
}
