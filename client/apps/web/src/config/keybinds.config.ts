import { formatForDisplay, type Hotkey } from "@tanstack/react-hotkeys";
import { isMacPlatform } from "@trenova/shared/lib/shortcuts";

export interface Keybind {
  id: string;
  label: string;
  /**
   * The keys as printed on the keyboard, with "Mod" for the platform's primary modifier
   * (⌘ on a Mac, Ctrl elsewhere); `keybindKeys` spells them for the person's platform.
   */
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
        keys: ["Mod", "K"],
        description: "Search records, jump to pages, run commands and ask the assistant",
      },
      {
        id: "toggle-sidebar",
        label: "Fold the sidebar",
        keys: ["Mod", "B"],
        description: "Fold the module's pages away beside the rail, or bring them back",
      },
      {
        id: "user-settings",
        label: "User settings",
        keys: ["Mod", "Shift", "S"],
        description: "Open your settings from any page",
        hotkey: "Mod+Shift+S",
      },
      {
        id: "keyboard-shortcuts",
        label: "Keyboard shortcuts",
        keys: ["Mod", "/"],
        description: "Show this keyboard shortcuts dialog",
        hotkey: "Mod+/",
      },
      {
        id: "assistant",
        label: "Assistant",
        keys: ["Mod", "J"],
        description: "Open or close the assistant from any page",
      },
      {
        id: "page-access",
        label: "Page access",
        keys: ["Mod", "Alt", "P"],
        description: "See what this screen needs and what each role sees on it",
        hotkey: "Mod+Alt+P",
      },
    ],
  },
  {
    id: "go-to",
    label: "Go to",
    keybinds: [
      { id: "go-home", label: "Home", keys: ["G", "H"], description: "Open Home from anywhere" },
      { id: "go-desk", label: "Desk", keys: ["G", "D"], description: "Open the Desk" },
      { id: "go-triage", label: "Triage", keys: ["G", "T"], description: "Open Triage" },
      { id: "go-intake", label: "Intake", keys: ["G", "I"], description: "Open Intake" },
      { id: "go-shipments", label: "Shipments", keys: ["G", "S"], description: "Open Shipments" },
      { id: "go-dispatch", label: "Dispatch", keys: ["G", "P"], description: "Open Dispatch" },
      { id: "go-equipment", label: "Equipment", keys: ["G", "E"], description: "Open Equipment" },
      { id: "go-fuel", label: "Fuel", keys: ["G", "F"], description: "Open Fuel & tax" },
      { id: "go-billing", label: "Billing", keys: ["G", "B"], description: "Open Billing" },
      {
        id: "go-carrier-settlements",
        label: "Carrier settlements",
        keys: ["G", "C"],
        description: "Open Carrier settlements",
      },
      {
        id: "go-accounting",
        label: "Accounting",
        keys: ["G", "A"],
        description: "Open Accounting",
      },
      { id: "go-reports", label: "Reports", keys: ["G", "R"], description: "Open Reports" },
      {
        id: "go-settings",
        label: "Settings",
        keys: ["G", ","],
        description: "Open Organization settings",
      },
    ],
  },
  {
    id: "desk",
    label: "Desk",
    keybinds: [
      {
        id: "desk-new",
        label: "New conversation",
        keys: ["Mod", "Shift", "O"],
        description: "Go to Today and start a new conversation",
      },
      {
        id: "desk-rail",
        label: "Hide the rail",
        keys: ["Mod", "B"],
        description: "Fold the list of conversations away, or bring it back",
      },
      {
        id: "desk-ask",
        label: "Ask Nova",
        keys: ["Mod", "J"],
        description: "Put the caret in the box in front of you, or open Today to ask from there",
      },
      {
        id: "desk-workspace",
        label: "Workspace",
        keys: ["Mod", "\\"],
        description: "Open the workspace on everything the conversation made, or fold it away",
      },
      {
        id: "desk-go-today",
        label: "Today",
        keys: ["G", "D"],
        description: "Go to Today from anywhere in the Desk",
      },
      {
        id: "desk-go-needs",
        label: "Needs you",
        keys: ["G", "N"],
        description: "Go to what waits on you, one row per record",
      },
      {
        id: "desk-go-watchtower",
        label: "Watchtower",
        keys: ["G", "W"],
        description: "Go to the Watchtower queue",
      },
      {
        id: "desk-go-triage",
        label: "Triage",
        keys: ["G", "T"],
        description: "Go to Triage inside the Desk",
      },
      {
        id: "desk-lens",
        label: "Lens on Needs you",
        keys: ["1–4"],
        description: "Everything, Watchtower, Triage or Decisions",
      },
      {
        id: "desk-approve",
        label: "Approve",
        keys: ["Mod", "Enter"],
        description:
          "Approve the change waiting in the conversation, while the message box is empty",
      },
      {
        id: "desk-starter",
        label: "Ask a starter question",
        keys: ["Mod", "1–3"],
        description: "Ask one of the agent's starter questions outright",
      },
      {
        id: "desk-queue",
        label: "Queue a follow-up",
        keys: ["Alt", "Enter"],
        description: "Send the message after the reply under way finishes",
      },
      {
        id: "desk-stop",
        label: "Stop the reply",
        keys: ["Esc"],
        description: "Stop the agent's reply from the message box, even with text in it",
      },
      {
        id: "desk-attach",
        label: "Attach files",
        keys: ["Mod", "U"],
        description: "Add files to the message",
      },
      {
        id: "desk-shortcuts",
        label: "Keyboard shortcuts",
        keys: ["?"],
        description: "Show this list from anywhere in the Desk outside a text box",
      },
    ],
  },
  {
    id: "dispatch-console",
    label: "Dispatch console",
    keybinds: [
      {
        id: "dispatch-next",
        label: "Next and previous move",
        keys: ["J", "K"],
        description: "Walk the queue in the order it is drawn, opening each move in the inspector",
      },
      {
        id: "dispatch-approve",
        label: "Approve Nova's proposal",
        keys: ["A"],
        description:
          "Approve the driver Nova proposed for the open move, once its dry run has loaded",
      },
      {
        id: "dispatch-decline",
        label: "Decline Nova's proposal",
        keys: ["D"],
        description: "Open the reason box to decline the proposal for the open move",
      },
      {
        id: "dispatch-pick",
        label: "Pick a driver",
        keys: ["1–9"],
        description: "Open that ranked driver's pre-flight; Enter assigns once it has loaded",
      },
      {
        id: "dispatch-undo",
        label: "Undo",
        keys: ["U"],
        description:
          "Take back the last assignment or approval, putting back any driver it replaced",
      },
      {
        id: "dispatch-search",
        label: "Search moves",
        keys: ["/"],
        description: "Search the board by PRO, customer or city",
      },
      {
        id: "dispatch-window",
        label: "Move the window",
        keys: ["←", "→"],
        description: "Show the previous or next window; T jumps back to today",
      },
      {
        id: "dispatch-close",
        label: "Back out",
        keys: ["Esc"],
        description: "Close the open pre-flight or decline, then the inspector",
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
        keys: ["Mod", "Enter"],
        description: "Open the selected result in a new browser tab",
      },
      {
        id: "palette-copy-link",
        label: "Copy link",
        keys: ["Mod", "L"],
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
        keys: ["Mod", "Enter"],
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
        keys: ["Mod", "B"],
        description: "Toggle bold formatting in the comment editor",
      },
      {
        id: "comment-italic",
        label: "Italic",
        keys: ["Mod", "I"],
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
        keys: ["Mod", "Enter"],
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
        keys: ["Mod", "S"],
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
    id: "triage",
    label: "Triage",
    keybinds: [
      {
        id: "triage-briefing",
        label: "Briefing",
        keys: ["B"],
        description: "Go back to the briefing",
      },
      {
        id: "triage-next",
        label: "Next message",
        keys: ["J"],
        description: "Open the next message; the down arrow does the same",
      },
      {
        id: "triage-previous",
        label: "Previous message",
        keys: ["K"],
        description: "Open the previous message; the up arrow does the same",
      },
      {
        id: "triage-search",
        label: "Search",
        keys: ["/"],
        description: "Search the open folder",
      },
      {
        id: "triage-panel",
        label: "Actions panel",
        keys: ["I"],
        description: "Show or hide the actions and information beside the message",
      },
      {
        id: "triage-do",
        label: "Do the next step",
        keys: ["Enter"],
        description: "Do the next open step of the message",
      },
      {
        id: "triage-handle",
        label: "Mark handled",
        keys: ["E"],
        description: "Mark the message, or every selected one, handled",
      },
      {
        id: "triage-ignore",
        label: "Ignore",
        keys: ["#"],
        description: "Ignore the message, or every selected one",
      },
      {
        id: "triage-reply",
        label: "Reply",
        keys: ["R"],
        description: "Write a reply to the sender",
      },
      {
        id: "triage-snooze",
        label: "Snooze until tomorrow",
        keys: ["S"],
        description: "Snooze the message, or every selected one, until tomorrow morning",
      },
      {
        id: "triage-assign",
        label: "Assign to me",
        keys: ["M"],
        description: "Assign the message, or every selected one, to you",
      },
      {
        id: "triage-select",
        label: "Select",
        keys: ["X"],
        description: "Select or deselect the open message for a bulk change",
      },
      {
        id: "triage-link",
        label: "Link to a record",
        keys: ["L"],
        description: "Say which shipment or customer the message is about",
      },
      {
        id: "triage-desk",
        label: "Ask an agent in Desk",
        keys: ["A"],
        description: "Hand the message and its open steps to one of your agents",
      },
      {
        id: "triage-shortcuts",
        label: "Shortcuts",
        keys: ["?"],
        description: "Show these shortcuts",
      },
      {
        id: "triage-escape",
        label: "Back out",
        keys: ["Esc"],
        description: "Clear the selection, then close the panel, then close the message",
      },
    ],
  },
  {
    id: "watchtower",
    label: "Watchtower",
    keybinds: [
      {
        id: "watchtower-next",
        label: "Next item",
        keys: ["J"],
        description: "Move to the next item in the queue; the down arrow does the same",
      },
      {
        id: "watchtower-previous",
        label: "Previous item",
        keys: ["K"],
        description: "Move to the previous item in the queue; the up arrow does the same",
      },
      {
        id: "watchtower-hand-off",
        label: "Hand off",
        keys: ["H"],
        description: "Hand the item to the agent best placed to take it, when no agent has it",
      },
      {
        id: "watchtower-done",
        label: "Done",
        keys: ["E"],
        description: "Mark the item done; Undo brings it back",
      },
      {
        id: "watchtower-snooze",
        label: "Snooze",
        keys: ["S"],
        description: "Set the item aside until tomorrow morning; Undo brings it back",
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
        keys: ["Mod", "L"],
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
        keys: ["Mod", "Enter"],
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

/**
 * A configured keybind as the person's keyboard spells it, for a hint beside the control it
 * runs: "⇧⌘S" on a Mac (run together, the way macOS menus print it), "Ctrl+Shift+S" elsewhere.
 */
export function hotkeyLabel(groupId: string, keybindId: string): string {
  const hotkey = hotkeyOf(groupId, keybindId);
  return isMacPlatform()
    ? formatForDisplay(hotkey, { platform: "mac", separatorToken: "" })
    : formatForDisplay(hotkey, { platform: "windows" });
}

const MAC_MODIFIERS: Readonly<Record<string, string>> = { Alt: "⌥", Shift: "⇧", Mod: "⌘" };
/** The order macOS prints modifiers in: ⌥ before ⇧ before ⌘. */
const MAC_MODIFIER_ORDER = ["Alt", "Shift", "Mod"];

/**
 * A keybind's keys as the person's keyboard spells them, one label per key: ⇧ ⌘ S on a Mac,
 * in the order macOS prints modifiers, and Ctrl Shift S elsewhere.
 */
export function keybindKeys(keys: readonly string[], mac: boolean = isMacPlatform()): string[] {
  if (!mac) return keys.map((key) => (key === "Mod" ? "Ctrl" : key));
  const modifiers = MAC_MODIFIER_ORDER.filter((modifier) => keys.includes(modifier));
  const rest = keys.filter((key) => !MAC_MODIFIER_ORDER.includes(key));
  return [...modifiers.map((modifier) => MAC_MODIFIERS[modifier] ?? modifier), ...rest];
}
