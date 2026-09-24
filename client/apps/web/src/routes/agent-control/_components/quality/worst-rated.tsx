import { NEGATIVE_REASONS, POSITIVE_REASONS } from "@/components/ai-feedback/feedback-reasons";
import { DataTable } from "@/components/data-table/data-table";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { recordPath } from "@/config/record-links";
import {
  AGENT_WORST_RATED_LIST_KEY,
  createAgentWorstRatedTableGraphQLConfig,
  type AgentWorstRatedAnswer,
  type AgentWorstRatedRow,
} from "@/lib/graphql/agent-quality";
import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryState } from "nuqs";
import { useMemo } from "react";
import { Link } from "react-router";
import { QUALITY_AGENT_PARAM, qualityAgentParser } from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { AgentScope } from "./agent-scope";
import { answerQuestion, getWorstRatedColumns } from "./quality-columns";

const REASON_LABEL = new Map<string, string>(
  [...NEGATIVE_REASONS, ...POSITIVE_REASONS].map((option) => [option.value, option.label]),
);

/**
 * What the person saw when they rated an answer down: the question, the
 * answer and the tools it used, as they were kept. Restricted values were
 * replaced before they were kept. The conversation opens only for its owner,
 * because nobody else can read it.
 */
export function WorstRatedDetails({ answer }: { answer: AgentWorstRatedAnswer }) {
  const t = useT();
  const snapshot = answer.sample?.turnSnapshot;

  return (
    <div className="flex flex-col gap-2">
      <DescriptionList layout="split">
        <DescriptionItem label={t("Question")}>
          <span className="whitespace-pre-wrap">{snapshot?.question || "—"}</span>
        </DescriptionItem>
        <DescriptionItem label={t("Answer")}>
          <span className="whitespace-pre-wrap">{snapshot?.answer || "—"}</span>
        </DescriptionItem>
        {snapshot && snapshot.tools.length > 0 ? (
          <DescriptionItem label={t("Tools")}>
            <ul className="flex flex-col gap-0.5">
              {snapshot.tools.map((tool, index) => (
                <li key={`${tool.name}-${index}`} className="text-xs">
                  <span className="font-mono">{tool.name}</span>
                  {tool.summary ? ` — ${tool.summary}` : ""}
                  {tool.failed ? (
                    <Badge variant="danger" className="ml-1.5">
                      {t("Failed")}
                    </Badge>
                  ) : null}
                </li>
              ))}
              {snapshot.omittedTools > 0 ? (
                <li className="text-muted-foreground text-xs">
                  {t("{0, plural, one {# more tool} other {# more tools}}", snapshot.omittedTools)}
                </li>
              ) : null}
            </ul>
          </DescriptionItem>
        ) : null}
        {answer.sample && answer.sample.reasons.length > 0 ? (
          <DescriptionItem label={t("Why")}>
            <div className="flex flex-wrap gap-1">
              {answer.sample.reasons.map((reason) => (
                <Badge key={reason} variant="neutral">
                  {t(REASON_LABEL.get(reason) ?? reason)}
                </Badge>
              ))}
            </div>
          </DescriptionItem>
        ) : null}
        {answer.sample?.comment ? (
          <DescriptionItem label={t("Comment")}>
            <span className="whitespace-pre-wrap">{answer.sample.comment}</span>
          </DescriptionItem>
        ) : null}
      </DescriptionList>
      <div className="flex items-center gap-3 text-xs">
        {snapshot?.redacted ? (
          <span className="text-muted-foreground">
            {t("Restricted values were replaced before this was kept.")}
          </span>
        ) : null}
        {answer.canOpenThread && answer.threadId ? (
          <Link
            to={recordPath("assistant_thread", answer.threadId)}
            className="text-brand ui-focus-ring rounded-sm hover:underline"
          >
            {t("Open the conversation")}
          </Link>
        ) : null}
      </div>
    </div>
  );
}

function WorstRatedPanel({ open, onOpenChange, row }: DataTablePanelProps<AgentWorstRatedRow>) {
  const t = useT();

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row ? answerQuestion(row) || t("Answer") : t("Answer")}
      description={
        row ? t("{0} down, {1} up · {2}", row.negative, row.positive, row.agentName) : undefined
      }
      size="lg"
    >
      {row ? <WorstRatedDetails answer={row} /> : null}
    </DataTablePanelContainer>
  );
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
