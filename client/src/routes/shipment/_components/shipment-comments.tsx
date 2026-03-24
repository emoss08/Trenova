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
} from "@/components/ui/alert-dialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useShipmentComments } from "@/hooks/use-shipment-comments";
import { api } from "@/lib/api";
import {
  commentPriorityChoices,
  commentTypeChoices,
  commentVisibilityChoices,
} from "@/lib/choices";
import { cn } from "@/lib/utils";
import { userInitials } from "@/routes/admin/audit-logs/_components/audit-log-formatters";
import { useAuthStore } from "@/stores/auth-store";
import type { GenericSelectOption } from "@/types/fields";
import type {
  CommentPriority,
  CommentType,
  CommentVisibility,
  ShipmentComment,
} from "@/types/shipment-comment";
import { formatDistanceToNow, fromUnixTime } from "date-fns";
import {
  ArrowDownIcon,
  CheckIcon,
  EllipsisVerticalIcon,
  EyeIcon,
  FlagIcon,
  LoaderIcon,
  MessageSquareIcon,
  PencilIcon,
  SendIcon,
  TagIcon,
  TrashIcon,
  XIcon,
} from "lucide-react";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
  type ReactNode,
} from "react";

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

function getChoiceLabel<T extends string>(
  choices: ReadonlyArray<GenericSelectOption<T>>,
  value: T,
): string {
  return choices.find((c) => c.value === value)?.label ?? value;
}

export default function ShipmentCommentsTab({ shipmentId }: { shipmentId: string }) {
  const {
    comments,
    isLoading,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    createComment,
    isCreating,
    updateComment,
    isUpdating,
    deleteComment,
    isDeleting,
  } = useShipmentComments(shipmentId);

  const [editingCommentId, setEditingCommentId] = useState<string | null>(null);
  const [showScrollButton, setShowScrollButton] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const observerRef = useRef<HTMLDivElement>(null);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const isAtBottomRef = useRef(true);
  const hasInitialScrolled = useRef(false);

  useEffect(() => {
    const viewport = scrollAreaRef.current?.querySelector(
      '[data-slot="scroll-area-viewport"]',
    ) as HTMLElement | null;
    if (!viewport) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = viewport;
      const atBottom = scrollHeight - scrollTop - clientHeight < 50;
      isAtBottomRef.current = atBottom;
      setShowScrollButton(!atBottom);
    };

    viewport.addEventListener("scroll", handleScroll, { passive: true });
    return () => viewport.removeEventListener("scroll", handleScroll);
  }, [isLoading]);

  const scrollToBottom = useCallback(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  useEffect(() => {
    const target = observerRef.current;
    if (!target) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    observer.observe(target);
    return () => observer.unobserve(target);
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  useEffect(() => {
    if (isLoading || comments.length === 0) return;

    if (!hasInitialScrolled.current) {
      hasInitialScrolled.current = true;
      requestAnimationFrame(() => {
        bottomRef.current?.scrollIntoView({ behavior: "instant" });
      });
      return;
    }

    if (isAtBottomRef.current) {
      requestAnimationFrame(() => {
        bottomRef.current?.scrollIntoView({ behavior: "smooth" });
      });
    }
  }, [isLoading, comments.length]);

  if (isLoading) return <LoadingSkeleton />;

  return (
    <div className="flex h-full flex-col">
      <div className="relative min-h-0 flex-1">
        <ScrollArea ref={scrollAreaRef} className="h-full">
          {comments.length === 0 ? (
            <EmptyState />
          ) : (
            <div className="flex flex-col gap-3 px-4">
              <div ref={observerRef} className="h-px" />
              {isFetchingNextPage && (
                <div className="flex items-center justify-center py-2">
                  <span className="text-2xs text-muted-foreground">Loading older comments...</span>
                </div>
              )}
              {comments.map((comment) => (
                <CommentItem
                  key={comment.id}
                  comment={comment}
                  isEditing={editingCommentId === comment.id}
                  onEdit={() => setEditingCommentId(comment.id)}
                  onCancelEdit={() => setEditingCommentId(null)}
                  onSaveEdit={(text, mentionedUserIds, type, visibility, priority) => {
                    updateComment(
                      {
                        commentId: comment.id,
                        id: comment.id,
                        comment: text,
                        mentionedUserIds,
                        type,
                        visibility,
                        priority,
                        version: comment.version,
                      },
                      { onSuccess: () => setEditingCommentId(null) },
                    );
                  }}
                  isUpdating={isUpdating}
                  onDelete={() => deleteComment(comment.id)}
                  isDeleting={isDeleting}
                />
              ))}
              <div ref={bottomRef} />
            </div>
          )}
        </ScrollArea>
        {showScrollButton && (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="outline"
                  size="xs"
                  className="absolute right-4 bottom-2 z-10 size-8 cursor-pointer rounded-full shadow-md"
                  onClick={scrollToBottom}
                >
                  <ArrowDownIcon className="size-4" />
                </Button>
              }
            />
            <TooltipContent side="left">Scroll to latest</TooltipContent>
          </Tooltip>
        )}
      </div>
      <div className="relative shrink-0 border-t border-border px-4 pt-3 pb-4">
        {editingCommentId && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/80 backdrop-blur-[1px]">
            <span className="text-xs text-muted-foreground">Finish editing to add a new comment</span>
          </div>
        )}
        <CommentComposer onSubmit={createComment} isSubmitting={isCreating} />
      </div>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="rounded-lg border border-border bg-card p-3">
          <div className="flex items-center gap-3">
            <Skeleton className="size-7 rounded-full" />
            <div className="flex flex-1 flex-col gap-1.5">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-3 w-full" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
      <MessageSquareIcon className="mb-3 size-8 opacity-40" />
      <p className="text-sm font-medium">No comments yet</p>
      <p className="mt-1 max-w-[260px] text-center text-xs">
        Add operational notes, tag team members, and coordinate on this shipment.
      </p>
    </div>
  );
}

