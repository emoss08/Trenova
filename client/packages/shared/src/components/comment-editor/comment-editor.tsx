import { Extension, type JSONContent } from "@tiptap/core";
import Mention from "@tiptap/extension-mention";
import Placeholder from "@tiptap/extension-placeholder";
import { EditorContent, useEditor, type Editor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { PluginKey } from "@tiptap/pm/state";
import {
  BoldIcon,
  ItalicIcon,
  LinkIcon,
  ListIcon,
  ListOrderedIcon,
  UnderlineIcon,
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
import { Button } from "../ui/button";
import { Popover, PopoverContent } from "../ui/popover";
import { cn } from "../../lib/utils";
import {
  SuggestionList,
  createSuggestionRender,
  type EditorSuggestionItem,
  type SuggestionController,
  type SuggestionFetcher,
  type SuggestionListHandle,
  type SuggestionSessionState,
} from "./suggestion-popup";

export const COMMENT_EDITOR_MAX_LENGTH = 5000;

export interface CommentEditorHandle {
  focus: () => void;
  clear: () => void;
  isEmpty: () => boolean;
  getBody: () => Record<string, unknown> | null;
  getText: () => string;
  getMentionIds: () => string[];
  setContent: (body: Record<string, unknown> | null, fallbackText: string) => void;
}

export interface CommentEditorProps {
  initialBody?: Record<string, unknown> | null;
  initialComment?: string;
  placeholder?: string;
  autoFocus?: boolean;
  compact?: boolean;
  disabled?: boolean;
  onSubmit?: () => void;
  onEscape?: () => void;
  onTypingChange?: () => void;
  onBlur?: () => void;
  onFiles?: (files: File[]) => void;
  fetchMentionCandidates: SuggestionFetcher;
  fetchEntityRefCandidates?: SuggestionFetcher;
  toolbar?: ReactNode;
}

const mentionPluginKey = new PluginKey("commentMentionSuggestion");
const entityRefPluginKey = new PluginKey("commentEntityRefSuggestion");

const EntityRefMention = Mention.extend({
  name: "entityRef",

  addAttributes() {
    return {
      entityType: {
        default: "",
        parseHTML: (element) => element.getAttribute("data-entity-type") ?? "",
        renderHTML: (attributes) => ({ "data-entity-type": attributes.entityType }),
      },
      entityId: {
        default: "",
        parseHTML: (element) => element.getAttribute("data-entity-id") ?? "",
        renderHTML: (attributes) => ({ "data-entity-id": attributes.entityId }),
      },
      label: {
        default: "",
        parseHTML: (element) => element.getAttribute("data-label") ?? "",
        renderHTML: (attributes) => ({ "data-label": attributes.label }),
      },
    };
  },
});

function plainTextToDoc(text: string): JSONContent {
  const paragraphs = text.split("\n").map((line) => ({
    type: "paragraph",
    content: line === "" ? [] : [{ type: "text", text: line }],
  }));
  return { type: "doc", content: paragraphs.length > 0 ? paragraphs : [{ type: "paragraph" }] };
}

function collectMentionIds(doc: JSONContent | null): string[] {
  if (!doc) return [];
  const ids: string[] = [];
  const seen = new Set<string>();

  const walk = (node: JSONContent) => {
    if (node.type === "mention") {
      const id = typeof node.attrs?.id === "string" ? node.attrs.id : "";
      if (id && !seen.has(id)) {
        seen.add(id);
        ids.push(id);
      }
      return;
    }
    node.content?.forEach(walk);
  };
  walk(doc);

  return ids;
}

interface ToolbarButtonProps {
  icon: ReactNode;
  label: string;
  isActive: boolean;
  onClick: () => void;
  disabled?: boolean;
}

function ToolbarButton({ icon, label, isActive, onClick, disabled }: ToolbarButtonProps) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="xs"
      aria-label={label}
      aria-pressed={isActive}
      disabled={disabled}
      className={cn(
        "size-6 text-muted-foreground hover:text-foreground",
        isActive && "bg-accent text-foreground",
      )}
      onMouseDown={(event) => {
        event.preventDefault();
        onClick();
      }}
    >
      {icon}
    </Button>
  );
}

