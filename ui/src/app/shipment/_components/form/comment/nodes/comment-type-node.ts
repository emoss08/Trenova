/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Node, mergeAttributes } from "@tiptap/core";
import { ReactNodeViewRenderer } from "@tiptap/react";
import { CommentTypeComponent } from "./comment-type-component";

export interface CommentTypeNodeOptions {
  HTMLAttributes: Record<string, any>;
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    commentType: {
      insertCommentType: (type: string) => ReturnType;
    };
  }
}

export const CommentTypeNode = Node.create<CommentTypeNodeOptions>({
  name: "commentType",

  group: "inline",
  inline: true,
  atom: true,
  selectable: true,
  draggable: false,

  addOptions() {
    return {
      HTMLAttributes: {},
    };
  },

  addAttributes() {
    return {
      type: {
        default: null,
        parseHTML: (element) => element.getAttribute("data-comment-type"),
        renderHTML: (attributes) => {
          if (!attributes.type) {
            return {};
          }
          return {
            "data-comment-type": attributes.type,
          };
        },
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: "span[data-comment-type]",
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "span",
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        "data-comment-type": HTMLAttributes["data-comment-type"],
      }),
    ];
  },

  addNodeView() {
    return ReactNodeViewRenderer(CommentTypeComponent);
  },

  addCommands() {
    return {
      insertCommentType:
        (type: string) =>
        ({ commands }) => {
          return commands.insertContent([
            {
              type: this.name,
              attrs: { type },
            },
            {
              type: "text",
              text: " ",
            },
          ]);
        },
    };
  },
});
