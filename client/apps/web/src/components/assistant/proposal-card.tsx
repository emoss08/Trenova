import { Button } from "@trenova/shared/components/ui/button";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { invalidateProposalViews, markProposalDecided } from "@/lib/proposal-cache";
import { apiService } from "@/services/api";
import type { AssistantProposal, ProposalDecision } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  PauseCircleIcon,
  PenLineIcon,
  PencilIcon,
  TriangleAlertIcon,
  XIcon,
} from "lucide-react";
import { useState } from "react";
import { DecisionFrame, DecisionReceipt, ProposedBy, useWatchedChange } from "./decision-chrome";
import { useDecisionFollowUp } from "./decision-follow-up";
import { ProposalEditor, type ProposalEditorRequest } from "./proposal-editor";
import { presentProposal } from "./proposal-presenters";
import { classifyProposal, ranWithoutApproval, type ProposalPresentation } from "./proposal-state";

/**
 * A change the assistant is asking for.
 *
 * The card answers one question — should this run — so it is built around the
 * sentence that answers it and nothing else. What the tool is called, which
 * identifiers it would send and how confident the model claims to be are not
 * inputs to that decision: the first is jargon, the second is a storage key, and
 * the third is a number no model can calibrate. They are either dropped or put
 * behind Details, which still holds the literal payload for anyone who wants it.
 *
 * Once decided — or when it never needed deciding, because the write was one
 * the agent may make on its own — the card becomes a receipt of what came of
 * it, and changes into it in place.
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

  const followUp = useDecisionFollowUp();
  const decideMutation = useApiMutation({
    mutationFn: (decision: ProposalDecision) =>
      apiService.assistantService.decideProposal(proposal.id, decision),
    onSuccess: async (_result, decision) => {
      markProposalDecided(queryClient, proposal.id, decision);
      await invalidateProposalViews(queryClient, threadId);
      // The agent says what came of it. An approval used to end the
      // conversation on a card that read "done", with no word on whether
      // the change existed or where to find it.
      followUp?.(proposal.id);
    },
    // A decision the server refuses almost always means this card is showing
    // a proposal somebody already resolved — in another tab, or by a click
    // this one did not hear about. Refetching turns that into the card
    // catching up rather than a dead end the reader has to reload out of.
    onError: () => void invalidateProposalViews(queryClient, threadId),
    resourceName: "Proposal",
  });
  const [editor, setEditor] = useState<ProposalEditorRequest | null>(null);

  const awaiting = state === "awaiting";
  // Decided while this card was on screen: the receipt that replaces the
  // question rises into its place rather than swapping in.
  const decidedHere = useWatchedChange(awaiting || state === "held");
  const fields = proposal.fields ?? [];
  const editable = awaiting && fields.length > 0;
  const byline = <ProposedBy agentId={proposal.agentId} agentName={proposal.agentName} />;

  // The values open as a form built from the tool's schema; approval carries
  // only what was changed, and the server validates it before recording.
  const openEditor = () =>
    setEditor({
      summary: view.summary,
      fields,
      arguments: proposal.arguments,
      onConfirm: async (modifications) => {
        await apiService.assistantService.decideProposal(proposal.id, "Modified", modifications);
        markProposalDecided(queryClient, proposal.id, "Modified");
        await invalidateProposalViews(queryClient, threadId);
        followUp?.(proposal.id);
      },
    });

  // Once a decision is made the card is history, not a question. It keeps the
  // sentence and the outcome and drops everything that existed to help decide.
  // A held proposal is still a question, just not one anybody can answer yet,
  // so it keeps the full card and swaps the buttons for the reason.
  if (!awaiting && state !== "held") {
    return (
      <DecisionReceipt state={state} summary={view.summary} byline={byline} arrived={decidedHere}>
        <OutcomeLine proposal={proposal} state={state} />
      </DecisionReceipt>
    );
  }

  return (
    <DecisionFrame
      icon={PenLineIcon}
      title={view.title}
      state={state}
      byline={byline}
      footer={
        state === "held" ? (
          <HoldLine hold={proposal.hold} />
        ) : (
          <div className="border-border-subtle flex flex-wrap items-center gap-2 border-t px-3 py-2.5">
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
            {editable && (
              <Button
                size="sm"
                variant="ghost"
                onClick={openEditor}
                disabled={decideMutation.isPending}
              >
                <PencilIcon className="size-3.5" />
                {t("Modify")}
              </Button>
            )}

            {/* Reversibility is only news one way round. Saying a change can be
                undone reassures nobody; saying it cannot is the thing to read. */}
            {!view.reversible && (
              <span className="text-warning ml-auto flex items-center gap-1 text-xs">
                <TriangleAlertIcon className="size-3" />
                {t("Permanent")}
              </span>
            )}
          </div>
        )
      }
    >
      {view.severity && (
        <span className="text-foreground-muted -mt-1 text-xs">
          {t("Severity")}{" "}
          <span className={view.severity.tone === "danger" ? "text-danger" : "text-foreground"}>
            {view.severity.label}
          </span>
        </span>
      )}

      <p className="text-sm leading-snug">{view.summary}</p>

      {view.highlights.length > 0 && (
        <dl className="flex flex-col gap-1.5 text-xs">
          {view.highlights.map((entry) => (
            <HighlightRow key={entry.label} label={entry.label} value={entry.value} />
          ))}
        </dl>
      )}

      {proposal.rationale !== "" && (
        <p className="text-foreground-muted text-xs leading-relaxed whitespace-pre-wrap">
          {proposal.rationale}
        </p>
      )}
      <ProposalEditor request={editor} onClose={() => setEditor(null)} />
    </DecisionFrame>
  );
}

