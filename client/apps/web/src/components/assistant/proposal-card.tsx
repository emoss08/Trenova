import { Button } from "@trenova/shared/components/ui/button";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import { toneVar } from "@/components/kpi/tone";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { apiService } from "@/services/api";
import type { AssistantProposal, ProposalDecision } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  CircleAlertIcon,
  CircleCheckIcon,
  CircleSlashIcon,
  FlaskConicalIcon,
  LoaderIcon,
  PauseCircleIcon,
  TriangleAlertIcon,
  XIcon,
} from "lucide-react";
import { m } from "motion/react";
import { presentProposal } from "./proposal-presenters";
import { classifyProposal, type ProposalPresentation } from "./proposal-state";

/**
 * A change the assistant is asking for.
 *
 * The card answers one question — should this run — so it is built around the
 * sentence that answers it and nothing else. What the tool is called, which
 * identifiers it would send and how confident the model claims to be are not
 * inputs to that decision: the first is jargon, the second is a storage key, and
 * the third is a number no model can calibrate. They are either dropped or put
 * behind Details, which still holds the literal payload for anyone who wants it.
 */
export function ProposalCard({
  proposal,
  threadId,
}: {
  proposal: AssistantProposal;
  threadId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const state = classifyProposal(proposal);
  const view = presentProposal(proposal);

  const decideMutation = useApiMutation({
    mutationFn: (decision: ProposalDecision) =>
      apiService.assistantService.decideProposal(proposal.id, decision),
    onSuccess: () => invalidateProposalViews(queryClient, threadId),
    resourceName: "Proposal",
  });

  const awaiting = state === "awaiting";

  // Once a decision is made the card is history, not a question. It keeps the
  // sentence and the outcome and drops everything that existed to help decide.
  // A held proposal is still a question, just not one anybody can answer yet,
  // so it keeps the full card and swaps the buttons for the reason.
  if (!awaiting && state !== "held") {
    return (
      <m.div
        layout
        className="border-border/70 text-muted-foreground flex items-start gap-2 rounded-lg border px-3 py-2 text-xs"
      >
        <OutcomeIcon state={state} />
        <span className="min-w-0 flex-1">
          <span className="text-foreground block">{view.summary}</span>
          <OutcomeLine proposal={proposal} state={state} />
        </span>
      </m.div>
    );
  }

  return (
    <m.div
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-card ring-foreground/10 flex flex-col overflow-hidden rounded-xl ring-1"
    >
      <div className="flex flex-col gap-2 px-3.5 pt-3 pb-2.5">
        <div className="flex items-center gap-2">
          <span
            aria-hidden
            className="size-1.5 shrink-0 rounded-full"
            style={{ backgroundColor: toneVar("warning") }}
          />
          <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs">
            {view.title}
          </span>
          {view.severity && (
            <span
              className="shrink-0 text-xs font-medium"
              style={{ color: toneVar(view.severity.tone) }}
            >
              {view.severity.label}
            </span>
          )}
        </div>

        <p className="text-sm leading-snug">{view.summary}</p>

        {view.highlights.length > 0 && (
          <dl className="flex flex-col gap-1.5 text-xs">
            {view.highlights.map((entry) => (
              <HighlightRow key={entry.label} label={entry.label} value={entry.value} />
            ))}
          </dl>
        )}

        {proposal.rationale !== "" && (
          <p className="text-muted-foreground text-xs leading-relaxed whitespace-pre-wrap">
            {proposal.rationale}
          </p>
        )}
      </div>

      {state === "held" ? (
        <HoldLine hold={proposal.hold} />
      ) : (
        <div className="flex items-center gap-2 px-3 pb-3">
          <Button
            size="sm"
            onClick={() => decideMutation.mutate("Accepted")}
            disabled={decideMutation.isPending}
            isLoading={decideMutation.isPending && decideMutation.variables === "Accepted"}
          >
            <CheckIcon className="size-3.5" />
            {t("Approve")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => decideMutation.mutate("Rejected")}
            disabled={decideMutation.isPending}
            isLoading={decideMutation.isPending && decideMutation.variables === "Rejected"}
          >
            <XIcon className="size-3.5" />
            {t("Reject")}
          </Button>

          <span className="ml-auto flex items-center gap-2">
            {/* Reversibility is only news one way round. Saying a change can be
              undone reassures nobody; saying it cannot is the thing to read. */}
            {!view.reversible && (
              <span
                className="flex items-center gap-1 text-xs"
                style={{ color: toneVar("warning") }}
              >
                <TriangleAlertIcon className="size-3" />
                {t("Permanent")}
              </span>
            )}
          </span>
        </div>
      )}
    </m.div>
  );
}

/**
 * Why nobody can decide yet, naming the switch.
 *
 * There are two switches on two tabs, and the organization-wide pause is on
 * from the day an organization is created. A line that said only "shadow mode"
 * sent people to turn it off on the agent, where it already was, and back to
 * the same refusal.
 */
export function HoldLine({ hold }: { hold: AssistantProposal["hold"] }) {
  const t = useT();
  if (!hold) {
    return null;
  }

  return (
    <p className="text-muted-foreground flex items-center gap-1.5 px-3.5 pb-3 text-xs">
      <PauseCircleIcon className="size-3.5 shrink-0" />
      <span>
        {hold.reason === "AgentShadow" && hold.agentName !== ""
          ? t("On hold: {0} is in shadow mode in AI Control.", hold.agentName)
          : t("On hold: all agents are paused in AI Control.")}
      </span>
    </p>
  );
}

/**
 * A short value reads as a label/value pair; a sentence does not. The old card
 * ran every value through a fixed 7rem gutter, which turned "what the agent
 * found" into a narrow column beside a wall of wrapped text.
 */
const INLINE_VALUE_LIMIT = 48;

function HighlightRow({ label, value }: { label: string; value: string }) {
  if (value.length > INLINE_VALUE_LIMIT) {
    return (
      <div>
        <dt className="text-muted-foreground">{label}</dt>
        <dd className="mt-0.5 break-words">{value}</dd>
      </div>
    );
  }

  return (
    <div className="flex gap-3">
      <dt className="text-muted-foreground w-24 shrink-0 truncate">{label}</dt>
      <dd className="min-w-0 flex-1 break-words">{value}</dd>
    </div>
  );
}

export function OutcomeIcon({ state }: { state: ProposalPresentation }) {
  const className = "mt-px size-3.5 shrink-0";

  switch (state) {
    case "failed":
      return <CircleAlertIcon className={className} style={{ color: toneVar("danger") }} />;
    case "done":
      return <CircleCheckIcon className={className} style={{ color: toneVar("success") }} />;
    case "running":
      return <LoaderIcon className={cn(className, "animate-spin")} />;
    case "simulated":
      return <FlaskConicalIcon className={className} style={{ color: toneVar("info") }} />;
    default:
      return <CircleSlashIcon className={className} />;
  }
}

/**
 * What became of the decision, stated separately from the decision itself.
 *
 * "Approved" and "done" are different facts and the card never runs them
 * together: an approval whose tool failed says so, with the reason, because the
 * approver is the one who needs to know their instruction did not take effect.
 */
function OutcomeLine({
  proposal,
  state,
}: {
  proposal: AssistantProposal;
  state: ProposalPresentation;
}) {
  const t = useT();

  switch (state) {
    case "failed":
      return (
        <span className="block" style={{ color: toneVar("danger") }}>
          {proposal.executionError === ""
            ? t("Approved, but it did not run. Nothing was changed.")
            : t("Approved, but it did not run: {0}", proposal.executionError)}
        </span>
      );
    case "done":
      return (
        <span className="block">
          {proposal.executedAt
            ? t("Done {0}", generateDateTimeStringFromUnixTimestamp(proposal.executedAt))
            : t("Done")}
        </span>
      );
    case "running":
      return <span className="block">{t("Approved. Waiting for it to run.")}</span>;
    case "declined":
      return <span className="block">{t("Rejected. Nothing was changed.")}</span>;
    case "simulated":
      return <SimulationLine simulation={proposal.simulation} />;
    default:
      return <span className="block">{t("Expired without a decision.")}</span>;
  }
}

/**
 * What a simulated write would have changed. The approver cleared it and it
 * did not happen, on purpose, so the line says both and shows the preview
 * rather than a success.
 */
export function SimulationLine({ simulation }: { simulation: AssistantProposal["simulation"] }) {
  const t = useT();

  return (
    <span className="block">
      <span className="block">{t("Simulated: approved, and nothing was changed.")}</span>
      {simulation?.summary ? <span className="block">{simulation.summary}</span> : null}
      {simulation && simulation.changes.length > 0 && (
        <ul className="mt-1 flex flex-col gap-0.5">
          {simulation.changes.map((change) => (
            <li key={change.field} className="tabular-nums">
              {change.field}: {change.from !== "" ? `${change.from} → ` : ""}
              {change.to}
            </li>
          ))}
        </ul>
      )}
    </span>
  );
}
