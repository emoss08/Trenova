import {
  approvalGate,
  canApprove,
  gateDigest,
  type ApprovalGate,
} from "@/components/assistant/proposal-preview/preview-gate";
import { useProposalPreview } from "@/components/assistant/proposal-preview/use-proposal-preview";
import type {
  DispatchBoardMove,
  DispatchCandidate,
  DispatchMoveProposal,
} from "@/lib/graphql/dispatch-console";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  HelpCircleIcon,
  ReverseLeftIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatClockDurationMs, formatMinutesSpan, formatUnixTime } from "@trenova/shared/lib/date";
import { useId } from "react";
import { formatMiles, hosStrategyMeta } from "./dispatch-vocabulary";

const MAX_NOTE_LENGTH = 2000;

function DryRunLine({ gate, reviewRequired }: { gate: ApprovalGate; reviewRequired: boolean }) {
  const t = useT();

  switch (gate.state) {
    case "loading":
      return (
        <div className="flex items-start gap-1.5">
          <span className="ui-shimmer rounded-sm text-transparent">{t("Dry run: checking")}</span>
        </div>
      );
    case "ready":
      return (
        <div className="flex items-start gap-1.5">
          <CheckCircleIcon
            className="text-success-foreground mt-px size-3.5 shrink-0"
            aria-hidden
          />
          <span>{t("Dry run: would go through as shown")}</span>
        </div>
      );
    case "stale":
      return (
        <div className="flex items-start gap-1.5">
          <AlertTriangleIcon
            className="text-warning-foreground mt-px size-3.5 shrink-0"
            aria-hidden
          />
          <span>
            {t(
              "The move changed since this was proposed, so approving it would be refused. Decline it or pick a driver below.",
            )}
          </span>
        </div>
      );
    case "unavailable":
      return (
        <div className="flex items-start gap-1.5">
          <HelpCircleIcon className="text-muted-foreground mt-px size-3.5 shrink-0" aria-hidden />
          <span>
            {reviewRequired
              ? t("The dry run didn't load, and this change can't be approved without it.")
              : t("The dry run didn't load; approving now is recorded as not reviewed.")}
          </span>
        </div>
      );
  }
}

export type DeclineDraft = { open: boolean; note: string };

/**
 * Nova's pick for a move, decided here or in Decisions: the same proposal, so deciding it
 * once takes it out of both. Approve sends the digest of the dry run on screen, so a move
 * that changed since is refused rather than assigned to a plan nobody saw.
 */
