/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { mergeAttributes } from "@tiptap/core";
import Mention from "@tiptap/extension-mention";

export const CustomMention = Mention.configure({
  HTMLAttributes: {
    class: "mention",
  },
  renderHTML({ options, node }) {
    return [
      "span",
      mergeAttributes(options.HTMLAttributes, {
        "data-mention": node.attrs.id,
        "data-label": node.attrs.label,
      }),
      `@${node.attrs.label}`,
    ];
  },
  suggestion: {
    char: "@",
    allowSpaces: false,
    startOfLine: false,
    render: () => {
      return {
        onStart: () => {},
        onUpdate: () => {},
        onKeyDown: () => false,
        onExit: () => {},
      };
    },
  },
});
