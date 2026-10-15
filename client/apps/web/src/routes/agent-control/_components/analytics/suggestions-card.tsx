import type {
  AIAnalyticsAgentSuggestions,
  AIAnalyticsSuggestions,
} from "@/lib/graphql/ai-analytics";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { memo } from "react";
import { percent } from "./analytics-model";
import { AnalyticsCard } from "./card";

type Kind = "a" | "m" | "u" | "r" | "i";

function parts(row: AIAnalyticsAgentSuggestions): readonly (readonly [Kind, number])[] {
  return [
    ["a", row.switched],
    ["m", row.sentThrough],
    ["u", row.autoSwitched],
    ["r", row.kept],
    ["i", row.passedOver],
  ];
}

/**
 * What people did with the agents the Desk named for their questions, by the
 * agent named: switched to it, sent the question to it, let the Desk move the
 * question, kept their own agent, or passed it over. The server counts and
 * rates them; the card only draws.
 */
export const SuggestionsCard = memo(function SuggestionsCard({
  suggestions,
}: {
  suggestions: AIAnalyticsSuggestions;
}) {
  const t = useT();
  const rt = useRichT();

  return (
    <AnalyticsCard
      title={t("Agent suggestions")}
      description={t(
        "How often people took the agent Desk named for a question, by the agent named.",
      )}
      className="an-dc an-sg"
      right={
        <span className="an-sgf">
          {rt(
            "<b>{0}</b> suggested",
            { b: (chunk) => <b className="mono">{chunk}</b> },
            formatNumber(suggestions.suggested),
          )}
          <span aria-hidden> · </span>
          {rt(
            "<b>{0}</b> taken",
            { b: (chunk) => <b className="mono">{chunk}</b> },
            percent(suggestions.takenRate),
          )}
        </span>
      }
    >
      {suggestions.agents.map((row, index) => (
        <div key={row.agentDefinitionId} className="an-dcr" data-slot="suggestion-row">
          <span className="an-dcn">
            <b>{row.name || t("Removed agent")}</b>
            <em className="mono">{formatNumber(row.suggested)}</em>
          </span>
          <span
            className="an-stk"
            role="img"
            aria-label={t(
              "{0} switched, {1} sent to it, {2} moved without asking, {3} kept their own agent, {4} passed it over",
              row.switched,
              row.sentThrough,
              row.autoSwitched,
              row.kept,
              row.passedOver,
            )}
          >
            {parts(row).map(([kind, count]) =>
              count > 0 ? (
                <i
                  key={kind}
                  className={`p-${kind}`}
                  style={{ flexGrow: count, animationDelay: `${index * 50}ms` }}
                />
              ) : null,
            )}
          </span>
          <span className="an-dcv mono">{percent(row.takenRate)}</span>
        </div>
      ))}
      <div className="an-lgd">
        <span>
          <i className="p-a" />
          {t("Switched")}
        </span>
        <span>
          <i className="p-m" />
          {t("Sent to it")}
        </span>
        <span>
          <i className="p-u" />
          {t("Moved without asking")}
        </span>
        <span>
          <i className="p-r" />
          {t("Kept their own")}
        </span>
        <span>
          <i className="p-i" />
          {t("Passed over")}
        </span>
      </div>
    </AnalyticsCard>
  );
});
