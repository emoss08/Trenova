import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { apiService } from "@/services/api";
import type { FormulaTemplate } from "@/types/formula-template";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, SendIcon, XIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

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
  const config = ACTION_CONFIG[action];
  const Icon = config.icon;

  useEffect(() => {
    if (!open) {
      setComment("");
      setShowCommentError(false);
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
    onError: () => {
      toast.error(`Failed to ${action} template`, {
        description: "Please try again or contact your system administrator.",
      });
    },
  });

  const handleConfirm = () => {
    const trimmedComment = comment.trim();

    if (action === "reject" && !trimmedComment) {
      setShowCommentError(true);
      return;
    }

    mutation.mutate(trimmedComment);
  };

  const commentInvalid = showCommentError && action === "reject" && !comment.trim();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon className="size-4" />
            {config.title}
            {template?.name && <span className="text-muted-foreground">— {template.name}</span>}
          </DialogTitle>
          <DialogDescription>{config.description}</DialogDescription>
        </DialogHeader>

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
