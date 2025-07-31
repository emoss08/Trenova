/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { MeasuredContainer } from "@/components/measured-container";
import { UserSchema } from "@/lib/schemas/user-schema";
import { cn } from "@/lib/utils";
import "@/styles/tiptap.css";
import Placeholder from "@tiptap/extension-placeholder";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { forwardRef, useEffect, useImperativeHandle } from "react";
import { CustomMention } from "./mention-extension";
import { MentionFloatingMenu } from "./mention-list";
import { type CommentType } from "./utils";

interface TiptapEditorProps {
  value?: string;
  isReply?: boolean;
  onChange?: (value: string) => void;
  onMentionedUsersChange?: (userIds: string[]) => void;
  onCommentTypeChange?: (type: CommentType | null) => void;
  searchUsers: (query: string) => Promise<UserSchema[]>;
  isInvalid?: boolean;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

export interface TiptapEditorRef {
  focus: () => void;
  getValue: () => string;
}

export const TiptapEditor = forwardRef<TiptapEditorRef, TiptapEditorProps>(
  (
    {
      value = "",
      onChange,
      onMentionedUsersChange,
      searchUsers,
      placeholder,
      disabled = false,
      isInvalid,
      isReply,
      className,
    },
    ref,
  ) => {
    const effectivePlaceholder =
      placeholder ||
      (isReply
        ? "Write a reply... Use @ to mention users"
        : "Add a comment... Use @ to mention users, / for comment types");

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
          placeholder: effectivePlaceholder,
          emptyEditorClass: "is-editor-empty",
        }),
        CustomMention,
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
        const text = editor.getText();
        onChange?.(text);

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
      },
    });

    useImperativeHandle(ref, () => ({
      focus: () => editor?.commands.focus(),
      getValue: () => editor?.getText() || "",
    }));

    useEffect(() => {
      if (editor && value !== editor.getText()) {
        editor.commands.setContent(value);
      }
    }, [value, editor]);

    return (
      <>
        <MeasuredContainer
          as="div"
          name="editor"
          className={cn(
            "block w-full rounded-md border text-sm",
            "overflow-y-auto",
            "shadow-xs",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            "border-muted-foreground/20 bg-muted px-3 py-2",
            "focus-within:border-blue-600 focus-within:ring-4 focus-within:ring-blue-600/20",
            isInvalid &&
              "border-red-500 bg-red-500/20 ring-0 ring-red-500 focus-within:border-red-600 focus-within:ring-4 focus-within:ring-red-400/20",
            className,
          )}
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
      </>
    );
  },
);

TiptapEditor.displayName = "TiptapEditor";
