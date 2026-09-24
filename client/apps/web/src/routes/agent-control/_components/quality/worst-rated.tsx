import { NEGATIVE_REASONS, POSITIVE_REASONS } from "@/components/ai-feedback/feedback-reasons";
import { SectionTable } from "@/components/data-table/section-table";
import { SectionPanel } from "@/components/section-panel";
import { recordPath } from "@/config/record-links";
import type { AgentWorstRatedAnswer } from "@/lib/graphql/agent-quality";
import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useMemo } from "react";
import { Link } from "react-router";
import { useQualityPages } from "./use-quality-pages";

const RATED_FORMAT = { month: "short", day: "numeric" } as const;

const REASON_LABEL = new Map<string, string>(
  [...NEGATIVE_REASONS, ...POSITIVE_REASONS].map((option) => [option.value, option.label]),
);

const answerRowId = (answer: AgentWorstRatedAnswer) =>
  `${answer.targetType}:${answer.targetId}:${answer.targetPart}`;

function question(answer: AgentWorstRatedAnswer): string {
  return answer.sample?.turnSnapshot?.question.trim() ?? "";
}

function worstRatedColumns(t: TranslateFn): ColumnDef<AgentWorstRatedAnswer>[] {
  return [
    {
      id: "question",
      header: t("Question"),
      cell: ({ row }) => (
        <span className="line-clamp-2">
          {question(row.original) || t("What was asked was not kept")}
        </span>
      ),
      meta: { cellClassName: "max-w-96" },
    },
    {
      id: "agent",
      header: t("Agent"),
      cell: ({ row }) => <span className="truncate">{row.original.agentName || "—"}</span>,
    },
    {
      id: "ratings",
      header: t("Down / up"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.negative} / {row.original.positive}
        </span>
      ),
    },
    {
      id: "rated",
      header: t("Last rated"),
      cell: ({ row }) => (
        <span className="text-muted-foreground tabular-nums">
          {formatUnixInUserTimezone(row.original.lastRatedAt, RATED_FORMAT)}
        </span>
      ),
    },
  ];
}

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
    <div className="flex flex-col gap-2 px-3 py-2">
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

const renderDetails = (answer: AgentWorstRatedAnswer) => <WorstRatedDetails answer={answer} />;

/**
 * The answers people liked least, a page at a time, most disliked first.
 * Everything shown is what was kept when the person rated, never the live
 * conversation.
 */
export function WorstRatedPanel({
  agentDefinitionId = null,
}: {
  agentDefinitionId?: string | null;
}) {
  const t = useT();
  const columns = useMemo(() => worstRatedColumns(t), [t]);
  const { query, rows, pagination } = useQualityPages(
    `worst:${agentDefinitionId ?? "all"}`,
    (page) => queries.agentQuality.worstRated(agentDefinitionId, page),
  );

  return (
    <SectionPanel
      title={t("Worst-rated answers")}
      help={t(
        "Answers people rated down in the last 30 days, the most disliked first, with what they saw when they rated. Restricted values were replaced before anything was kept.",
      )}
    >
      <SectionTable
        label={t("Worst-rated answers")}
        columns={columns}
        rows={rows}
        getRowId={answerRowId}
        rowLabel={(answer) => question(answer) || t("Answer")}
        renderDetails={renderDetails}
        isLoading={query.isPending}
        isRefreshing={query.isPlaceholderData}
        error={query.isError ? t("Worst-rated answers could not be loaded.") : null}
        onRetry={() => void query.refetch()}
        empty={t("Nobody has rated an answer down in the last 30 days.")}
        pagination={pagination}
      />
    </SectionPanel>
  );
}
