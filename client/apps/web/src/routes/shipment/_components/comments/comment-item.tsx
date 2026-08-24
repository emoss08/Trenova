import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import {
  CommentEditor,
  type CommentEditorHandle,
} from "@trenova/shared/components/comment-editor/comment-editor";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@trenova/shared/components/ui/tooltip";
import {
  commentPriorityChoices,
  commentTypeChoices,
  commentTypeColor,
  commentTypeLabel,
  commentVisibilityChoices,
} from "@trenova/shared/lib/comment-choices";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation } from "@trenova/shared/types/permission";
import { formatDistanceToNow, fromUnixTime } from "date-fns";
import {
  CheckCircle2Icon,
  CheckIcon,
  CircleSlashIcon,
  EllipsisVerticalIcon,
  EyeIcon,
  FlagIcon,
  LoaderIcon,
  PencilIcon,
  PinIcon,
  PinOffIcon,
  ReplyIcon,
  RotateCcwIcon,
  TagIcon,
  Trash2Icon,
  TrashIcon,
  UndoIcon,
  XIcon,
} from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  fetchAllEntityRefCandidates,
  fetchMentionCandidates,
} from "@/hooks/shipment-comments/use-comment-suggestions";
import { usePermission } from "@/hooks/use-permission";
import type { LocalShipmentComment } from "@/lib/shipment-comment-cache";
import type {
  CommentPriority,
  CommentType,
  CommentVisibility,
  ShipmentCommentUpdateInput,
} from "@/types/shipment-comment";
import { AcknowledgeRow } from "./acknowledge-row";
import { CommentAttachments } from "./comment-attachments";
import { CommentBody } from "./comment-body";
import { CommentOptionPill } from "./option-pills";

const PRIORITY_INDICATOR: Record<CommentPriority, { icon: string; className: string } | null> = {
  Low: null,
  Normal: null,
  High: {
    icon: "▲",
    className: "border-amber-600 bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400",
  },
  Urgent: {
    icon: "!",
    className: "border-red-600 bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400",
  },
};

export interface CommentItemActions {
  onReply?: () => void;
  onSaveEdit: (input: ShipmentCommentUpdateInput) => void;
  isUpdating: boolean;
  onDelete: () => void;
  isDeleting: boolean;
  onTogglePin?: (pinned: boolean) => void;
  onToggleResolve: (resolved: boolean) => void;
  onAcknowledge: () => void;
  isAcknowledging: boolean;
  onRetry: () => void;
  onDiscard: () => void;
}

