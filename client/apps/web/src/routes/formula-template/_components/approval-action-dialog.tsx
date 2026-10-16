import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { DateTimePicker } from "@/components/fields/date-field/datetime-picker";
import { fromUserWallClock } from "@trenova/shared/lib/date";
import { describeApiError } from "@/lib/api-error-message";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  MessageAlertSquareIcon,
  Send01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { ApprovalImpactPanel } from "./approval-impact-panel";
import { ReadinessPanel } from "./readiness-panel";
import { ReviewDiffPanel } from "./review-diff-panel";
import { ReviewHistory } from "./review-history";
import { defineLabels, type LabelMap } from "@trenova/shared/i18n/labels";

export type ApprovalAction = "submit" | "approve" | "reject" | "requestChanges";

/** Actions whose comment is the whole point, so it cannot be left blank. */
export const COMMENT_REQUIRED_ACTIONS: ReadonlySet<ApprovalAction> = new Set([
  "reject",
  "requestChanges",
]);

const ACTION_CONFIG: Record<
  ApprovalAction,
  {
    copy: LabelMap<{
      title: string;
      description: string;
      confirmLabel: string;
      loadingLabel: string;
      failureTitle: string;
      failureFallback: string;
      commentLabel: string;
      commentPlaceholder: string;
    }>;
    icon: React.ComponentType<{ className?: string }>;
    destructive: boolean;
  }
