import { previewWarningTone } from "@/components/assistant/proposal-preview/preview-warnings";
import { isPreviewConflict } from "@/components/assistant/proposal-preview/preview-gate";
import {
  usePlanPreview,
  useProposalPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import { approvalFailure } from "@/lib/approval-outcome";
import { decideAgentPlan, decideAgentProposal } from "@/lib/graphql/agent-decisions";
import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import {
  commitAgentDecisionNow,
  undoAgentDecision,
  type DeskCaseDecision,
} from "@/lib/graphql/desk-case";
import { agentPreview } from "@/lib/queries/agent-preview";
import { attention } from "@/lib/queries/attention";
import { deskActivity } from "@/lib/queries/desk-activity";
import { deskAttention } from "@/lib/queries/desk-attention";
import { deskCase } from "@/lib/queries/desk-case";
import { watchtower } from "@/lib/queries/watchtower";
import type { AssistantProposal } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef, useState } from "react";
import { useDecisionSteps } from "../attention/use-decision-steps";
import {
  PENDING_DECISIONS_KEY,
  PENDING_DECISION_SUMMARY_KEY,
} from "../decisions/use-pending-decisions";

/** Whether a step's check says its write would be refused as it stands. */
export function previewRefuses(preview: ProposalPreview | undefined | null): boolean {
  return (preview?.warnings ?? []).some((warning) => previewWarningTone(warning) === "danger");
}

export type CaseDecisionAction = "approve" | "decline" | "undo" | "commit";

export type CaseDecision = {
  /** The changes the decision covers, in the order they run. */
  steps: AssistantProposal[];
  /** What each change would do, by proposal, as the server previews it now. */
  previews: ReadonlyMap<string, ProposalPreview>;
  stepsLoading: boolean;
  stepsFailed: boolean;
  previewLoading: boolean;
  /** True when the preview could not be read; the changes are then drawn from the proposal. */
  previewFailed: boolean;
  /** True when a record a step changes moved on since it was proposed. */
  stale: boolean;
  /**
   * The place of the first step the check says would be refused as things
   * stand, or null when every step would go through. It is a warning, never a
   * lock: what would refuse it may be put right before the person approves.
   */
  refusedStep: number | null;
  /** Whether approving waits on a check that is still loading or out of date. */
  blocked: boolean;
  /** The request in flight; null when none is. */
  busy: CaseDecisionAction | null;
  /** Why the last request did not go through, in the server's words. */
  failure: string;
  approve: () => Promise<void>;
  decline: (reason: string) => Promise<void>;
  undo: () => Promise<void>;
  doNow: () => Promise<void>;
};

const EMPTY: ReadonlyMap<string, ProposalPreview> = new Map();

/** How often the case is read again while a plan told to run now has not started. */
const COMMIT_POLL_MS = 1500;
/** How long that is waited on before the button is given back. */
const COMMIT_WAIT_MS = 30_000;

/**
 * Deciding the plan on a record's case. Where the plan stands always comes
 * from the server (`DeskCaseDecision.phase`); this holds only the request in
 * flight and why the last one was refused. An approval carries the digest of
 * the preview drawn in the card and asks for the undo window, so nothing
 * runs until the window closes or the person says "Do it now".
 */
