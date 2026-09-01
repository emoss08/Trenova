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
import { apiService } from "@/services/api";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, SendIcon, XIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { ApprovalImpactPanel } from "./approval-impact-panel";

export type ApprovalAction = "submit" | "approve" | "reject";

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
    description: "Rejecting returns this template to draft so the author can make changes.",
    confirmLabel: "Reject",
    loadingLabel: "Rejecting...",
    successMessage: "Template rejected and returned to draft",
    commentLabel: "Comment (required)",
    commentPlaceholder: "Explain why this template is being rejected",
    icon: XIcon,
    destructive: true,
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
  const config = ACTION_CONFIG[action];
  const Icon = config.icon;

  useEffect(() => {
    if (!open) {
      setComment("");
      setShowCommentError(false);
      setServerError(null);
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
      }
    },
    onSuccess: () => {
      toast.success(config.successMessage);
      void queryClient.invalidateQueries({ queryKey: ["formula-template-list"] });
      void queryClient.invalidateQueries({ queryKey: ["formulaTemplate"] });
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

    if (action === "reject" && !trimmedComment) {
      setShowCommentError(true);
      return;
    }

    setServerError(null);
    mutation.mutate(trimmedComment);
  };

  const commentInvalid = showCommentError && action === "reject" && !comment.trim();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={action === "approve" ? "sm:max-w-[520px]" : "sm:max-w-[420px]"}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon className="size-4" />
            {config.title}
            {template?.name && <span className="text-muted-foreground">— {template.name}</span>}
          </DialogTitle>
          <DialogDescription>{config.description}</DialogDescription>
        </DialogHeader>

        {action === "approve" && open && template?.id && (
          <ApprovalImpactPanel templateId={template.id} />
        )}

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
            <p className="text-2xs text-destructive">A comment is required to reject</p>
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
          >
            {config.confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
