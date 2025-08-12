/* eslint-disable react/display-name */
/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { MeasuredContainer } from "@/components/measured-container";
import { Icon } from "@/components/ui/icons";
import { UserSchema } from "@/lib/schemas/user-schema";
import { cn } from "@/lib/utils";
import { useCommentEditStore } from "@/stores/comment-edit-store";
import "@/styles/tiptap.css";
import { faXmark } from "@fortawesome/pro-solid-svg-icons";
import Placeholder from "@tiptap/extension-placeholder";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { forwardRef, useEffect, useImperativeHandle } from "react";
import { CustomMention } from "./extensions/mention-extension";
import { MentionFloatingMenu } from "./extensions/mention-list";
import { SlashCommandExtension } from "./extensions/slash-command-extension";
import { SlashCommandMenu } from "./extensions/slash-command-menu";
import { CommentTypeNode } from "./nodes/comment-type-node";
import { type CommentType } from "./utils";

interface TiptapEditorProps {
  value?: string | Record<string, any>;
  onChange?: (value: string) => void;
  onJsonChange?: (json: Record<string, any>) => void;
  onMentionedUsersChange?: (userIds: string[]) => void;
  onCommentTypeChange?: (type: CommentType | null) => void;
  searchUsers: (query: string) => Promise<UserSchema[]>;
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

export const TiptapEditor = forwardRef<TiptapEditorRef, TiptapEditorProps>(
  (
    {
      value = "",
      onChange,
      onJsonChange,
      onMentionedUsersChange,
      onCommentTypeChange,
      hasIncompleteSlashCommand,
      searchUsers,
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
              class: "m-0",
            },
          },
          heading: false,
          codeBlock: false,
          blockquote: false,
          bulletList: false,
          orderedList: false,
          listItem: false,
          horizontalRule: false,
        }),
        Placeholder.configure({
          emptyNodeClass: "is-editor-empty",
          placeholder: effectivePlaceholder,
          includeChildren: false,
        }),
        CustomMention,
        SlashCommandExtension,
        CommentTypeNode,
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

        // Extract mentioned user IDs
        if (onMentionedUsersChange) {
          const mentions: string[] = [];
          editor.state.doc.descendants((node) => {
            if (node.type.name === "mention" && node.attrs.id) {
              mentions.push(node.attrs.id);
            }
          });
          onMentionedUsersChange([...new Set(mentions)]);
        }

        // Check for comment type changes
        if (onCommentTypeChange) {
          let currentCommentType: string | null = null;
          editor.state.doc.descendants((node) => {
            if (node.type.name === "commentType" && node.attrs.type) {
              currentCommentType = node.attrs.type;
              return false; // Stop at first comment type
            }
          });
          onCommentTypeChange(currentCommentType);
        }
      },
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

    // Load JSON content when editing
    useEffect(() => {
      if (editor && isEditMode && editingComment?.metadata?.editorContent) {
        editor.commands.setContent(editingComment.metadata.editorContent);
      }
    }, [editor, isEditMode, editingComment]);

    const handleContainerClick = () => {
      if (editor && !editor.isFocused) {
        editor.commands.focus("end");
      }
    };

    const handleClearEditMode = () => {
      clearEditMode();
      setEditingComment(null);
      if (editor) {
        editor.commands.setContent("");
      }
    };

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
        )}
        <MeasuredContainer
          as="div"
          name="editor"
          className={cn(
            "block w-full rounded-md border text-sm",
            "overflow-y-auto",
            "min-h-[100px]",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            "border-muted-foreground/20 bg-muted px-3 py-2",
            "cursor-text",
            isEditMode && "rounded-bl-none",
            "focus-within:border-blue-600 focus-within:ring-4 focus-within:ring-blue-600/20",
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
        {editor && searchUsers && (
          <MentionFloatingMenu
            editor={editor}
            searchUsers={searchUsers}
            onMentionedUsersChange={onMentionedUsersChange}
          />
        )}
        {editor && (
          <SlashCommandMenu
            editor={editor}
            onCommentTypeChange={onCommentTypeChange}
          />
        )}
      </div>
    );
  },
);
