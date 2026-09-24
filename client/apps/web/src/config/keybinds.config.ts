export interface Keybind {
  id: string;
  label: string;
  keys: string[];
  description: string;
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
];
