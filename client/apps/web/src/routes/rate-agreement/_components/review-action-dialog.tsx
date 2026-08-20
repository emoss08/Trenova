import { apiService } from "@/services/api";
import type { RateAgreementReviewAction } from "@/services/rate";
import { useMutation, useQueryClient } from "@tanstack/react-query";
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
import type { RateAgreement } from "@trenova/shared/types/rate";
import { ArchiveIcon, CheckIcon, PauseIcon, PlayIcon, SendIcon, XIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

const ACTION_CONFIG: Record<
  RateAgreementReviewAction,
  {
    title: string;
    description: string;
    confirmLabel: string;
    loadingLabel: string;
    successMessage: string;
    commentLabel: string;
    commentPlaceholder: string;
    commentRequired: boolean;
    icon: React.ComponentType<{ className?: string }>;
    destructive: boolean;
  }
> = {
  submit: {
    title: "Submit for Review",
    description: "Send this agreement to a reviewer for approval before it can price shipments.",
    confirmLabel: "Submit for Review",
    loadingLabel: "Submitting...",
    successMessage: "Agreement submitted for review",
    commentLabel: "Comment (optional)",
    commentPlaceholder: "Describe what changed and why it needs review",
    commentRequired: false,
    icon: SendIcon,
    destructive: false,
  },
  approve: {
    title: "Approve Agreement",
    description:
      "Approving activates this agreement, and shipments on its lanes start pricing against it.",
    confirmLabel: "Approve",
    loadingLabel: "Approving...",
    successMessage: "Agreement approved and activated",
    commentLabel: "Comment (optional)",
    commentPlaceholder: "Add an approval note",
    commentRequired: false,
    icon: CheckIcon,
    destructive: false,
  },
  reject: {
    title: "Reject Agreement",
    description: "Rejecting returns this agreement to draft so the author can make changes.",
    confirmLabel: "Reject",
    loadingLabel: "Rejecting...",
    successMessage: "Agreement rejected and returned to draft",
    commentLabel: "Comment (required)",
    commentPlaceholder: "Explain why this agreement is being rejected",
    commentRequired: true,
    icon: XIcon,
    destructive: true,
  },
  suspend: {
    title: "Suspend Agreement",
    description:
      "A suspended agreement stops pricing shipments immediately, and can be resumed without another review.",
    confirmLabel: "Suspend",
    loadingLabel: "Suspending...",
    successMessage: "Agreement suspended",
    commentLabel: "Comment (optional)",
    commentPlaceholder: "Why pricing is being paused",
    commentRequired: false,
    icon: PauseIcon,
    destructive: true,
  },
  resume: {
    title: "Resume Agreement",
    description: "The agreement returns to active and its lanes price shipments again.",
    confirmLabel: "Resume",
    loadingLabel: "Resuming...",
    successMessage: "Agreement resumed",
    commentLabel: "Comment (optional)",
    commentPlaceholder: "Add a note",
    commentRequired: false,
    icon: PlayIcon,
    destructive: false,
  },
  archive: {
    title: "Archive Agreement",
    description:
      "Archiving is permanent. The agreement stops pricing and is kept only because quotes point at it.",
    confirmLabel: "Archive",
    loadingLabel: "Archiving...",
    successMessage: "Agreement archived",
    commentLabel: "Comment (optional)",
    commentPlaceholder: "Why this agreement is being retired",
    commentRequired: false,
    icon: ArchiveIcon,
    destructive: true,
  },
};

type ReviewActionDialogProps = {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly action: RateAgreementReviewAction;
  readonly agreement: RateAgreement | null;
};

export function ReviewActionDialog({
  open,
  onOpenChange,
  action,
  agreement,
}: ReviewActionDialogProps) {
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
    mutationFn: (trimmedComment: string) =>
      apiService.rateAgreementService.review(agreement?.id ?? "", action, trimmedComment),
    onSuccess: () => {
      toast.success(config.successMessage);
      void queryClient.invalidateQueries({ queryKey: ["rate-agreement-list"] });
      void queryClient.invalidateQueries({ queryKey: ["rate-agreement"] });
      onOpenChange(false);
    },
    onError: () => {
      toast.error(`Failed to ${action} agreement`, {
        description: "Please try again or contact your system administrator.",
      });
    },
  });

  const handleConfirm = () => {
    const trimmedComment = comment.trim();

    if (config.commentRequired && !trimmedComment) {
      setShowCommentError(true);
      return;
    }

    mutation.mutate(trimmedComment);
  };

  const commentInvalid = showCommentError && config.commentRequired && !comment.trim();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon className="size-4" />
            {config.title}
            {agreement?.name && <span className="text-muted-foreground">— {agreement.name}</span>}
          </DialogTitle>
          <DialogDescription>{config.description}</DialogDescription>
        </DialogHeader>

        <div className="space-y-1.5 py-2">
          <label htmlFor="review-comment" className="text-xs font-medium">
            {config.commentLabel}
          </label>
          <Textarea
            id="review-comment"
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
            <p className="text-2xs text-destructive">A comment is required to {action}</p>
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