function CommentItem({
  comment,
  isEditing,
  onEdit,
  onCancelEdit,
  onSaveEdit,
  isUpdating,
  onDelete,
  isDeleting,
}: {
  comment: ShipmentComment;
  isEditing: boolean;
  onEdit: () => void;
  onCancelEdit: () => void;
  onSaveEdit: (
    text: string,
    mentionedUserIds: string[],
    type: CommentType,
    visibility: CommentVisibility,
    priority: CommentPriority,
  ) => void;
  isUpdating: boolean;
  onDelete: () => void;
  isDeleting: boolean;
}) {
  const currentUser = useAuthStore((s) => s.user);
  const isOwner = currentUser?.id === comment.userId;
  const userName = comment.user?.name ?? "Unknown user";
  const date = fromUnixTime(comment.createdAt);
  const relativeTime = formatDistanceToNow(date, { addSuffix: true });

  const showType = comment.type !== "Internal";
  const showVisibility = comment.visibility !== "Internal";
  const showMeta = showType || showVisibility;
  const typeColor = commentTypeChoices.find((c) => c.value === comment.type)?.color;

  return (
    <div className="group/comment flex items-center gap-3 rounded-lg px-2 py-2.5 transition-colors hover:bg-muted/50">
      <div className="relative shrink-0">
        <Avatar className="size-9">
          <AvatarImage src={comment.user?.profilePicUrl ?? undefined} />
          <AvatarFallback className="text-xs">{userInitials(comment.user?.name)}</AvatarFallback>
        </Avatar>
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
          <span className="text-sm font-semibold">{userName}</span>
          <span className="text-2xs text-muted-foreground">{relativeTime}</span>
          {comment.editedAt != null && (
            <span className="text-2xs text-muted-foreground italic">edited</span>
          )}
          {isOwner && !isEditing && (
            <div className="ml-auto shrink-0 opacity-0 transition-opacity group-hover/comment:opacity-100">
              <CommentActions onEdit={onEdit} onDelete={onDelete} isDeleting={isDeleting} />
            </div>
          )}
        </div>

        {isEditing ? (
          <CommentEditForm
            initialText={comment.comment}
            initialMentions={
              comment.mentionedUsers
                ?.filter((m) => m.mentionedUser?.name)
                .map((m) => ({ id: m.mentionedUserId, name: m.mentionedUser!.name })) ?? []
            }
            initialType={comment.type}
            initialVisibility={comment.visibility}
            initialPriority={comment.priority}
            onSave={onSaveEdit}
            onCancel={onCancelEdit}
            isSubmitting={isUpdating}
          />
        ) : (
          <CommentBody comment={comment} />
        )}

        {showMeta && !isEditing && (
          <div className="mt-1.5 flex items-center gap-2">
            {showType && (
              <span className="flex items-center gap-1 text-2xs text-muted-foreground">
                <span
                  className="inline-block size-1.5 shrink-0 rounded-full"
                  style={{ backgroundColor: typeColor }}
                />
                {getChoiceLabel(commentTypeChoices, comment.type)}
              </span>
            )}
            {showVisibility && (
              <span className="flex items-center gap-1 text-2xs text-muted-foreground">
                <EyeIcon className="size-3" />
                {getChoiceLabel(commentVisibilityChoices, comment.visibility)}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function CommentActions({
  onEdit,
  onDelete,
  isDeleting,
}: {
  onEdit: () => void;
  onDelete: () => void;
  isDeleting: boolean;
}) {
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
          <DropdownMenuItem
            startContent={<PencilIcon className="mr-2 size-3.5" />}
            title="Edit"
            onClick={onEdit}
          />
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
        </DropdownMenuContent>
      </DropdownMenu>

      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete comment?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. This will permanently delete this comment.
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

function CommentBody({ comment }: { comment: ShipmentComment }) {
  const mentionedUsers = comment.mentionedUsers ?? [];
  if (mentionedUsers.length === 0) {
    return <p className="text-sm whitespace-pre-wrap text-foreground">{comment.comment}</p>;
  }

  let segments: ReactNode[] = [comment.comment];
  for (const mention of mentionedUsers) {
    const name = mention.mentionedUser?.name;
    if (!name) continue;
    const pattern = `@${name}`;
    segments = segments.flatMap((seg) => {
      if (typeof seg !== "string") return [seg];
      const parts: ReactNode[] = [];
      let remaining = seg;
      let idx = remaining.indexOf(pattern);
      while (idx !== -1) {
        if (idx > 0) parts.push(remaining.slice(0, idx));
        parts.push(
          <span key={`${mention.id}-${idx}`} className="font-medium text-blue-500">
            {pattern}
          </span>,
        );
        remaining = remaining.slice(idx + pattern.length);
        idx = remaining.indexOf(pattern);
      }
      if (remaining) parts.push(remaining);
      return parts;
    });
  }

  return <p className="text-sm whitespace-pre-wrap text-foreground">{segments}</p>;
}

function CommentOptionPill<T extends string>({
  label,
  icon,
  value,
  options,
  onChange,
}: {
  label: string;
  icon: ReactNode;
  value: T;
  options: ReadonlyArray<GenericSelectOption<T>>;
  onChange: (value: T) => void;
}) {
  const [popoverOpen, setPopoverOpen] = useState(false);
  const [tooltipOpen, setTooltipOpen] = useState(false);
  const selected = options.find((o) => o.value === value);

  return (
    <Popover open={popoverOpen} onOpenChange={setPopoverOpen}>
      <Tooltip open={popoverOpen ? false : tooltipOpen} onOpenChange={setTooltipOpen}>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={
                <Button variant="ghostInvert" size="xs" className="size-6">
                  {icon}
                </Button>
              }
            />
          }
        />
        <TooltipContent side="top">
          {label}: {selected?.label ?? value}
        </TooltipContent>
      </Tooltip>
      <PopoverContent align="start" className="max-h-60 w-44 gap-1 overflow-y-auto p-1">
        <div className="px-2 py-1 text-2xs font-medium text-muted-foreground">{label}</div>
        {options.map((option) => (
          <button
            key={String(option.value)}
            type="button"
            className={cn(
              "flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent",
              option.value === value && "bg-accent",
            )}
            onClick={() => {
              onChange(option.value as T);
              setPopoverOpen(false);
            }}
          >
            {option.color && (
              <span
                className="size-2 shrink-0 rounded-full"
                style={{ backgroundColor: option.color }}
              />
            )}
            <span className="truncate">{option.label}</span>
            {option.value === value && <CheckIcon className="ml-auto size-3 shrink-0" />}
          </button>
        ))}
      </PopoverContent>
    </Popover>
  );
}

type Mention = { id: string; name: string };

interface MentionInputRef {
  focus: () => void;
  clear: () => void;
  getText: () => string;
  getMentionedUserIds: () => string[];
  isEmpty: () => boolean;
}

const MentionInput = forwardRef<
  MentionInputRef,
  {
    initialText?: string;
    initialMentions?: Mention[];
    placeholder?: string;
    toolbar?: ReactNode;
    onKeyDown?: (e: React.KeyboardEvent<HTMLDivElement>) => void;
  }
>(
  (
    {
      initialText,
      initialMentions,
      placeholder = "Add a comment... Use @ to mention",
      toolbar,
      onKeyDown,
    },
    ref,
  ) => {
    const editorRef = useRef<HTMLDivElement>(null);
    const mentionsRef = useRef<Map<string, Mention>>(new Map());
    const [mentionState, setMentionState] = useState<{
      isOpen: boolean;
      query: string;
      anchorNode: Node | null;
      anchorOffset: number;
    }>({ isOpen: false, query: "", anchorNode: null, anchorOffset: 0 });
    const [isEmpty, setIsEmpty] = useState(!initialText);
    const suggestionsRef = useRef<Array<{ value: string; label: string }>>([]);

    useEffect(() => {
      const el = editorRef.current;
      if (!el) return;
      if (initialText && initialMentions?.length) {
        let html = escapeHtml(initialText);
        for (const m of initialMentions) {
          const pattern = `@${m.name}`;
          const pill = makeMentionHtml(m.id, m.name);
          html = html.replaceAll(escapeHtml(pattern), pill);
          mentionsRef.current.set(m.id, m);
        }
        el.innerHTML = html;
      } else if (initialText) {
        el.textContent = initialText;
      }
      setIsEmpty(!initialText);
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const extractText = useCallback((): string => {
      const el = editorRef.current;
      if (!el) return "";
      let text = "";
      for (const node of el.childNodes) {
        if (node.nodeType === Node.TEXT_NODE) {
          text += node.textContent ?? "";
        } else if (node instanceof HTMLElement && node.dataset.mentionId) {
          text += `@${node.dataset.mentionName}`;
        } else if (node instanceof HTMLElement) {
          text += node.textContent ?? "";
        }
      }
      return text;
    }, []);

    const extractMentionIds = useCallback((): string[] => {
      const el = editorRef.current;
      if (!el) return [];
      const ids: string[] = [];
      el.querySelectorAll<HTMLElement>("[data-mention-id]").forEach((span) => {
        const id = span.dataset.mentionId;
        if (id && !ids.includes(id)) ids.push(id);
      });
      return ids;
    }, []);

    useImperativeHandle(ref, () => ({
      focus: () => editorRef.current?.focus(),
      clear: () => {
        if (editorRef.current) {
          editorRef.current.innerHTML = "";
          mentionsRef.current.clear();
          setIsEmpty(true);
        }
      },
      getText: extractText,
      getMentionedUserIds: extractMentionIds,
      isEmpty: () => isEmpty,
    }));

    const dismissMention = useCallback(() => {
      setMentionState({ isOpen: false, query: "", anchorNode: null, anchorOffset: 0 });
    }, []);

    const handleInput = useCallback(() => {
      const el = editorRef.current;
      if (!el) return;
      const hasContent =
        el.textContent?.trim() !== "" || el.querySelector("[data-mention-id]") != null;
      setIsEmpty(!hasContent);

      const sel = window.getSelection();
      if (!sel || sel.rangeCount === 0) {
        dismissMention();
        return;
      }

      const range = sel.getRangeAt(0);
      const node = range.startContainer;
      if (node.nodeType !== Node.TEXT_NODE) {
        dismissMention();
        return;
      }

      const textBefore = (node.textContent ?? "").slice(0, range.startOffset);
      const atIdx = textBefore.lastIndexOf("@");

      if (atIdx === -1) {
        dismissMention();
        return;
      }

      const charBefore = atIdx > 0 ? textBefore[atIdx - 1] : null;
      if (charBefore && charBefore !== " " && charBefore !== "\n") {
        dismissMention();
        return;
      }

      const query = textBefore.slice(atIdx + 1);
      if (query.split(" ").length > 3) {
        dismissMention();
        return;
      }

      setMentionState({
        isOpen: true,
        query,
        anchorNode: node,
        anchorOffset: atIdx,
      });
    }, [dismissMention]);

    const insertMention = useCallback(
      (userId: string, userName: string) => {
        const el = editorRef.current;
        const { anchorNode, anchorOffset } = mentionState;
        if (!el || !anchorNode || anchorNode.nodeType !== Node.TEXT_NODE) return;

        const sel = window.getSelection();
        if (!sel) return;

        const textContent = anchorNode.textContent ?? "";
        const cursorOffset = sel.getRangeAt(0).startOffset;
        const before = textContent.slice(0, anchorOffset);
        const after = textContent.slice(cursorOffset);

        const beforeNode = document.createTextNode(before);
        const pill = createMentionElement(userId, userName);
        const spaceNode = document.createTextNode("\u00A0");
        const afterNode = document.createTextNode(after || "\u00A0");

        const parent = anchorNode.parentNode!;
        parent.replaceChild(afterNode, anchorNode);
        parent.insertBefore(spaceNode, afterNode);
        parent.insertBefore(pill, spaceNode);
        parent.insertBefore(beforeNode, pill);

        mentionsRef.current.set(userId, { id: userId, name: userName });

        const range = document.createRange();
        range.setStart(afterNode, afterNode === spaceNode ? 1 : 0);
        range.collapse(true);
        sel.removeAllRanges();
        sel.addRange(range);

        dismissMention();
        setIsEmpty(false);
      },
      [mentionState, dismissMention],
    );

    const handleKeyDown = useCallback(
      (e: React.KeyboardEvent<HTMLDivElement>) => {
        if (mentionState.isOpen) {
          const suggestions = suggestionsRef.current;
          if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Tab") {
            e.preventDefault();
            return;
          }
          if (e.key === "Enter") {
            if (suggestions.length > 0) {
              e.preventDefault();
              return;
            }
          }
          if (e.key === "Escape") {
            e.preventDefault();
            dismissMention();
            return;
          }
        }

        if (e.key === "Enter" && !e.shiftKey && !e.metaKey && !e.ctrlKey) {
          e.preventDefault();
          return;
        }

        onKeyDown?.(e);
      },
      [mentionState.isOpen, dismissMention, onKeyDown],
    );

    return (
      <div className="relative rounded-md border border-border bg-background transition-[border-color,box-shadow] duration-200 ease-in-out focus-within:border-brand focus-within:ring-4 focus-within:ring-brand/20">
        <div
          ref={editorRef}
          contentEditable
          suppressContentEditableWarning
          role="textbox"
          aria-placeholder={placeholder}
          onInput={handleInput}
          onKeyDown={handleKeyDown}
          data-empty={isEmpty}
          className={cn(
            "min-h-[100px] w-full px-3 py-2 text-sm focus:outline-none",
            "wrap-break-word whitespace-pre-wrap",
            "data-[empty=true]:before:pointer-events-none data-[empty=true]:before:float-left data-[empty=true]:before:h-0 data-[empty=true]:before:text-muted-foreground data-[empty=true]:before:content-[attr(aria-placeholder)]",
          )}
        />
        {toolbar && (
          <div className="flex items-center gap-1 rounded-b-md border-t border-border bg-muted px-2 py-1">
            {toolbar}
          </div>
        )}
        {mentionState.isOpen && (
          <MentionSuggestions
            query={mentionState.query}
            onSelect={insertMention}
            onDismiss={dismissMention}
            suggestionsRef={suggestionsRef}
          />
        )}
      </div>
    );
  },
);

MentionInput.displayName = "MentionInput";

function makeMentionHtml(id: string, name: string): string {
  return `<span contenteditable="false" data-mention-id="${escapeAttr(id)}" data-mention-name="${escapeAttr(name)}" class="mention-pill">@${escapeHtml(name)}</span>`;
}

function createMentionElement(id: string, name: string): HTMLSpanElement {
  const span = document.createElement("span");
  span.contentEditable = "false";
  span.dataset.mentionId = id;
  span.dataset.mentionName = name;
  span.className = "mention-pill";
  span.textContent = `@${name}`;
  return span;
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function escapeAttr(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/"/g, "&quot;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

function CommentEditForm({
  initialText,
  initialMentions,
  initialType,
  initialVisibility,
  initialPriority,
  onSave,
  onCancel,
  isSubmitting,
}: {
  initialText: string;
  initialMentions?: Array<{ id: string; name: string }>;
  initialType: CommentType;
  initialVisibility: CommentVisibility;
  initialPriority: CommentPriority;
  onSave: (
    text: string,
    mentionedUserIds: string[],
    type: CommentType,
    visibility: CommentVisibility,
    priority: CommentPriority,
  ) => void;
  onCancel: () => void;
  isSubmitting: boolean;
}) {
  const inputRef = useRef<MentionInputRef>(null);
  const [commentType, setCommentType] = useState<CommentType>(initialType);
  const [visibility, setVisibility] = useState<CommentVisibility>(initialVisibility);
  const [priority, setPriority] = useState<CommentPriority>(initialPriority);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const handleSave = () => {
    if (!inputRef.current) return;
    const trimmed = inputRef.current.getText().trim();
    if (!trimmed) return;
    onSave(trimmed, inputRef.current.getMentionedUserIds(), commentType, visibility, priority);
  };

  return (
    <div className="mt-2">
      <MentionInput
        ref={inputRef}
        initialText={initialText}
        initialMentions={initialMentions}
        onKeyDown={(e) => {
          if (e.key === "Escape") onCancel();
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            handleSave();
          }
        }}
        toolbar={
          <TooltipProvider>
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
            <div className="ml-auto flex items-center gap-1">
              <Button
                variant="ghost"
                size="xs"
                className="size-6 text-muted-foreground hover:text-foreground"
                onClick={onCancel}
                disabled={isSubmitting}
              >
                <XIcon className="size-3.5" />
              </Button>
              <Button size="xs" className="size-6" onClick={handleSave} disabled={isSubmitting}>
                {isSubmitting ? (
                  <LoaderIcon className="size-3.5 animate-spin" />
                ) : (
                  <CheckIcon className="size-3.5" />
                )}
              </Button>
            </div>
          </TooltipProvider>
        }
      />
    </div>
  );
}

function CommentComposer({
  onSubmit,
  isSubmitting,
}: {
  onSubmit: (data: {
    comment: string;
    mentionedUserIds: string[];
    type: CommentType;
    visibility: CommentVisibility;
    priority: CommentPriority;
  }) => void;
  isSubmitting: boolean;
}) {
  const inputRef = useRef<MentionInputRef>(null);
  const [commentType, setCommentType] = useState<CommentType>("Internal");
  const [visibility, setVisibility] = useState<CommentVisibility>("Internal");
  const [priority, setPriority] = useState<CommentPriority>("Normal");

  const handleSubmit = () => {
    if (!inputRef.current) return;
    const trimmed = inputRef.current.getText().trim();
    if (!trimmed) return;
    onSubmit({
      comment: trimmed,
      mentionedUserIds: inputRef.current.getMentionedUserIds(),
      type: commentType,
      visibility,
      priority,
    });
    inputRef.current.clear();
    setCommentType("Internal");
    setVisibility("Internal");
    setPriority("Normal");
  };

  return (
    <MentionInput
      ref={inputRef}
      onKeyDown={(e) => {
        if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
          e.preventDefault();
          handleSubmit();
        }
      }}
      toolbar={
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
          <div className="ml-auto">
            <Button size="xs" onClick={handleSubmit} disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Spinner variant="ellipsis" className="size-3.5" />
                  Sending...
                </>
              ) : (
                <>
                  <SendIcon className="size-3.5" />
                  Send
                </>
              )}
            </Button>
          </div>
        </>
      }
    />
  );
}

function MentionSuggestions({
  query,
  onSelect,
  onDismiss,
  suggestionsRef,
}: {
  query: string;
  onSelect: (userId: string, userName: string) => void;
  onDismiss: () => void;
  suggestionsRef: React.RefObject<Array<{ value: string; label: string }>>;
}) {
  const [options, setOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    setIsLoading(true);
    api
      .get<{ results: Array<{ id: string; name: string; profilePicUrl?: string }> }>(
        `/users/select-options/?query=${encodeURIComponent(query)}&limit=8`,
        { signal: controller.signal },
      )
      .then((data) => {
        const mapped = (data.results ?? []).map((u) => ({
          value: u.id,
          label: u.name,
        }));
        setOptions(mapped);
        setActiveIndex(0);
      })
      .catch(() => {})
      .finally(() => setIsLoading(false));

    return () => controller.abort();
  }, [query]);

  useEffect(() => {
    suggestionsRef.current = options;
  }, [options, suggestionsRef]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, options.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter" || e.key === "Tab") {
        if (options.length > 0) {
          e.preventDefault();
          const selected = options[activeIndex];
          if (selected) onSelect(selected.value, selected.label);
        }
      } else if (e.key === "Escape") {
        e.preventDefault();
        onDismiss();
      }
    };

    document.addEventListener("keydown", handleKeyDown, true);
    return () => document.removeEventListener("keydown", handleKeyDown, true);
  }, [options, activeIndex, onSelect, onDismiss]);

  if (!isLoading && options.length === 0) {
    if (query.length > 0) {
      return (
        <div className="absolute bottom-full left-0 z-50 mb-1 w-64 rounded-md border border-border bg-popover p-2 text-sm text-muted-foreground shadow-md">
          No users found
        </div>
      );
    }
    return null;
  }

  return (
    <div className="absolute bottom-full left-0 z-50 mb-1 w-64 rounded-md border border-border bg-popover shadow-md">
      {isLoading ? (
        <div className="p-2 text-sm text-muted-foreground">Searching...</div>
      ) : (
        options.map((option, idx) => (
          <button
            key={option.value}
            type="button"
            className={cn(
              "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-accent",
              idx === activeIndex && "bg-accent",
            )}
            onMouseDown={(e) => {
              e.preventDefault();
              onSelect(option.value, option.label);
            }}
          >
            <span className="flex size-5 items-center justify-center rounded-full bg-muted text-2xs font-medium">
              {userInitials(option.label)}
            </span>
            <span className="truncate">{option.label}</span>
          </button>
        ))
      )}
    </div>
  );
}
