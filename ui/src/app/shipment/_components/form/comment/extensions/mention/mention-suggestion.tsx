import { api } from "@/services/api";
import { computePosition, flip, shift } from "@floating-ui/dom";
import type { Editor } from "@tiptap/core";
import { posToDOMRect, ReactRenderer } from "@tiptap/react";
import type { SuggestionOptions, SuggestionProps } from "@tiptap/suggestion";
import { MentionList, type MentionListRef } from "./mention-list";

const updatePosition = (editor: Editor, element: HTMLElement) => {
  const virtualElement = {
    getBoundingClientRect: () =>
      posToDOMRect(
        editor.view,
        editor.state.selection.from,
        editor.state.selection.to,
      ),
  };

  computePosition(virtualElement, element, {
    placement: "bottom-start",
    strategy: "fixed",
    middleware: [shift(), flip()],
  }).then(({ x, y }) => {
    element.style.width = "max-content";
    element.style.position = "fixed";
    element.style.left = `${x}px`;
    element.style.top = `${y}px`;
    element.style.zIndex = "99999";
  });
};

export default function suggestion(): Omit<SuggestionOptions, "editor"> {
  return {
    items: async ({ query }) => {
      if (!query || query.length < 1) {
        return [];
      }

      try {
        const response = await api.user.searchUsers(query);
        return response.results || [];
      } catch (error) {
        console.error("Failed to search users:", error);
        return [];
      }
    },
    render: () => {
      let component: ReactRenderer<MentionListRef, SuggestionProps> | undefined;

      return {
        onStart: (props) => {
          component = new ReactRenderer(MentionList, {
            props,
            editor: props.editor,
          });

          if (!props.clientRect) {
            return;
          }

          component.element.style.position = "fixed";

          const portalContainer =
            document.querySelector("[data-radix-portal]") ||
            document.querySelector('[data-slot="sheet-portal"]') ||
            document.body;

          portalContainer.appendChild(component.element);

          updatePosition(props.editor, component.element);
        },

        onUpdate(props) {
          if (!component) return;

          component.updateProps(props);

          if (!props.clientRect) {
            return;
          }

          updatePosition(props.editor, component.element);
        },

        onKeyDown(props) {
          if (props.event.key === "Escape") {
            if (component) {
              component.destroy();
            }
            return true;
          }

          return component?.ref?.onKeyDown(props) ?? false;
        },

        onExit() {
          if (component) {
            component.element.remove();
            component.destroy();
          }
        },
      };
    },
  };
}
