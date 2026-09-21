import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { describeToolCall } from "@/components/assistant/tool-presentation";
import { toneVar } from "@/components/kpi/tone";
import { fetchAgentEvaluation, type AgentEvaluationDetail } from "@/lib/graphql/agent-evaluations";
import { useQuery } from "@tanstack/react-query";
import { EvaluationStatusBadge, VerdictBadge } from "./agent-badges";
import { readComparison, summarizeComparison, type ReplayMatch } from "./evaluation-comparison";

export function EvaluationDetailDialog({
  evaluationId,
  onClose,
}: {
  evaluationId: string | null;
  onClose: () => void;
}) {
  const t = useT();
  const query = useQuery({
    queryKey: ["agent-evaluation", evaluationId],
    queryFn: ({ signal }) => fetchAgentEvaluation(evaluationId ?? "", { signal }),
    enabled: evaluationId !== null,
    // A replay takes a while; the dialog keeps up with it.
    refetchInterval: (state) =>
      state.state.data?.status === "Pending" || state.state.data?.status === "Running"
        ? 5_000
        : false,
  });

  return (
    <Dialog open={evaluationId !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent size="xl">
        <DialogHeader>
          <DialogTitle>{t("Replay against the current agent")}</DialogTitle>
          <DialogDescription>
            {t(
              "What the agent, as it is now, would have done with the same run. Every write was simulated; nothing was changed.",
            )}
          </DialogDescription>
        </DialogHeader>
        {query.isLoading || !query.data ? (
          <Skeleton className="h-40 w-full" />
        ) : (
          <EvaluationDetail evaluation={query.data} />
        )}
      </DialogContent>
    </Dialog>
  );
}

function EvaluationDetail({ evaluation }: { evaluation: AgentEvaluationDetail }) {
  const t = useT();
  const comparison = readComparison(evaluation.comparison);

  return (
    <div className="flex max-h-[70vh] flex-col gap-4 overflow-y-auto pr-1">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <EvaluationStatusBadge value={evaluation.status} t={t} />
        {evaluation.model !== "" && (
          <span className="text-muted-foreground font-mono">{evaluation.model}</span>
        )}
        <span className="text-muted-foreground">
          {t("Agent version {0}", evaluation.definitionVersion)}
        </span>
        {comparison && (
          <span className="text-muted-foreground">{summarizeComparison(comparison, t)}</span>
        )}
      </div>

      {evaluation.status === "Failed" && (
        <p className="text-sm" style={{ color: toneVar("danger") }}>
          {evaluation.errorMessage || t("The replay did not finish.")}
        </p>
      )}

      {evaluation.input !== "" && (
        <section className="flex flex-col gap-1">
          <h3 className="text-sm font-semibold">{t("What it was asked")}</h3>
          <p className="text-muted-foreground text-xs leading-relaxed whitespace-pre-wrap">
            {evaluation.input}
          </p>
        </section>
      )}

      {comparison && comparison.matches.length > 0 && (
        <section className="flex flex-col gap-1">
          <h3 className="text-sm font-semibold">{t("Original against replay")}</h3>
          <ul className="divide-border flex flex-col divide-y">
            {comparison.matches.map((match, index) => (
              <MatchRow key={`${match.toolName}-${index}`} match={match} />
            ))}
          </ul>
        </section>
      )}

      {evaluation.reply !== "" && (
        <section className="flex flex-col gap-1">
          <h3 className="text-sm font-semibold">{t("What the replay said")}</h3>
          <p className="text-muted-foreground text-xs leading-relaxed whitespace-pre-wrap">
            {evaluation.reply}
          </p>
        </section>
      )}
    </div>
  );
}

const OUTCOME_LABEL: Record<string, string> = {
  accepted: "a person approved it",
  rejected: "a person rejected it",
  undecided: "nobody decided",
};

function MatchRow({ match }: { match: ReplayMatch }) {
  const t = useT();
  const title = describeToolCall(match.toolName, null).title;

  return (
    <li className="flex flex-col gap-1 py-2 first:pt-0 last:pb-0">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm">{title}</span>
        <VerdictBadge value={match.verdict} t={t} />
        {match.originalOutcome && (
          <Badge variant="neutral" appearance="outline" className="text-2xs">
            {t(OUTCOME_LABEL[match.originalOutcome] ?? match.originalOutcome)}
          </Badge>
        )}
      </div>
      <span className="text-muted-foreground text-xs">{verdictLine(match, t)}</span>
      {match.changes && match.changes.length > 0 && (
        <ul className="text-muted-foreground flex flex-col gap-0.5 text-xs tabular-nums">
          {match.changes.map((change) => (
            <li key={change.field}>
              {change.field}: {change.from !== "" ? `${change.from} → ` : ""}
              {change.to}
            </li>
          ))}
        </ul>
      )}
    </li>
  );
}

function verdictLine(match: ReplayMatch, t: ReturnType<typeof useT>): string {
  switch (match.verdict) {
    case "Agreed":
      return t("The replay proposes this again, as a person approved.");
    case "Improved":
      return t("A person rejected this and the replay no longer proposes it.");
    case "Regressed":
      return t("A person approved this and the replay no longer proposes it.");
    case "Repeated":
      return t("A person rejected this and the replay proposes it again.");
    case "Changed":
      return t("The replay proposes the same change with different parameters.");
    case "Added":
      return t("The replay proposes this; the original did not.");
    default:
      return t("Nobody decided on the original, so this says nothing yet.");
  }
}