> = {
  submit: {
    copy: defineLabels({
      title: "Submit for review",
      description: "Send this template to a reviewer for approval before it can be activated.",
      confirmLabel: "Submit for review",
      loadingLabel: "Submitting...",
      failureTitle: "Could not submit template",
      failureFallback: "Failed to submit the template.",
      commentLabel: "Comment (optional)",
      commentPlaceholder: "Describe what changed and why it needs review",
    }),
    icon: Send01Icon,
    destructive: false,
  },
  approve: {
    copy: defineLabels({
      title: "Approve template",
      description:
        "Approving makes this content the one that rates shipments, from now or from the date you choose.",
      confirmLabel: "Approve",
      loadingLabel: "Approving...",
      failureTitle: "Could not approve template",
      failureFallback: "Failed to approve the template.",
      commentLabel: "Comment (optional)",
      commentPlaceholder: "Add an approval note",
    }),
    icon: CheckIcon,
    destructive: false,
  },
  reject: {
    copy: defineLabels({
      title: "Reject template",
      description:
        "Rejecting closes this review round. A template that was approved before returns to draft and its approved content keeps billing; one that never was is archived. Use Request Changes to send it back to the author instead.",
      confirmLabel: "Reject",
      loadingLabel: "Rejecting...",
      failureTitle: "Could not reject template",
      failureFallback: "Failed to reject the template.",
      commentLabel: "Comment (required)",
      commentPlaceholder: "Explain why this template is being rejected",
    }),
    icon: XCloseIcon,
    destructive: true,
  },
  requestChanges: {
    copy: defineLabels({
      title: "Request changes",
      description:
        "Send the template back to its author with what needs fixing. The round stays open, so their resubmission continues this review.",
      confirmLabel: "Request changes",
      loadingLabel: "Sending...",
      failureTitle: "Could not request changes on template",
      failureFallback: "Failed to request changes on the template.",
      commentLabel: "What needs to change (required)",
      commentPlaceholder:
        "e.g. Guard totalWeight with coalesce; the hazmat surcharge should be $200",
    }),
    icon: MessageAlertSquareIcon,
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
  const t = useT();

  const queryClient = useQueryClient();
  const [comment, setComment] = useState("");
  const [showCommentError, setShowCommentError] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);
  const [effectiveFrom, setEffectiveFrom] = useState<Date | undefined>(undefined);
  // null while the check is loading; true when the server says go, or when
  // the check itself failed and the server must be the judge.
  const [ready, setReady] = useState<boolean | null>(null);
  const config = ACTION_CONFIG[action];
  const copy = config.copy;
  const Icon = config.icon;
  const gated = action === "submit" || action === "approve";

  useEffect(() => {
    if (!open) {
      setComment("");
      setShowCommentError(false);
      setServerError(null);
      setReady(null);
      setEffectiveFrom(undefined);
    }
  }, [open]);

  const mutation = useMutation({
    mutationFn: (trimmedComment: string) => {
      const templateId = template?.id ?? "";

      switch (action) {
        case "submit":
          return apiService.formulaTemplateService.submit(templateId, trimmedComment);
        case "approve":
          return apiService.formulaTemplateService.approve(
            templateId,
            trimmedComment,
            fromUserWallClock(effectiveFrom),
          );
        case "reject":
          return apiService.formulaTemplateService.reject(templateId, trimmedComment);
        case "requestChanges":
          return apiService.formulaTemplateService.requestChanges(templateId, trimmedComment);
      }
    },
    onSuccess: () => {
      void invalidateFormulaTemplate(queryClient);
      onOpenChange(false);
    },
    onError: (error) => {
      // The server says exactly why a review step was refused: a self
      // approval, a failing scenario, an invalid expression. That reason is
      // the whole point of the dialog, so it stays on screen.
      const reason = describeApiError(error, copy.failureFallback);
      setServerError(reason);
      toast.error(copy.failureTitle, { description: reason });
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

  const compact = action === "reject" || action === "requestChanges";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size={compact ? "sm" : "lg"}
        className="max-h-[85vh] grid-rows-[auto_minmax(0,1fr)_auto]"
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon className="size-4" />
            {copy.title}
            {template?.name && <span className="text-muted-foreground">— {template.name}</span>}
          </DialogTitle>
          <DialogDescription>{copy.description}</DialogDescription>
        </DialogHeader>

        <ScrollArea className="-mx-4 min-h-0">
          <div className="space-y-4 px-4">
            {gated && open && template?.id && (
              <ReadinessPanel templateId={template.id} step={action} onReadinessChange={setReady} />
            )}

            {gated && open && template?.id && <ReviewDiffPanel templateId={template.id} />}

            {action === "approve" && open && template?.id && (
              <ApprovalImpactPanel templateId={template.id} />
            )}

            {gated && open && template?.id && <ReviewHistory templateId={template.id} />}

            {action === "approve" && (
              <div className="space-y-1.5">
                <label htmlFor="approval-effective-from" className="text-xs font-medium">
                  {t("Start billing on (optional)")}
                </label>
                <DateTimePicker
                  id="approval-effective-from"
                  dateTime={effectiveFrom}
                  setDateTime={setEffectiveFrom}
                  placeholder={t("Now")}
                  clearable
                />
                <p className="text-2xs text-muted-foreground">
                  {t(
                    "Leave empty to bill with it as soon as it is approved. With a date, the approved content in force today keeps billing until then.",
                  )}
                </p>
              </div>
            )}

            <div className="space-y-1.5 py-2">
              <label htmlFor="approval-comment" className="text-xs font-medium">
                {copy.commentLabel}
              </label>
              <Textarea
                id="approval-comment"
                value={comment}
                onChange={(e) => {
                  setComment(e.target.value);
                  setShowCommentError(false);
                }}
                placeholder={copy.commentPlaceholder}
                minRows={3}
                maxRows={6}
                isInvalid={commentInvalid}
              />
              {commentInvalid && (
                <p className="text-2xs text-destructive">
                  {action === "reject"
                    ? t("A comment is required to reject")
                    : t("Say what needs to change before sending it back")}
                </p>
              )}
            </div>

            {serverError && (
              <Alert variant="destructive" size="sm">
                <AlertDescription>{serverError}</AlertDescription>
              </Alert>
            )}
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            {t("Cancel")}
          </Button>
          <Button
            variant={config.destructive ? "destructive" : "default"}
            onClick={handleConfirm}
            isLoading={mutation.isPending}
            loadingText={copy.loadingLabel}
            disabled={gated && ready === false}
          >
            {copy.confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
