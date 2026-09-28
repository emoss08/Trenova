import { PlanPreview } from "@/components/assistant/proposal-preview/plan-preview";
import { canApprove, gateDigest } from "@/components/assistant/proposal-preview/preview-gate";
import {
  PreviewLoadState,
  ProposalPreview,
  type PreviewDensity,
} from "@/components/assistant/proposal-preview/proposal-preview";
import {
  useApprovalGate,
  usePlanPreview,
  useProposalPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import type { PreviewScope } from "@/lib/graphql/agent-preview";
import {
  askAgentMessage,
  type WouldFailReason,
} from "@/components/assistant/proposal-preview/preview-warnings";
import type { WouldFailActions } from "@/components/assistant/proposal-preview/proposal-preview";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Label } from "@trenova/shared/components/ui/label";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { useId, useState, type ReactNode } from "react";

/**
 * The write a decision is about, previewed inside the dialog. Approving waits
 * for the preview and sends its digest; rejecting shows it and waits for
 * nothing, because a change that can no longer be approved can still be
 * turned down.
 */
export type ReasonDialogPreview = {
  kind: "proposal" | "plan";
  scope: PreviewScope;
  id: string;
  approving: boolean;
  /** Compact where the surface behind the dialog already shows the preview in full. */
  density?: PreviewDensity;
  /** The sentence a plan step is known by, in place of "Step 2". */
  stepTitle?: (proposalId: string, step: number) => ReactNode;
};

export type ReasonDialogRequest = {
  title: string;
  description: string;
  confirmLabel: string;
  reasonLabel: string;
  /** When true the confirm button stays disabled until a reason is typed. */
  requireReason: boolean;
  destructive?: boolean;
  preview?: ReasonDialogPreview;
  /** What the reason starts as: the reasons a write would be refused over, when the person asked to reject over them. */
  initialReason?: string;
  /**
   * Where the surface sends a person who wants the agent to fix a write the
   * preview says would be refused: an approval dialog closes and opens the
   * rejection with the reasons written. Absent, the dialog writes them into
   * its own reason, which a rejection then carries to the agent.
   */
  onAskAgent?: (reason: string) => void;
  onConfirm: (reason: string, previewDigest: string | undefined) => Promise<void> | void;
};

/** The most a reason may hold. */
const REASON_MAX_LENGTH = 500;

/**
 * A key per request, so a surface that swaps one request for another while
 * the dialog is open (an approval turned into a rejection) gets a fresh form
 * that starts from the new request's reason.
 */
const requestKeys = new WeakMap<ReasonDialogRequest, number>();
let lastRequestKey = 0;

function requestKey(request: ReasonDialogRequest): number {
  const known = requestKeys.get(request);
  if (known !== undefined) {
    return known;
  }
  lastRequestKey += 1;
  requestKeys.set(request, lastRequestKey);

  return lastRequestKey;
}

type ReasonDialogProps = {
  request: ReasonDialogRequest | null;
  onClose: () => void;
};

/**
 * One dialog for every decision the activity tab takes: accepting or
 * rejecting a proposal, reviewing or dismissing an exception. A reason is
 * recorded with each so the audit trail says why, not only what. A decision
 * on a proposal or a plan shows what it would change first.
 */