function LinkControl({ editor, disabled }: { editor: Editor; disabled?: boolean }) {
  const [isOpen, setIsOpen] = useState(false);
  const [href, setHref] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isOpen) inputRef.current?.focus();
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) return;
    const onOutsideMouseDown = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (target && containerRef.current?.contains(target)) return;
      setIsOpen(false);
      setHref("");
    };
    document.addEventListener("mousedown", onOutsideMouseDown, true);
    return () => document.removeEventListener("mousedown", onOutsideMouseDown, true);
  }, [isOpen]);

  const isActive = editor.isActive("link");

  const apply = useCallback(() => {
    const trimmed = href.trim();
    setIsOpen(false);
    setHref("");
    if (!trimmed) return;
    const normalized = /^https?:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`;
    editor.chain().focus().extendMarkRange("link").setLink({ href: normalized }).run();
  }, [editor, href]);

  const toggle = useCallback(() => {
    if (isActive) {
      editor.chain().focus().extendMarkRange("link").unsetLink().run();
      return;
    }
    setIsOpen((open) => !open);
  }, [editor, isActive]);

  return (
    <div ref={containerRef} className="relative flex items-center">
      <ToolbarButton
        icon={<LinkIcon className="size-3.5" />}
        label={isActive ? "Remove link" : "Add link"}
        isActive={isActive || isOpen}
        onClick={toggle}
        disabled={disabled}
      />
      {isOpen && (
        <div className="absolute bottom-full left-0 z-50 mb-1.5 flex items-center gap-1 rounded-md border border-border bg-popover p-1 shadow-md">
          <input
            ref={inputRef}
            value={href}
            onChange={(event) => setHref(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                event.stopPropagation();
                apply();
              }
              if (event.key === "Escape") {
                event.stopPropagation();
                setIsOpen(false);
                setHref("");
              }
            }}
            placeholder="https://example.com"
            className="h-6 w-48 rounded border-none bg-transparent px-1.5 text-xs focus:outline-none"
          />
          <Button
            type="button"
            size="xs"
            className="h-6 px-2 text-xs"
            onMouseDown={(event) => {
              event.preventDefault();
              apply();
            }}
          >
            Apply
          </Button>
        </div>
      )}
    </div>
  );
}

export const CommentEditor = forwardRef<CommentEditorHandle, CommentEditorProps>(
  (
    {
      initialBody,
      initialComment,
      placeholder = "Add a comment… @ to mention, # to reference",
      autoFocus = false,
      compact = false,
      disabled = false,
      onSubmit,
      onEscape,
      onTypingChange,
      onBlur,
      onFiles,
      fetchMentionCandidates,
      fetchEntityRefCandidates,
      toolbar,
    },
    ref,
  ) => {
    const submitRef = useRef(onSubmit);
    submitRef.current = onSubmit;
    const escapeRef = useRef(onEscape);
    escapeRef.current = onEscape;
    const typingRef = useRef(onTypingChange);
    typingRef.current = onTypingChange;
    const filesRef = useRef(onFiles);
    filesRef.current = onFiles;
    const fetchMentionsRef = useRef(fetchMentionCandidates);
    fetchMentionsRef.current = fetchMentionCandidates;
    const fetchEntityRefsRef = useRef(fetchEntityRefCandidates);
    fetchEntityRefsRef.current = fetchEntityRefCandidates;

    const [characterCount, setCharacterCount] = useState(0);
    const [suggestionSession, setSuggestionSession] = useState<SuggestionSessionState | null>(null);
    const suggestionListRef = useRef<SuggestionListHandle>(null);
    const suggestionSessionRef = useRef(suggestionSession);
    suggestionSessionRef.current = suggestionSession;
    const suggestionControllerRef = useRef<SuggestionController>({
      open: () => {},
      close: () => {},
      handleKeyDown: () => false,
    });
    suggestionControllerRef.current.open = setSuggestionSession;
    suggestionControllerRef.current.close = () => setSuggestionSession(null);
    suggestionControllerRef.current.handleKeyDown = (event) =>
      suggestionListRef.current?.onKeyDown(event) ?? false;

    const getSuggestionAnchor = useCallback(() => {
      const session = suggestionSessionRef.current;
      if (!session) return null;
      return {
        getBoundingClientRect: () => session.getAnchorRect() ?? new DOMRect(),
      };
    }, []);

    const editor = useEditor({
      extensions: [
        StarterKit.configure({
          heading: false,
          blockquote: false,
          code: false,
          codeBlock: false,
          horizontalRule: false,
          link: {
            openOnClick: false,
            autolink: true,
            defaultProtocol: "https",
            protocols: ["http", "https"],
          },
        }),
        Placeholder.configure({ placeholder }),
        Extension.create({
          name: "commentEditorKeymap",
          addKeyboardShortcuts() {
            return {
              "Mod-Enter": () => {
                submitRef.current?.();
                return true;
              },
              Escape: () => {
                if (escapeRef.current) {
                  escapeRef.current();
                  return true;
                }
                return false;
              },
            };
          },
        }),
        Mention.configure({
          HTMLAttributes: {
            class: "rounded bg-blue-500/10 px-0.5 font-medium text-blue-500",
          },
          deleteTriggerWithBackspace: true,
          suggestion: {
            char: "@",
            pluginKey: mentionPluginKey,
            items: () => [],
            command: ({ editor: commandEditor, range, props }) => {
              commandEditor
                .chain()
                .focus()
                .insertContentAt(range, [
                  { type: "mention", attrs: { id: props.id, label: props.label } },
                  { type: "text", text: " " },
                ])
                .run();
            },
            render: createSuggestionRender("@", suggestionControllerRef),
          },
        }),
        EntityRefMention.configure({
          HTMLAttributes: {
            class: "rounded bg-violet-500/10 px-0.5 font-medium text-violet-500",
          },
          deleteTriggerWithBackspace: true,
          suggestion: {
            char: "#",
            pluginKey: entityRefPluginKey,
            items: () => [],
            command: ({ editor: commandEditor, range, props }) => {
              const item = props as unknown as EditorSuggestionItem;
              commandEditor
                .chain()
                .focus()
                .insertContentAt(range, [
                  {
                    type: "entityRef",
                    attrs: {
                      entityType: item.kind ?? "",
                      entityId: item.id,
                      label: item.label,
                    },
                  },
                  { type: "text", text: " " },
                ])
                .run();
            },
            render: createSuggestionRender("#", suggestionControllerRef),
          },
        }),
      ],
      content: initialBody ?? (initialComment ? plainTextToDoc(initialComment) : undefined),
      autofocus: autoFocus ? "end" : false,
      editable: !disabled,
      editorProps: {
        attributes: {
          role: "textbox",
          "aria-multiline": "true",
          "aria-label": "Comment editor",
          class: cn(
            "w-full px-3 py-2 text-sm focus:outline-none wrap-break-word",
            compact ? "min-h-[44px]" : "min-h-[100px]",
          ),
        },
        handlePaste: (_view, event) => {
          const files = Array.from(event.clipboardData?.files ?? []);
          if (files.length > 0 && filesRef.current) {
            event.preventDefault();
            filesRef.current(files);
            return true;
          }
          return false;
        },
        handleDrop: (_view, event) => {
          const files = Array.from(event.dataTransfer?.files ?? []);
          if (files.length > 0 && filesRef.current) {
            event.preventDefault();
            filesRef.current(files);
            return true;
          }
          return false;
        },
      },
      onUpdate: ({ editor: updatedEditor }) => {
        setCharacterCount(updatedEditor.getText().length);
        typingRef.current?.();
      },
      onBlur: () => {
        setSuggestionSession(null);
        onBlur?.();
      },
    });

    useEffect(() => {
      editor?.setEditable(!disabled);
    }, [editor, disabled]);

    useImperativeHandle(
      ref,
      () => ({
        focus: () => editor?.commands.focus("end"),
        clear: () => {
          editor?.commands.clearContent(true);
          setCharacterCount(0);
        },
        isEmpty: () => {
          if (!editor) return true;
          return editor.isEmpty || editor.getText().trim() === "";
        },
        getBody: () => (editor ? (editor.getJSON() as Record<string, unknown>) : null),
        getText: () => editor?.getText() ?? "",
        getMentionIds: () => collectMentionIds(editor?.getJSON() ?? null),
        setContent: (body, fallbackText) => {
          if (!editor) return;
          editor.commands.setContent(body ?? plainTextToDoc(fallbackText));
          setCharacterCount(editor.getText().length);
        },
      }),
      [editor],
    );

    const overLimit = characterCount > COMMENT_EDITOR_MAX_LENGTH;
    const nearLimit = characterCount > COMMENT_EDITOR_MAX_LENGTH - 500;

    return (
      <div
        className={cn(
          "relative rounded-md border border-border bg-background transition-[border-color,box-shadow] duration-200 ease-in-out focus-within:border-brand focus-within:ring-4 focus-within:ring-brand/20",
          disabled && "opacity-60",
        )}
      >
        <EditorContent editor={editor} />
        <Popover
          open={suggestionSession != null}
          onOpenChange={(open) => {
            if (!open) setSuggestionSession(null);
          }}
          modal={false}
        >
          <PopoverContent
            side="bottom"
            align="start"
            sideOffset={6}
            anchor={getSuggestionAnchor}
            initialFocus={false}
            finalFocus={false}
            className="flex max-h-[min(18rem,var(--available-height))] w-64 flex-col overflow-hidden p-0"
          >
            {suggestionSession && (
              <SuggestionList
                ref={suggestionListRef}
                prefix={suggestionSession.prefix}
                query={suggestionSession.query}
                fetchPage={
                  suggestionSession.prefix === "@"
                    ? (query, page) => fetchMentionsRef.current(query, page)
                    : (query, page) =>
                        fetchEntityRefsRef.current?.(query, page) ??
                        Promise.resolve({ items: [], hasMore: false })
                }
                command={suggestionSession.command}
              />
            )}
          </PopoverContent>
        </Popover>
        <div className="flex items-center gap-0.5 rounded-b-md border-t border-border bg-muted px-2 py-1">
          {editor && (
            <>
              <ToolbarButton
                icon={<BoldIcon className="size-3.5" />}
                label="Bold"
                isActive={editor.isActive("bold")}
                onClick={() => editor.chain().focus().toggleBold().run()}
                disabled={disabled}
              />
              <ToolbarButton
                icon={<ItalicIcon className="size-3.5" />}
                label="Italic"
                isActive={editor.isActive("italic")}
                onClick={() => editor.chain().focus().toggleItalic().run()}
                disabled={disabled}
              />
              <ToolbarButton
                icon={<UnderlineIcon className="size-3.5" />}
                label="Underline"
                isActive={editor.isActive("underline")}
                onClick={() => editor.chain().focus().toggleUnderline().run()}
                disabled={disabled}
              />
              <ToolbarButton
                icon={<ListIcon className="size-3.5" />}
                label="Bullet list"
                isActive={editor.isActive("bulletList")}
                onClick={() => editor.chain().focus().toggleBulletList().run()}
                disabled={disabled}
              />
              <ToolbarButton
                icon={<ListOrderedIcon className="size-3.5" />}
                label="Numbered list"
                isActive={editor.isActive("orderedList")}
                onClick={() => editor.chain().focus().toggleOrderedList().run()}
                disabled={disabled}
              />
              <LinkControl editor={editor} disabled={disabled} />
            </>
          )}
          {nearLimit && (
            <span
              className={cn(
                "ml-1 text-2xs tabular-nums",
                overLimit ? "font-medium text-red-500" : "text-muted-foreground",
              )}
            >
              {characterCount}/{COMMENT_EDITOR_MAX_LENGTH}
            </span>
          )}
          {toolbar && <div className="ml-auto flex items-center gap-1">{toolbar}</div>}
        </div>
      </div>
    );
  },
);

CommentEditor.displayName = "CommentEditor";

export type { EditorSuggestionItem, SuggestionFetcher, SuggestionPage } from "./suggestion-popup";
