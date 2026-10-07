import type { Hotkey } from "@tanstack/react-hotkeys";

export interface Keybind {
  id: string;
  label: string;
  keys: string[];
  description: string;
  /** The binding as `useHotkey` takes it, for a keybind bound from this config. */
  hotkey?: Hotkey;
}

export interface KeybindGroup {
  id: string;
  label: string;
  keybinds: Keybind[];
}

export const keybindGroups: KeybindGroup[] = [
  {
    id: "general",
    label: "General",
    keybinds: [
      {
        id: "command-palette",
        label: "Command palette",
        keys: ["Ctrl", "K"],
        description: "Search records, jump to pages, run commands and ask the assistant",
      },
      {
        id: "toggle-sidebar",
        label: "Toggle sidebar",
        keys: ["Ctrl", "B"],
        description: "Expand or collapse the sidebar navigation",
      },
      {
        id: "user-settings",
        label: "User settings",
        keys: ["Ctrl", "Shift", "S"],
        description: "Open the user settings dialog",
      },
      {
        id: "keyboard-shortcuts",
        label: "Keyboard shortcuts",
        keys: ["Ctrl", "/"],
        description: "Show this keyboard shortcuts dialog",
      },
      {
        id: "assistant",
        label: "Assistant",
        keys: ["Ctrl", "J"],
        description: "Open or close the assistant from any page",
      },
    ],
  },
  {
    id: "approval-box",
    label: "Approval box",
    keybinds: [
      {
        id: "approval-approve",
        label: "Approve",
        keys: ["Ctrl", "Enter"],
        description: "Approve the change the approval box shows; Enter alone never approves",
      },
      {
        id: "approval-tell",
        label: "Tell the agent",
        keys: ["Esc"],
        description: "Open or close the note that turns the change down and tells the agent why",
      },
      {
        id: "approval-later",
        label: "Decide later",
        keys: ["Alt", "L"],
        description: "Put the waiting decisions off and bring back the message box",
      },
    ],
  },
  {
    id: "command-palette",
    label: "Command palette",
    keybinds: [
      {
        id: "palette-scope",
        label: "Change scope",
        keys: ["Tab"],
        description: "Move between all results, one kind of record, pages and commands",
      },
      {
        id: "palette-actions",
        label: "Show actions",
        keys: ["→"],
        description: "List everything you can do with the selected result",
      },
      {
        id: "palette-new-tab",
        label: "Open in new tab",
        keys: ["Ctrl", "Enter"],
        description: "Open the selected result in a new browser tab",
      },
      {
        id: "palette-copy-link",
        label: "Copy link",
        keys: ["Ctrl", "L"],
        description: "Copy a link to the selected result",
      },
      {
        id: "palette-copy-id",
        label: "Copy number",
        keys: ["Alt", "C"],
        description: "Copy the selected record's PRO number, name or file name",
      },
    ],
  },
  {
    id: "comments",
    label: "Comments",
    keybinds: [
      {
        id: "comment-send",
        label: "Send comment",
        keys: ["Ctrl", "Enter"],
        description: "Send the comment you are composing",
      },
      {
        id: "comment-cancel",
        label: "Cancel edit or reply",
        keys: ["Esc"],
        description: "Close the inline comment editor or reply composer",
      },
      {
        id: "comment-bold",
        label: "Bold",
        keys: ["Ctrl", "B"],
        description: "Toggle bold formatting in the comment editor",
      },
      {
        id: "comment-italic",
        label: "Italic",
        keys: ["Ctrl", "I"],
        description: "Toggle italic formatting in the comment editor",
      },
      {
        id: "comment-mention",
        label: "Mention or reference",
        keys: ["@", "#"],
        description:
          "Mention a teammate with @ or reference a shipment, worker, or customer with #",
      },
    ],
  },
  {
    id: "forms",
    label: "Forms & dialogs",
    keybinds: [
      {
        id: "submit-form",
        label: "Submit form",
        keys: ["Ctrl", "Enter"],
        description: "Submit the current form in a modal or panel",
      },
      {
        id: "close-dialog",
        label: "Close dialog",
        keys: ["Esc"],
        description: "Close the current dialog or modal",
      },
    ],
  },
  {
    id: "edit-sheet",
    label: "Editing",
    keybinds: [
      {
        id: "save",
        label: "Save changes",
        keys: ["Ctrl", "S"],
        description: "Save the editor you are in",
        hotkey: "Mod+S",
      },
      {
        id: "close",
        label: "Close editor",
        keys: ["Esc"],
        description: "Close the editor, asking first when there are unsaved changes",
        hotkey: "Escape",
      },
    ],
  },
  {
    id: "ai-control",
    label: "AI control",
    keybinds: [
      {
        id: "tab",
        label: "Go to a tab",
        keys: ["1–9"],
        description: "Open the first nine tabs of AI control in order",
      },
      {
        id: "search",
        label: "Search",
        keys: ["/"],
        description: "Search the list on the current tab",
        hotkey: "/",
      },
      {
        id: "new",
        label: "New",
        keys: ["N"],
        description: "Create a new agent or provider on its tab",
        hotkey: "N",
      },
      {
        id: "edit",
        label: "Edit",
        keys: ["E"],
        description: "Open the selected row's editor",
        hotkey: "E",
      },
    ],
  },
  {
    id: "navigation",
    label: "Record navigation",
    keybinds: [
      {
        id: "prev-record",
        label: "Previous record",
        keys: ["↑"],
        description: "Navigate to the previous record in an edit dialog",
      },
      {
        id: "next-record",
        label: "Next record",
        keys: ["↓"],
        description: "Navigate to the next record in an edit dialog",
      },
    ],
  },
  {
    id: "intake",
    label: "Intake",
    keybinds: [
      {
        id: "intake-next-document",
        label: "Next document",
        keys: ["J"],
        description: "Move to the next document in the open stack",
      },
      {
        id: "intake-previous-document",
        label: "Previous document",
        keys: ["K"],
        description: "Move to the previous document in the open stack",
      },
      {
        id: "intake-file-document",
        label: "File document",
        keys: ["F"],
        description: "File the document you are on, once it has a record",
      },
      {
        id: "intake-turn-document",
        label: "Turn document",
        keys: ["R"],
        description: "Turn every page of the document you are on a quarter to the right",
      },
      {
        id: "intake-save-split",
        label: "Save split",
        keys: ["S"],
        description: "Save how the pages divide into documents",
      },
      {
        id: "intake-page-previous",
        label: "Previous page",
        keys: ["←"],
        description: "Show the page before in a page preview",
      },
      {
        id: "intake-page-next",
        label: "Next page",
        keys: ["→"],
        description: "Show the page after in a page preview",
      },
    ],
  },
  {
    id: "shipments",
    label: "Shipments",
    keybinds: [
      {
        id: "shipments-next-row",
        label: "Next shipment",
        keys: ["J"],
        description: "Move to the next row; the down arrow does the same",
      },
      {
        id: "shipments-previous-row",
        label: "Previous shipment",
        keys: ["K"],
        description: "Move to the previous row; the up arrow does the same",
      },
      {
        id: "shipments-expand",
        label: "Open or close a shipment",
        keys: ["Enter"],
        description: "Expand the shipment under the cursor, or collapse it if it is open",
      },
      {
        id: "shipments-collapse",
        label: "Close",
        keys: ["Esc"],
        description: "Collapse the open shipment, then clear the cursor",
      },
      {
        id: "shipments-select",
        label: "Select",
        keys: ["X"],
        description: "Select or deselect the shipment under the cursor",
      },
      {
        id: "shipments-edit",
        label: "Edit",
        keys: ["E"],
        description: "Edit the shipment under the cursor",
      },
      {
        id: "shipments-copy-pro",
        label: "Copy PRO number",
        keys: ["Alt", "C"],
        description: "Copy the PRO number of the shipment under the cursor",
      },
      {
        id: "shipments-copy-link",
        label: "Copy link",
        keys: ["Ctrl", "L"],
        description: "Copy a link to the shipment under the cursor",
      },
      {
        id: "shipments-search",
        label: "Search",
        keys: ["/"],
        description: "Focus the shipment search and show the quick filters",
      },
      {
        id: "shipments-approve",
        label: "Approve suggestion",
        keys: ["Ctrl", "Enter"],
        description: "Run the suggested action at the top of the brief",
      },
      {
        id: "shipments-later",
        label: "Later",
        keys: ["Alt", "L"],
        description: "Move the current suggestion to the back of the queue",
      },
    ],
  },
];

/**
 * The `useHotkey` binding of a keybind, so the key a shortcut sheet shows and the key that
 * is bound are the same entry. Throws for a keybind that is not bound from config, which is
 * a mistake in the caller rather than something to recover from.
 */
export function hotkeyOf(groupId: string, keybindId: string): Hotkey {
  const hotkey = keybindGroups
    .find((group) => group.id === groupId)
    ?.keybinds.find((keybind) => keybind.id === keybindId)?.hotkey;
  if (!hotkey) {
    throw new Error(`No hotkey is configured for ${groupId}.${keybindId}`);
  }
  return hotkey;
}