export function NovaProposalPanel({
  move,
  proposal,
  candidate,
  canDecide,
  approval,
  isDeciding,
  decline,
  onDeclineChange,
  onApprove,
  onDecline,
}: {
  move: DispatchBoardMove;
  proposal: DispatchMoveProposal;
  /** The proposed driver as the ranking scores them, when they are in it. */
  candidate: DispatchCandidate | null;
  canDecide: boolean;
  approval: ProposalApproval;
  isDeciding: boolean;
  decline: DeclineDraft;
  onDeclineChange: (draft: DeclineDraft) => void;
  onApprove: (previewDigest: string | undefined) => Promise<void>;
  onDecline: (note: string) => Promise<void>;
}) {
  const t = useT();
  const noteId = useId();

  const { gate, reviewRequired, approvable } = approval;

  const strategy = hosStrategyMeta(candidate?.hosStrategy);
  const verdict = candidate
    ? candidate.blocked
      ? (candidate.findings.find((finding) => finding.severity === "Block")?.message ??
        t("This pairing trips a rule your organization enforces."))
      : strategy && candidate.hosStrategy !== "currentClocks"
        ? candidate.hosRestStartDeadline > 0
          ? t(
              "{0}. Rest must start by {1}.",
              strategy.phrase,
              formatUnixTime(candidate.hosRestStartDeadline),
            )
          : strategy.phrase
        : t("Legal on current clocks.")
    : null;

  return (
    <section
      className="bg-card border-border animate-rise rounded-md border"
      aria-label={t("Nova's proposal")}
    >
      <header className="border-border flex h-8.5 items-center gap-1.5 border-b px-3 text-xs font-semibold">
        <NovaBadge size="sm" label={t("Nova")} />
        {t("Nova proposes")}
        <span className="text-muted-foreground ml-auto text-2xs font-normal">
          {t("{0} · also in Decisions", formatUnixTime(proposal.createdAt))}
        </span>
      </header>
      <div className="flex flex-col gap-2 p-3 text-xs">
        <div>
          <div className="text-sm leading-4.5 font-semibold">
            {proposal.workerName || t("A driver")}
          </div>
          {verdict ? <div className="mt-0.5">{verdict}</div> : null}
        </div>

        {candidate ? (
          <div className="border-border grid grid-cols-3 rounded-md border">
            <div className="border-border border-r px-2.5 py-1.75">
              <span className="block font-mono text-sm leading-4.5 font-medium tabular-nums">
                {formatMiles(candidate.deadheadMiles)}
              </span>
              <span className="text-muted-foreground text-2xs">{t("Empty miles")}</span>
            </div>
            <div className="border-border border-r px-2.5 py-1.75">
              <span className="block font-mono text-sm leading-4.5 font-medium tabular-nums">
                {formatClockDurationMs(candidate.driveRemainingMs)}
              </span>
              <span className="text-muted-foreground text-2xs">{t("Drive left")}</span>
            </div>
            <div className="px-2.5 py-1.75">
              <span className="block font-mono text-sm leading-4.5 font-medium tabular-nums">
                {formatMinutesSpan(candidate.minutesOfSlack)}
              </span>
              <span className="text-muted-foreground text-2xs">
                {candidate.minutesOfSlack < 0 ? t("Late at pickup") : t("Pickup margin")}
              </span>
            </div>
          </div>
        ) : null}

        <p className="text-muted-foreground">{proposal.rationale}</p>

        <div className="flex flex-col gap-1">
          {canDecide ? <DryRunLine gate={gate} reviewRequired={reviewRequired} /> : null}
          <div className="text-muted-foreground flex items-start gap-1.5">
            <ReverseLeftIcon className="mt-px size-3.5 shrink-0" aria-hidden />
            <span>{t("Can be undone: Undo puts {0} back to uncovered", move.proNumber)}</span>
          </div>
        </div>

        {!canDecide ? (
          <p className="text-muted-foreground">
            {t("Deciding proposals needs permission to update agent proposals.")}
          </p>
        ) : decline.open ? (
          <form
            className="animate-expand-in flex flex-col gap-1.5"
            onSubmit={(event) => {
              event.preventDefault();
              if (decline.note.trim()) void onDecline(decline.note.trim());
            }}
          >
            <label htmlFor={noteId} className="text-xs font-medium">
              {t("Why decline? Nova and whoever looks after it will see this.")}
            </label>
            <Textarea
              id={noteId}
              autoFocus
              minRows={2}
              maxLength={MAX_NOTE_LENGTH}
              value={decline.note}
              onChange={(event) => onDeclineChange({ open: true, note: event.target.value })}
              placeholder={t("Driver asked for a day near home")}
            />
            <div className="flex items-center gap-2">
              <Button
                type="submit"
                size="sm"
                disabled={!decline.note.trim() || isDeciding}
                isLoading={isDeciding}
              >
                {t("Decline proposal")}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={isDeciding}
                shortcut="Esc"
                onClick={() => onDeclineChange({ open: false, note: "" })}
              >
                {t("Cancel")}
              </Button>
            </div>
          </form>
        ) : (
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              disabled={!approvable || isDeciding}
              isLoading={isDeciding}
              shortcut="A"
              onClick={() => void onApprove(approval.digest)}
            >
              {t("Approve")}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={isDeciding}
              shortcut="D"
              onClick={() => onDeclineChange({ open: true, note: "" })}
            >
              {t("Decline")}
            </Button>
          </div>
        )}
      </div>
    </section>
  );
}

export type ProposalApproval = {
  gate: ApprovalGate;
  reviewRequired: boolean;
  /** Approve (the button and A) is offered: the same rule for both. */
  approvable: boolean;
  /** The digest of the dry run on screen, sent with the approval. */
  digest: string | undefined;
};

/**
 * Reads the proposal's dry run once for the panel and the A key, so both offer Approve on
 * exactly the same terms.
 */
export function useProposalApproval(
  proposal: DispatchMoveProposal | null,
  canDecide: boolean,
): ProposalApproval {
  const preview = useProposalPreview({
    scope: "approver",
    id: proposal?.id ?? "",
    enabled: canDecide && proposal !== null,
  });
  const gate = approvalGate(preview);
  const reviewRequired = preview.data?.reviewRequired ?? false;

  return {
    gate,
    reviewRequired,
    approvable:
      proposal !== null && canDecide && !proposal.stale && canApprove(gate, reviewRequired),
    digest: gateDigest(gate),
  };
}