export function CommentItem({
  comment,
  isEditing,
  onEdit,
  onCancelEdit,
  actions,
  isOnline,
  highlighted = false,
}: {
  comment: LocalShipmentComment;
  isEditing: boolean;
  onEdit: () => void;
  onCancelEdit: () => void;
  actions: CommentItemActions;
  isOnline: boolean;
  highlighted?: boolean;
}) {
  const currentUser = useAuthStore((s) => s.user);
  const { allowed: canPin } = usePermission("shipment_comment", Operation.Pin);
  const { allowed: canUnpin } = usePermission("shipment_comment", Operation.Unpin);
  const { allowed: canResolve } = usePermission("shipment_comment", Operation.Resolve);
  const { allowed: canManage } = usePermission("shipment_comment", Operation.Manage);

  if (comment.deletedAt != null) {
    return <CommentTombstone comment={comment} />;
  }

  const isOwner = currentUser?.id === comment.userId;
  const isReply = !!comment.parentCommentId;
  const isResolved = comment.resolvedAt != null;
  const isPinned = comment.pinnedAt != null;
  const userName = comment.user?.name ?? "Unknown user";
  const relativeTime = formatDistanceToNow(fromUnixTime(comment.createdAt), { addSuffix: true });

  const showType = comment.type !== "Internal";
  const showVisibility = comment.visibility !== "Internal";
  const typeColor = commentTypeColor(comment.type);

  return (
    <div
      data-comment-id={comment.id}
      className={cn(
        "group/comment hover:bg-muted/50 flex items-start gap-3 rounded-lg px-2 py-2.5 transition-colors",
        comment.pending && "opacity-60",
        comment.failed && "bg-red-500/5",
        highlighted && "animate-comment-highlight",
      )}
    >
      <div className="relative shrink-0">
        <ResolvedUserAvatar
          userId={comment.user?.id}
          name={comment.user?.name}
          profilePicUrl={comment.user?.profilePicUrl}
          thumbnailUrl={comment.user?.thumbnailUrl}
          className={cn("size-9", isResolved && "opacity-70")}
          fallbackClassName="text-xs"
        />
        {isOnline && (
          <span className="border-background absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full border-2 bg-emerald-500" />
        )}
        {PRIORITY_INDICATOR[comment.priority] && (
          <span
            className={cn(
              "absolute -top-1 -right-1 flex size-4 items-center justify-center rounded-full border text-[10px] leading-none font-bold",
              PRIORITY_INDICATOR[comment.priority]!.className,
            )}
          >
            {PRIORITY_INDICATOR[comment.priority]!.icon}
          </span>
        )}
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className={cn("text-sm font-semibold", isResolved && "text-muted-foreground")}>
            {userName}
          </span>
          {comment.pending ? (
            <span className="text-2xs text-muted-foreground flex items-center gap-1">
              <LoaderIcon className="size-3 animate-spin" />
              Sending…
            </span>
          ) : (
            <span className="text-2xs text-muted-foreground">{relativeTime}</span>
          )}
          {comment.editedAt != null && (
            <span className="text-2xs text-muted-foreground italic">edited</span>
          )}
          {isPinned && (
            <Tooltip>
              <TooltipTrigger render={<PinIcon className="size-3 shrink-0 text-amber-500" />} />
              <TooltipContent side="top">
                Pinned{comment.pinnedBy?.name ? ` by ${comment.pinnedBy.name}` : ""}
              </TooltipContent>
            </Tooltip>
          )}
          {!isEditing && !comment.pending && !comment.failed && (
            <div className="ml-auto shrink-0 opacity-0 transition-opacity group-hover/comment:opacity-100">
              <CommentActions
                comment={comment}
                canEdit={isOwner || canManage}
                canDelete={isOwner || canManage}
                canPin={!isReply && (isPinned ? canUnpin : canPin)}
                canResolve={canResolve}
                onReply={actions.onReply}
                onEdit={onEdit}
                onDelete={actions.onDelete}
                isDeleting={actions.isDeleting}
                onTogglePin={actions.onTogglePin}
                onToggleResolve={actions.onToggleResolve}
              />
            </div>
          )}
        </div>

        {isEditing ? (
          <CommentEditForm
            comment={comment}
            onSave={actions.onSaveEdit}
            onCancel={onCancelEdit}
            isSubmitting={actions.isUpdating}
          />
        ) : (
          <div className={cn(isResolved && "opacity-70")}>
            <CommentBody comment={comment} />
            <CommentAttachments attachments={comment.attachments ?? []} />
          </div>
        )}

        {comment.failed && (
          <div className="mt-1.5 flex items-center gap-2">
            <span className="text-2xs font-medium text-red-500">Failed to send</span>
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="text-2xs h-5 gap-1 px-1.5"
              onClick={actions.onRetry}
            >
              <RotateCcwIcon className="size-2.5" />
              Retry
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="xs"
              className="text-2xs text-muted-foreground h-5 gap-1 px-1.5"
              onClick={actions.onDiscard}
            >
              <Trash2Icon className="size-2.5" />
              Discard
            </Button>
          </div>
        )}

        {isResolved && !isEditing && (
          <div className="text-2xs text-muted-foreground mt-1.5 flex items-center gap-1.5">
            <CheckCircle2Icon className="size-3 text-emerald-500" />
            Resolved
            {comment.resolvedBy?.name ? ` by ${comment.resolvedBy.name}` : ""}
            {comment.resolvedAt != null &&
              ` · ${formatDistanceToNow(fromUnixTime(comment.resolvedAt), { addSuffix: true })}`}
          </div>
        )}

        {comment.requiresAcknowledgment && !isEditing && !comment.failed && (
          <AcknowledgeRow
            comment={comment}
            onAcknowledge={actions.onAcknowledge}
            isAcknowledging={actions.isAcknowledging}
          />
        )}

        {(showType || showVisibility) && !isEditing && (
          <div className="mt-1.5 flex items-center gap-2">
            {showType && (
              <span className="text-2xs text-muted-foreground flex items-center gap-1">
                <span
                  className="inline-block size-1.5 shrink-0 rounded-full"
                  style={{ backgroundColor: typeColor }}
                />
                {commentTypeLabel(comment.type)}
              </span>
            )}
            {showVisibility && (
              <span className="text-2xs text-muted-foreground flex items-center gap-1">
                <EyeIcon className="size-3" />
                {commentVisibilityChoices.find((c) => c.value === comment.visibility)?.label ??
                  comment.visibility}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function CommentTombstone({ comment }: { comment: LocalShipmentComment }) {
  return (
    <div data-comment-id={comment.id} className="flex items-center gap-3 rounded-lg px-2 py-2.5">
      <div className="bg-muted flex size-9 shrink-0 items-center justify-center rounded-full">
        <CircleSlashIcon className="text-muted-foreground size-4" />
      </div>
      <p className="text-muted-foreground text-sm italic">This comment was deleted</p>
    </div>
  );
}

function CommentActions({
  comment,
  canEdit,
  canDelete,
  canPin,
  canResolve,
  onReply,
  onEdit,
  onDelete,
  isDeleting,
  onTogglePin,
  onToggleResolve,
}: {
  comment: LocalShipmentComment;
  canEdit: boolean;
  canDelete: boolean;
  canPin: boolean;
  canResolve: boolean;
  onReply?: () => void;
  onEdit: () => void;
  onDelete: () => void;
  isDeleting: boolean;
  onTogglePin?: (pinned: boolean) => void;
  onToggleResolve: (resolved: boolean) => void;
}) {
  const isPinned = comment.pinnedAt != null;
  const isResolved = comment.resolvedAt != null;
  const hasReplies = (comment.replyCount ?? 0) > 0;
  const hasAnyAction = !!onReply || (canPin && !!onTogglePin) || canResolve || canEdit || canDelete;

  if (!hasAnyAction) return null;

  return (
    <AlertDialog>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button variant="ghost" size="xs" className="size-6">
              <EllipsisVerticalIcon className="size-3.5" />
            </Button>
          }
        />
        <DropdownMenuContent align="end">
          {onReply && (
            <DropdownMenuItem
              startContent={<ReplyIcon className="mr-2 size-3.5" />}
              title="Reply"
              onClick={onReply}
            />
          )}
          {canPin && onTogglePin && (
            <DropdownMenuItem
              startContent={
                isPinned ? (
                  <PinOffIcon className="mr-2 size-3.5" />
                ) : (
                  <PinIcon className="mr-2 size-3.5" />
                )
              }
              title={isPinned ? "Unpin" : "Pin"}
              onClick={() => onTogglePin(!isPinned)}
            />
          )}
          {canResolve && (
            <DropdownMenuItem
              startContent={
                isResolved ? (
                  <UndoIcon className="mr-2 size-3.5" />
                ) : (
                  <CheckCircle2Icon className="mr-2 size-3.5" />
                )
              }
              title={isResolved ? "Reopen" : "Resolve"}
              onClick={() => onToggleResolve(!isResolved)}
            />
          )}
          {canEdit && (
            <DropdownMenuItem
              startContent={<PencilIcon className="mr-2 size-3.5" />}
              title="Edit"
              onClick={onEdit}
            />
          )}
          {canDelete && (
            <AlertDialogTrigger
              render={
                <DropdownMenuItem
                  color="danger"
                  disabled={isDeleting}
                  title="Delete"
                  startContent={<TrashIcon className="mr-2 size-3.5" />}
                />
              }
            />
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete comment?</AlertDialogTitle>
          <AlertDialogDescription>
            {hasReplies
              ? "This comment has replies, so it will be replaced with a deleted-comment placeholder. Its content and attachments are removed permanently."
              : "This action cannot be undone. This will permanently delete this comment and its attachments."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={onDelete}>Delete</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function CommentEditForm({
  comment,
  onSave,
  onCancel,
  isSubmitting,
}: {
  comment: LocalShipmentComment;
  onSave: (input: ShipmentCommentUpdateInput) => void;
  onCancel: () => void;
  isSubmitting: boolean;
}) {
  const editorRef = useRef<CommentEditorHandle>(null);
  const [commentType, setCommentType] = useState<CommentType>(comment.type);
  const [visibility, setVisibility] = useState<CommentVisibility>(comment.visibility);
  const [priority, setPriority] = useState<CommentPriority>(comment.priority);

  useEffect(() => {
    editorRef.current?.focus();
  }, []);

  const handleSave = () => {
    const editor = editorRef.current;
    if (!editor || editor.isEmpty() || isSubmitting) return;
    const text = editor.getText().trim();
    if (!text) return;

    onSave({
      id: comment.id,
      comment: text,
      body: editor.getBody() ?? undefined,
      mentionedUserIds: editor.getMentionIds(),
      attachmentDocumentIds: (comment.attachments ?? []).map((attachment) => attachment.documentId),
      type: commentType,
      visibility,
      priority,
      requiresAcknowledgment: comment.requiresAcknowledgment,
      version: comment.version,
    });
  };

  const toolbar: ReactNode = (
    <TooltipProvider>
      {!comment.parentCommentId && (
        <>
          <CommentOptionPill
            label="Type"
            icon={<TagIcon className="size-3" />}
            value={commentType}
            options={commentTypeChoices}
            onChange={setCommentType}
          />
          <CommentOptionPill
            label="Visibility"
            icon={<EyeIcon className="size-3" />}
            value={visibility}
            options={commentVisibilityChoices}
            onChange={setVisibility}
          />
          <CommentOptionPill
            label="Priority"
            icon={<FlagIcon className="size-3" />}
            value={priority}
            options={commentPriorityChoices}
            onChange={setPriority}
          />
        </>
      )}
      <div className="ml-auto flex items-center gap-1">
        <Button
          variant="ghost"
          size="xs"
          className="text-muted-foreground hover:text-foreground size-6"
          onClick={onCancel}
          disabled={isSubmitting}
          aria-label="Cancel edit"
        >
          <XIcon className="size-3.5" />
        </Button>
        <Button
          size="xs"
          className="size-6"
          onClick={handleSave}
          disabled={isSubmitting}
          aria-label="Save comment"
        >
          {isSubmitting ? (
            <LoaderIcon className="size-3.5 animate-spin" />
          ) : (
            <CheckIcon className="size-3.5" />
          )}
        </Button>
      </div>
    </TooltipProvider>
  );

  return (
    <div className="mt-2">
      <CommentEditor
        ref={editorRef}
        initialBody={comment.body ?? null}
        initialComment={comment.comment}
        compact
        onSubmit={handleSave}
        onEscape={onCancel}
        fetchMentionCandidates={fetchMentionCandidates}
        fetchEntityRefCandidates={fetchAllEntityRefCandidates}
        toolbar={toolbar}
      />
    </div>
  );
}
