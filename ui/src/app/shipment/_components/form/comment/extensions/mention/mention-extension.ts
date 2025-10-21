import { mergeAttributes } from "@tiptap/core";
import Mention from "@tiptap/extension-mention";
import type { SuggestionOptions } from "@tiptap/suggestion";

export const createMentionExtension = (
  suggestionOptions: Omit<SuggestionOptions, "editor">,
) => {
  return Mention.extend({
    atom: true,
  }).configure({
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
    suggestion: suggestionOptions,
  });
};