/**
 * What the approver changed before approving, so a collapsed card says what
 * ran rather than only that something did.
 */
function ChangesLine({ modifications }: { modifications: AssistantProposal["modifications"] }) {
  const t = useT();
  const entries = Object.entries(modifications ?? {});
  if (entries.length === 0) {
    return null;
  }

  return (
    <span className="block">
      {t("Approved with changes:")}{" "}
      {entries.map(([key, value], index) => (
        <span key={key}>
          {index > 0 ? ", " : ""}
          {key}: {formatChange(value)}
        </span>
      ))}
    </span>
  );
}

function formatChange(value: unknown): string {
  if (value === null || value === undefined) return "—";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
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
    <p className="border-border-subtle text-foreground-muted flex items-center gap-1.5 border-t px-3 py-2.5 text-xs">
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
        <dt className="text-foreground-subtle">{label}</dt>
        <dd className="mt-0.5 break-words">{value}</dd>
      </div>
    );
  }

  return (
    <div className="flex gap-3">
      <dt className="text-foreground-subtle w-24 shrink-0 truncate">{label}</dt>
      <dd className="min-w-0 flex-1 break-words">{value}</dd>
    </div>
  );
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
  // A write the agent may make on its own was never waiting on anyone, so it
  // is not described as approved: it ran, and the line says so.
  const own = ranWithoutApproval(proposal);

  switch (state) {
    case "failed":
      return (
        <>
          <ChangesLine modifications={proposal.modifications} />
          <span className="text-danger block">
            {own
              ? proposal.executionError === ""
                ? t("It ran on its own but did not go through. Nothing was changed.")
                : t("It ran on its own but did not go through: {0}", proposal.executionError)
              : proposal.executionError === ""
                ? t("Approved, but it did not run. Nothing was changed.")
                : t("Approved, but it did not run: {0}", proposal.executionError)}
          </span>
        </>
      );
    case "done": {
      const when = proposal.executedAt
        ? generateDateTimeStringFromUnixTimestamp(proposal.executedAt)
        : "";
      return (
        <>
          <ChangesLine modifications={proposal.modifications} />
          <span className="block">
            {own
              ? when !== ""
                ? t("Done on its own {0}. No approval was needed.", when)
                : t("Done on its own. No approval was needed.")
              : when !== ""
                ? t("Done {0}", when)
                : t("Done")}
          </span>
        </>
      );
    }
    case "running":
      return (
        <>
          <ChangesLine modifications={proposal.modifications} />
          <span className="block">
            {own
              ? t("Running on its own. No approval is needed.")
              : t("Approved. Waiting for it to run.")}
          </span>
        </>
      );
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
