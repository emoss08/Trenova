import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { describeApiError } from "@/lib/api-error-message";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, MessageSquareWarningIcon, SendIcon, XIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { ApprovalImpactPanel } from "./approval-impact-panel";
import { ReadinessPanel } from "./readiness-panel";
import { ReviewDiffPanel } from "./review-diff-panel";
import { ReviewHistory } from "./review-history";

export type ApprovalAction = "submit" | "approve" | "reject" | "requestChanges";

/** Actions whose comment is the whole point, so it cannot be left blank. */
export const COMMENT_REQUIRED_ACTIONS: ReadonlySet<ApprovalAction> = new Set([
  "reject",
  "requestChanges",
]);

const ACTION_CONFIG: Record<
  ApprovalAction,
  {
    title: string;
    description: string;
    confirmLabel: string;
    loadingLabel: string;
    successMessage: string;
    commentLabel: string;
    commentPlaceholder: string;
    icon: React.ComponentType<{ className?: string }>;
    destructive: boolean;
  }
> = {
  submit: {
    title: "Submit for Review",
    description: "Send this template to a reviewer for approval before it can be activated.",
    confirmLabel: "Submit for Review",
    loadingLabel: "Submitting...",
    successMessage: "Template submitted for review",
    commentLabel: "Comment (optional)",
    commentPlaceholder: "Describe what changed and why it needs review",
    icon: SendIcon,
    destructive: false,
  },
  approve: {
    title: "Approve Template",
    description: "Approving activates this template so it can be used to rate shipments.",
    confirmLabel: "Approve",
    loadingLabel: "Approving...",
    successMessage: "Template approved and activated",
    commentLabel: "Comment (optional)",
    commentPlaceholder: "Add an approval note",
    icon: CheckIcon,
    destructive: false,
  },
  reject: {
    title: "Reject Template",
    description:
      "Rejecting closes this review round and archives the template; it cannot rate shipments until someone resubmits it from the archive. Use Request Changes to send it back to the author instead.",
    confirmLabel: "Reject",
    loadingLabel: "Rejecting...",
    successMessage: "Template rejected and archived",
    commentLabel: "Comment (required)",
    commentPlaceholder: "Explain why this template is being rejected",
    icon: XIcon,
    destructive: true,
  },
  requestChanges: {
    title: "Request Changes",
    description:
      "Send the template back to its author with what needs fixing. The round stays open, so their resubmission continues this review.",
    confirmLabel: "Request Changes",
    loadingLabel: "Sending...",
    successMessage: "Changes requested; the author has been notified",
    commentLabel: "What needs to change (required)",
    commentPlaceholder: "e.g. Guard totalWeight with coalesce; the hazmat surcharge should be $200",
    icon: MessageSquareWarningIcon,
    destructive: false,
  },
};

type ApprovalActionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  action: ApprovalAction;
  template: FormulaTemplate | null;
};

export function ApprovalActionDialog({
  open,
  onOpenChange,
  action,
  template,
}: ApprovalActionDialogProps) {
  const queryClient = useQueryClient();
  const [comment, setComment] = useState("");
  const [showCommentError, setShowCommentError] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);
  // null while the check is loading; true when the server says go, or when
  // the check itself failed and the server must be the judge.
  const [ready, setReady] = useState<boolean | null>(null);
  const config = ACTION_CONFIG[action];
  const Icon = config.icon;
  const gated = action === "submit" || action === "approve";

  useEffect(() => {
    if (!open) {
      setComment("");
      setShowCommentError(false);
      setServerError(null);
      setReady(null);
    }
  }, [open]);

  const mutation = useMutation({
    mutationFn: (trimmedComment: string) => {
      const templateId = template?.id ?? "";

      switch (action) {
        case "submit":
          return apiService.formulaTemplateService.submit(templateId, trimmedComment);
        case "approve":
          return apiService.formulaTemplateService.approve(templateId, trimmedComment);
        case "reject":
          return apiService.formulaTemplateService.reject(templateId, trimmedComment);
        case "requestChanges":
          return apiService.formulaTemplateService.requestChanges(templateId, trimmedComment);
      }
    },
    onSuccess: () => {
      toast.success(config.successMessage);
      void invalidateFormulaTemplate(queryClient);
      onOpenChange(false);
    },
    onError: (error) => {
      // The server says exactly why a review step was refused: a self
      // approval, a failing scenario, an invalid expression. That reason is
      // the whole point of the dialog, so it stays on screen.
      const reason = describeApiError(error, `Failed to ${action} the template.`);
      setServerError(reason);
      toast.error(`Could not ${action} template`, { description: reason });
    },
  });

  const handleConfirm = () => {
    const trimmedComment = comment.trim();

    if (COMMENT_REQUIRED_ACTIONS.has(action) && !trimmedComment) {
      setShowCommentError(true);
      return;
    }

    setServerError(null);
    mutation.mutate(trimmedComment);
  };

  const commentInvalid =
    showCommentError && COMMENT_REQUIRED_ACTIONS.has(action) && !comment.trim();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className={
          action === "reject" || action === "requestChanges"
            ? "sm:max-w-[420px]"
            : "flex max-h-[85vh] flex-col overflow-y-auto sm:max-w-[600px]"
        }
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon className="size-4" />
            {config.title}
            {template?.name && <span className="text-muted-foreground">— {template.name}</span>}
          </DialogTitle>
          <DialogDescription>{config.description}</DialogDescription>
        </DialogHeader>

        {gated && open && template?.id && (
          <ReadinessPanel templateId={template.id} step={action} onReadinessChange={setReady} />
        )}

        {gated && open && template?.id && <ReviewDiffPanel templateId={template.id} />}

        {action === "approve" && open && template?.id && (
          <ApprovalImpactPanel templateId={template.id} />
        )}

        {gated && open && template?.id && <ReviewHistory templateId={template.id} />}

        <div className="space-y-1.5 py-2">
          <label htmlFor="approval-comment" className="text-xs font-medium">
            {config.commentLabel}
          </label>
          <Textarea
            id="approval-comment"
            value={comment}
            onChange={(e) => {
              setComment(e.target.value);
              setShowCommentError(false);
            }}
            placeholder={config.commentPlaceholder}
            minRows={3}
            maxRows={6}
            isInvalid={commentInvalid}
          />
          {commentInvalid && (
            <p className="text-2xs text-destructive">
              {action === "reject"
                ? "A comment is required to reject"
                : "Say what needs to change before sending it back"}
            </p>
          )}
        </div>

        {serverError && (
          <div
            role="alert"
            className="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-3 py-2 text-xs"
          >
            {serverError}
          </div>
        )}

        <DialogFooter>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button
            size="sm"
            variant={config.destructive ? "destructive" : "default"}
            onClick={handleConfirm}
            isLoading={mutation.isPending}
            loadingText={config.loadingLabel}
            disabled={gated && ready === false}
          >
            {config.confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
