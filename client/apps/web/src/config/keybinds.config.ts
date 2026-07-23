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
        label: "Command Palette",
        keys: ["Ctrl", "K"],
        description: "Open the command palette to search routes and commands",
      },
      {
        id: "toggle-sidebar",
        label: "Toggle Sidebar",
        keys: ["Ctrl", "B"],
        description: "Expand or collapse the sidebar navigation",
      },
      {
        id: "user-settings",
        label: "User Settings",
        keys: ["Ctrl", "Shift", "S"],
        description: "Open the user settings dialog",
      },
      {
        id: "keyboard-shortcuts",
        label: "Keyboard Shortcuts",
        keys: ["Ctrl", "/"],
        description: "Show this keyboard shortcuts dialog",
      },
    ],
  },
  {
    id: "forms",
    label: "Forms & Dialogs",
    keybinds: [
      {
        id: "submit-form",
        label: "Submit Form",
        keys: ["Ctrl", "Enter"],
        description: "Submit the current form in a modal or panel",
      },
      {
        id: "close-dialog",
        label: "Close Dialog",
        keys: ["Esc"],
        description: "Close the current dialog or modal",
      },
    ],
  },
  {
    id: "navigation",
    label: "Record Navigation",
    keybinds: [
      {
        id: "prev-record",
        label: "Previous Record",
        keys: ["↑"],
        description: "Navigate to the previous record in an edit dialog",
      },
      {
        id: "next-record",
        label: "Next Record",
        keys: ["↓"],
        description: "Navigate to the next record in an edit dialog",
      },
    ],
  },
];
