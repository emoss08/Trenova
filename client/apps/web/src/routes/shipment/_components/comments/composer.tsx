import {
  CommentEditor,
  type CommentEditorHandle,
} from "@trenova/shared/components/comment-editor/comment-editor";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@trenova/shared/components/ui/tooltip";
import {
  commentPriorityChoices,
  commentTypeChoices,
  commentVisibilityChoices,
} from "@trenova/shared/lib/comment-choices";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import { EyeIcon, FlagIcon, PaperclipIcon, SendIcon, TagIcon } from "lucide-react";
import { useCallback, useMemo, useRef, useState, type DragEvent } from "react";
import { toast } from "sonner";
import {
  fetchAllEntityRefCandidates,
  fetchMentionCandidates,
} from "@/hooks/shipment-comments/use-comment-suggestions";
import { useUploadWithProgress } from "@/hooks/use-upload-with-progress";
import type {
  CommentPriority,
  CommentType,
  CommentVisibility,
  ShipmentCommentCreateInput,
} from "@/types/shipment-comment";
import { ComposerAttachments } from "./composer-attachments";
import { CommentOptionPill } from "./option-pills";

const MAX_ATTACHMENTS = 10;
const MAX_ATTACHMENT_BYTES = 25 * 1024 * 1024;

const ALLOWED_MIME_PREFIXES = ["image/"];
const ALLOWED_MIME_TYPES = new Set([
  "application/pdf",
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.ms-excel",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "text/csv",
  "text/plain",
]);

function isAllowedAttachment(file: File): boolean {
  if (ALLOWED_MIME_PREFIXES.some((prefix) => file.type.startsWith(prefix))) return true;
  return ALLOWED_MIME_TYPES.has(file.type);
}

export interface CommentComposerProps {
  shipmentId: string;
  parentCommentId?: string;
  compact?: boolean;
  autoFocus?: boolean;
  placeholder?: string;
  isSubmitting: boolean;
  onSubmit: (input: ShipmentCommentCreateInput) => void;
  onTyping?: () => void;
  onStopTyping?: () => void;
  onEscape?: () => void;
}