export function useCaseDecision(decision: DeskCaseDecision | null): CaseDecision {
  const t = useT();
  const queryClient = useQueryClient();
  const planId = decision?.kind === "Plan" ? decision.id : null;
  const proposalId = decision?.kind === "Proposal" ? decision.id : null;
  const steps = useDecisionSteps(
    planId !== null ? { planId } : proposalId !== null ? { proposalId } : null,
  );
  const awaiting = decision?.phase === "Awaiting";
  const planPreview = usePlanPreview({
    scope: "approver",
    id: planId ?? "",
    enabled: planId !== null && awaiting,
  });
  const proposalPreview = useProposalPreview({
    scope: "approver",
    id: proposalId ?? "",
    enabled: proposalId !== null && awaiting,
  });
  const [busy, setBusy] = useState<CaseDecisionAction | null>(null);
  const [failure, setFailure] = useState("");
  const sending = useRef(false);
  // "Do it now" is accepted before the plan starts: the server answers at
  // once and the case still reads Undoable until the run picks it up. The
  // button stays working for the decision it was pressed on until the case
  // says it has moved on, so it never falls back to "Do it now" in between.
  const [committed, setCommitted] = useState<string | null>(null);
  const waitingOnCommit =
    committed !== null && decision?.id === committed && decision.phase === "Undoable";
  useEffect(() => {
    if (!waitingOnCommit) {
      return;
    }
    const started = Date.now();
    const timer = setInterval(() => {
      if (Date.now() - started > COMMIT_WAIT_MS) {
        setCommitted(null);
        return;
      }
      void queryClient.invalidateQueries({ queryKey: deskCase._def });
    }, COMMIT_POLL_MS);

    return () => clearInterval(timer);
  }, [queryClient, waitingOnCommit]);

  let previews = EMPTY;
  let digest = "";
  let stale = false;
  if (planId !== null && planPreview.data) {
    previews = new Map(planPreview.data.steps.map((step) => [step.proposalId, step.preview]));
    digest = planPreview.data.digest;
    stale = planPreview.data.stale;
  } else if (proposalId !== null && proposalPreview.data) {
    previews = new Map([[proposalId, proposalPreview.data]]);
    digest = proposalPreview.data.digest;
    stale = proposalPreview.data.stale;
  }
  const preview = planId !== null ? planPreview : proposalPreview;
  const previewLoading = awaiting && preview.isPending && preview.fetchStatus !== "idle";
  const previewFailed = awaiting && preview.isError;

  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: deskCase._def }),
      queryClient.invalidateQueries({ queryKey: deskAttention._def }),
      queryClient.invalidateQueries({ queryKey: deskActivity._def }),
      queryClient.invalidateQueries({ queryKey: watchtower._def }),
      queryClient.invalidateQueries({ queryKey: attention._def }),
      queryClient.invalidateQueries({ queryKey: agentPreview._def }),
      queryClient.invalidateQueries({ queryKey: ["agent-proposal-list"] }),
      queryClient.invalidateQueries({ queryKey: [PENDING_DECISIONS_KEY] }),
      queryClient.invalidateQueries({ queryKey: [PENDING_DECISION_SUMMARY_KEY] }),
    ]);

  const run = async (action: CaseDecisionAction, send: () => Promise<string | null>) => {
    if (decision === null || sending.current) {
      return;
    }
    sending.current = true;
    setBusy(action);
    setFailure("");
    try {
      const why = await send();
      if (why !== null) {
        setFailure(why);
      }
    } catch (error) {
      setFailure(
        isPreviewConflict(error)
          ? t("This changed since you opened it. Look at it again before you approve.")
          : error instanceof Error
            ? error.message
            : t("That didn't go through."),
      );
    } finally {
      await refresh();
      sending.current = false;
      setBusy(null);
    }
  };

  const decide = (accepted: boolean, reason: string) => async () => {
    const input = {
      decision: accepted ? ("Accepted" as const) : ("Rejected" as const),
      reasonCode: accepted ? "approved_from_case" : "declined_from_case",
      note: accepted ? undefined : reason,
      previewDigest: accepted && digest !== "" ? digest : undefined,
      undoWindow: accepted,
    };
    if (planId !== null) {
      const { plan, refused } = await decideAgentPlan(planId, input);
      if (accepted && plan.status === "Failed" && refused) {
        return t("It was refused because the record changed: {0}", plan.failureError ?? "");
      }

      return null;
    }
    if (proposalId !== null) {
      const result = await decideAgentProposal(proposalId, input);

      return accepted ? approvalFailure(result) : null;
    }

    return null;
  };

  const target = planId !== null ? { planId } : { proposalId: proposalId ?? "" };
  const refusedAt = awaiting
    ? (steps.data ?? []).findIndex((step) => previewRefuses(previews.get(step.id)))
    : -1;
  const refusedStep = refusedAt >= 0 ? refusedAt : null;

  return {
    steps: steps.data ?? [],
    previews,
    stepsLoading: steps.isPending && steps.fetchStatus !== "idle",
    stepsFailed: steps.isError,
    previewLoading,
    previewFailed,
    stale,
    refusedStep,
    blocked: awaiting && (previewLoading || stale),
    busy: busy ?? (waitingOnCommit ? "commit" : null),
    failure,
    approve: () => run("approve", decide(true, "")),
    decline: (reason) => run("decline", decide(false, reason)),
    undo: () =>
      run("undo", async () => {
        await undoAgentDecision(target);
        return null;
      }),
    doNow: () =>
      run("commit", async () => {
        await commitAgentDecisionNow(target);
        setCommitted(decision?.id ?? null);
        return null;
      }),
  };
}