export function ReasonDialog({ request, onClose }: ReasonDialogProps) {
  return (
    <Dialog open={request !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent size={request?.preview ? "lg" : "md"}>
        {request && <ReasonForm key={requestKey(request)} request={request} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  );
}

// Mounted only while the dialog is open and keyed by its request, so each
// request starts from its own reason without any effect having to reset it.
function ReasonForm({ request, onClose }: { request: ReasonDialogRequest; onClose: () => void }) {
  const t = useT();
  const id = useId();
  const [reason, setReason] = useState(() =>
    (request.initialReason ?? "").slice(0, REASON_MAX_LENGTH),
  );
  const [isPending, setIsPending] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);

  const preview = request.preview;
  const proposalQuery = useProposalPreview({
    scope: preview?.scope ?? "approver",
    id: preview?.id ?? "",
    enabled: preview?.kind === "proposal",
  });
  const planQuery = usePlanPreview({
    scope: preview?.scope ?? "approver",
    id: preview?.id ?? "",
    enabled: preview?.kind === "plan",
  });
  const approval = useApprovalGate(preview?.kind === "plan" ? planQuery : proposalQuery);

  const trimmed = reason.trim();
  const previewAllows = preview?.approving !== true || canApprove(approval.gate);

  // A write that would be refused is turned down with its reasons, so the
  // agent learns what to fix. Nothing here can change a value: that is the
  // editor's, which the surface behind the dialog offers.
  const toolName =
    preview?.kind === "plan"
      ? (planQuery.data?.steps.find((step) =>
          step.preview.warnings.some((entry) => entry.code === "would_fail"),
        )?.preview.tool ?? "")
      : (proposalQuery.data?.tool ?? "");
  const wouldFail: WouldFailActions = {
    onAskAgent: (reasons: WouldFailReason[]) => {
      const text = askAgentMessage(toolName, reasons, t).slice(0, REASON_MAX_LENGTH);
      if (request.onAskAgent) {
        request.onAskAgent(text);
      } else {
        setReason(text);
      }
    },
  };

  const canConfirm = !isPending && (!request.requireReason || trimmed !== "") && previewAllows;

  const confirm = async () => {
    if (!canConfirm) {
      return;
    }
    approval.acknowledge();
    setIsPending(true);
    setFailure(null);
    try {
      await request.onConfirm(trimmed, preview ? gateDigest(approval.gate) : undefined);
      onClose();
    } catch (error) {
      // What the write would do moved while the dialog was open: nothing was
      // recorded, the preview is read again, and the dialog stays on it.
      if (!(preview && approval.handleDecisionError(error))) {
        setFailure(graphQLErrorMessage(error, t("The decision could not be recorded.")));
      }
    } finally {
      setIsPending(false);
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>{request.title}</DialogTitle>
        <DialogDescription>{request.description}</DialogDescription>
      </DialogHeader>
      {preview && (
        <section className="flex max-h-[50vh] min-w-0 flex-col gap-2 overflow-y-auto">
          <h3 className="text-foreground-subtle text-xs font-medium">{t("What changes")}</h3>
          {preview.kind === "plan" ? (
            <PreviewLoadState
              query={planQuery}
              changed={approval.changed}
              density={preview.density}
            >
              {(plan) => (
                <PlanPreview
                  plan={plan}
                  density={preview.density}
                  stepTitle={preview.stepTitle}
                  wouldFail={wouldFail}
                />
              )}
            </PreviewLoadState>
          ) : (
            <PreviewLoadState
              query={proposalQuery}
              changed={approval.changed}
              density={preview.density}
            >
              {(data) => (
                <ProposalPreview preview={data} density={preview.density} wouldFail={wouldFail} />
              )}
            </PreviewLoadState>
          )}
        </section>
      )}
      <div className="flex flex-col gap-1.5">
        <Label htmlFor={id}>
          {request.reasonLabel}
          {request.requireReason ? "" : ` (${t("optional")})`}
        </Label>
        <Textarea
          id={id}
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          minRows={3}
          maxLength={REASON_MAX_LENGTH}
          placeholder={t("A sentence the next person can act on.")}
        />
      </div>
      {failure && <p className="text-danger text-xs">{failure}</p>}
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={isPending}>
          {t("Cancel")}
        </Button>
        <Button
          type="button"
          variant={request.destructive ? "destructive" : "default"}
          onClick={() => void confirm()}
          disabled={!canConfirm}
          isLoading={isPending}
        >
          {request.confirmLabel}
        </Button>
      </DialogFooter>
    </>
  );
}