export function CommentComposer({
  shipmentId,
  parentCommentId,
  compact = false,
  autoFocus = false,
  placeholder,
  isSubmitting,
  onSubmit,
  onTyping,
  onStopTyping,
  onEscape,
}: CommentComposerProps) {
  const editorRef = useRef<CommentEditorHandle>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const attachedDocsRef = useRef<Document[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const [commentType, setCommentType] = useState<CommentType>("Internal");
  const [visibility, setVisibility] = useState<CommentVisibility>("Internal");
  const [priority, setPriority] = useState<CommentPriority>("Normal");

  const { uploads, uploadFiles, cancelUpload, retryUpload, removeUpload, clearAll, isUploading } =
    useUploadWithProgress({
      resourceId: shipmentId,
      resourceType: "shipment",
      onSuccess: (result) => {
        attachedDocsRef.current = [...attachedDocsRef.current, result as Document];
      },
    });

  const hasFailedUploads = uploads.some((upload) => upload.status === "error");

  const enqueueFiles = useCallback(
    (files: File[]) => {
      const accepted: File[] = [];
      for (const file of files) {
        if (!isAllowedAttachment(file)) {
          toast.error(`"${file.name}" is not a supported attachment type`);
          continue;
        }
        if (file.size > MAX_ATTACHMENT_BYTES) {
          toast.error(`"${file.name}" exceeds the 25 MB attachment limit`);
          continue;
        }
        accepted.push(file);
      }
      if (accepted.length === 0) return;
      if (uploads.length + accepted.length > MAX_ATTACHMENTS) {
        toast.error(`Comments support up to ${MAX_ATTACHMENTS} attachments`);
        return;
      }
      uploadFiles(accepted);
    },
    [uploads.length, uploadFiles],
  );

  const handleRemoveUpload = useCallback(
    (id: string) => {
      const upload = uploads.find((entry) => entry.id === id);
      if (upload) {
        attachedDocsRef.current = attachedDocsRef.current.filter(
          (doc) => !(doc.originalName === upload.file.name && doc.fileSize === upload.file.size),
        );
      }
      removeUpload(id);
    },
    [uploads, removeUpload],
  );

  const handleSubmit = useCallback(() => {
    const editor = editorRef.current;
    if (!editor || editor.isEmpty() || isSubmitting) return;
    if (isUploading) {
      toast.error("Wait for attachments to finish uploading");
      return;
    }
    if (hasFailedUploads) {
      toast.error("Remove failed attachments before sending");
      return;
    }

    const text = editor.getText().trim();
    if (!text) return;

    onSubmit({
      comment: text,
      body: editor.getBody() ?? undefined,
      parentCommentId,
      mentionedUserIds: editor.getMentionIds(),
      attachmentDocumentIds: attachedDocsRef.current.map((doc) => doc.id),
      type: commentType,
      visibility,
      priority,
    });

    editor.clear();
    clearAll();
    attachedDocsRef.current = [];
    setCommentType("Internal");
    setVisibility("Internal");
    setPriority("Normal");
    onStopTyping?.();
  }, [
    isSubmitting,
    isUploading,
    hasFailedUploads,
    onSubmit,
    parentCommentId,
    commentType,
    visibility,
    priority,
    clearAll,
    onStopTyping,
  ]);

  const handleDragOver = useCallback((event: DragEvent) => {
    if (event.dataTransfer.types.includes("Files")) {
      event.preventDefault();
      setIsDragging(true);
    }
  }, []);

  const handleDrop = useCallback(
    (event: DragEvent) => {
      event.preventDefault();
      setIsDragging(false);
      enqueueFiles(Array.from(event.dataTransfer.files));
    },
    [enqueueFiles],
  );

  const editorToolbar = useMemo(
    () => (
      <TooltipProvider>
        {!parentCommentId && (
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
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                variant="ghostInvert"
                size="xs"
                className="size-6"
                onClick={() => fileInputRef.current?.click()}
              >
                <PaperclipIcon className="size-3" />
              </Button>
            }
          />
          <TooltipContent side="top">Attach files</TooltipContent>
        </Tooltip>
        <Button
          size="xs"
          className={cn(compact && "size-6")}
          onClick={handleSubmit}
          disabled={isSubmitting || isUploading || hasFailedUploads}
        >
          {isSubmitting ? (
            <>
              <Spinner variant="ellipsis" className="size-3.5" />
              {!compact && "Sending..."}
            </>
          ) : (
            <>
              <SendIcon className="size-3.5" />
              {!compact && "Send"}
            </>
          )}
        </Button>
      </TooltipProvider>
    ),
    [
      parentCommentId,
      commentType,
      visibility,
      priority,
      compact,
      handleSubmit,
      isSubmitting,
      isUploading,
      hasFailedUploads,
    ],
  );

  return (
    <div
      className={cn(
        "relative rounded-md transition-shadow",
        isDragging && "ring-2 ring-brand ring-offset-2 ring-offset-background",
      )}
      onDragOver={handleDragOver}
      onDragLeave={() => setIsDragging(false)}
      onDrop={handleDrop}
    >
      <ComposerAttachments
        uploads={uploads}
        onCancel={cancelUpload}
        onRetry={retryUpload}
        onRemove={handleRemoveUpload}
      />
      <CommentEditor
        ref={editorRef}
        compact={compact}
        autoFocus={autoFocus}
        placeholder={
          placeholder ??
          (parentCommentId ? "Reply… @ to mention" : "Add a comment… @ to mention, # to reference")
        }
        disabled={isSubmitting}
        onSubmit={handleSubmit}
        onEscape={onEscape}
        onTypingChange={onTyping}
        onBlur={onStopTyping}
        onFiles={enqueueFiles}
        fetchMentionCandidates={fetchMentionCandidates}
        fetchEntityRefCandidates={fetchAllEntityRefCandidates}
        toolbar={editorToolbar}
      />
      {!parentCommentId && priority === "Urgent" && (
        <p className="mt-1 text-2xs text-muted-foreground">
          Urgent comments request acknowledgment from viewers.
        </p>
      )}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={(event) => {
          enqueueFiles(Array.from(event.target.files ?? []));
          event.target.value = "";
        }}
      />
    </div>
  );
}
