import { MeasuredContainer } from "@/components/measured-container";
import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import { useCommentEditStore } from "@/stores/comment-edit-store";
import "@/styles/tiptap.scss";
import { faXmark } from "@fortawesome/pro-regular-svg-icons";
import Mention from "@tiptap/extension-mention";
import Placeholder from "@tiptap/extension-placeholder";
import { CharacterCount } from "@tiptap/extensions";
import { EditorContent, useEditor, useEditorState } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { forwardRef, useCallback, useEffect, useImperativeHandle } from "react";
import suggestion from "./extensions/mention/mention-suggestion";
import { type CommentType } from "./utils";
interface TiptapEditorProps {
  value?: string | Record<string, any>;
  onChange?: (value: string) => void;
  onJsonChange?: (json: Record<string, any>) => void;
  onMentionedUsersChange?: (userIds: string[]) => void;
  onCommentTypeChange?: (type: CommentType | null) => void;
  isInvalid?: boolean;
  placeholder?: string;
  hasIncompleteSlashCommand?: boolean;
  disabled?: boolean;
  className?: string;
}

export interface TiptapEditorRef {
  focus: () => void;
  getValue: () => string;
  getJSON: () => Record<string, any>;
}

const MAX_CHARACTERS = 1000;

export const TiptapEditor = forwardRef<TiptapEditorRef, TiptapEditorProps>(
  (
    {
      value = "",
      onChange,
      onJsonChange,
      onMentionedUsersChange,
      // onCommentTypeChange,
      hasIncompleteSlashCommand,
      placeholder,
      disabled = false,
      isInvalid,
      className,
    },
    ref,
  ) => {
    const { isEditMode, clearEditMode, setEditingComment, editingComment } =
      useCommentEditStore();
    const effectivePlaceholder =
      placeholder ||
      "Add a comment... Use @ to mention users, / for comment types";

    const editor = useEditor({
      extensions: [
        StarterKit.configure({
          paragraph: {
            HTMLAttributes: {
              class: cn(
                "prose prose-sm max-w-none focus:outline-none",
                "min-h-[inherit] max-h-[inherit]",
                "overflow-y-auto",
                disabled && "cursor-not-allowed opacity-50",
              ),
            },
          },
        }),
        Placeholder.configure({
          emptyNodeClass: "is-editor-empty",
          placeholder: effectivePlaceholder,
          includeChildren: false,
        }),
        CharacterCount.configure({
          limit: MAX_CHARACTERS,
        }),
        Mention.configure({
          HTMLAttributes: {
            class: "mention",
          },
          suggestion: suggestion(),
        }),
      ],
      content: value,
      editorProps: {
        attributes: {
          class: cn(
            "prose prose-sm max-w-none focus:outline-none",
            "min-h-[inherit] max-h-[inherit]",
            "overflow-y-auto",
            disabled && "cursor-not-allowed opacity-50",
          ),
        },
      },
      editable: !disabled,
      onUpdate: ({ editor }) => {
        onChange?.(editor.getText());
        onJsonChange?.(editor.getJSON());

        if (onMentionedUsersChange) {
          const mentions: string[] = [];

          editor.state.doc.descendants((node) => {
            if (node.type.name === "mention" && node.attrs.id) {
              mentions.push(node.attrs.id);
            }
          });
          onMentionedUsersChange([...new Set(mentions)]);
        }
      },
    });

    const { characterCount, wordsCount } = useEditorState({
      editor,
      selector: (context) => ({
        characterCount: context.editor.storage.characterCount.characters(),
        wordsCount: context.editor.storage.characterCount.words(),
      }),
    });

    useImperativeHandle(ref, () => ({
      focus: () => editor?.commands.focus(),
      getValue: () => editor?.getText() || "",
      getJSON: () => editor?.getJSON() || { type: "doc", content: [] },
    }));

    useEffect(() => {
      if (editor && value !== editor.getText()) {
        editor.commands.setContent(value);
      }
    }, [value, editor]);

    useEffect(() => {
      if (editor && isEditMode && editingComment?.metadata?.editorContent) {
        editor.commands.setContent(editingComment.metadata.editorContent);
      }
    }, [editor, isEditMode, editingComment]);
    const handleContainerClick = useCallback(() => {
      if (!editor.isFocused) {
        editor.commands.focus("end");
      }
    }, [editor]);

    const handleClearEditMode = useCallback(() => {
      clearEditMode();
      setEditingComment(null);
      editor.commands.setContent("");
    }, [clearEditMode, setEditingComment, editor]);

    if (!editor) return null;

    return (
      <div className="relative">
        {isEditMode && (
          <div className="absolute -bottom-[22px] left-0 z-10 bg-blue-500/20 text-blue-400 text-xs font-semibold px-2 py-0.5 border-b border-x border-blue-500/20 rounded-b">
            Edit Mode Enabled
            <button
              onClick={handleClearEditMode}
              className="ml-2 text-blue-400 hover:text-blue-500"
            >
              <Icon icon={faXmark} />
            </button>
          </div>
        )}{" "}
        <MeasuredContainer
          as="div"
          name="editor"
          className={cn(
            "block w-full rounded-md bg-muted px-3 py-2 rounded-tr-none border border-muted-foreground/20 text-sm overflow-y-auto min-h-[100px]",
            "transition-[border-color,box-shadow] duration-200 ease-in-out cursor-text",
            isEditMode && "rounded-bl-none",
            "focus-within:border-foreground focus-within:ring-4 focus-within:ring-foreground/20",
            isInvalid &&
              "border-red-500 bg-red-500/20 ring-0 ring-red-500 focus-within:border-red-600 focus-within:ring-4 focus-within:ring-red-400/20",
            hasIncompleteSlashCommand &&
              "border-orange-500 bg-orange-500/20 ring-0 ring-orange-500 focus-within:border-orange-600 focus-within:ring-4 focus-within:ring-orange-400/20",
            className,
          )}
          onClick={handleContainerClick}
        >
          <EditorContent
            editor={editor}
            className={cn("minimal-tiptap-editor")}
          />
        </MeasuredContainer>
        <div className="absolute bottom-1 right-2 text-2xs text-muted-foreground">
          {characterCount} / {MAX_CHARACTERS} characters - {wordsCount} words
        </div>
      </div>
    );
  },
);
