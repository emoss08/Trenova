import type { IconComponent } from "@trenova/shared/components/icons";
import type { ReactNode } from "react";

/**
 * One row of a context menu, declared rather than built: the caller says
 * what the row is and what it does, and the menu decides how it looks, where
 * it goes and how it is reached from the keyboard. Every surface that opens a
 * menu — a list row, a sidebar folder, an attachment, a fact — declares its
 * rows this way, so they all behave alike.
 */
export type ContextMenuItem = {
  type?: "item";
  /** Stable within its menu; the label when absent. */
  id?: string;
  label: string;
  /** A glyph on the left. */
  icon?: IconComponent;
  /** Something other than a glyph on the left: an avatar, a kind's dot. Wins over `icon`. */
  leading?: ReactNode;
  /** Quiet text after the label: an address, a time, a count. */
  hint?: string;
  /** The key that does the same from the page, drawn on the right. */
  shortcut?: string;
  /** Drawn in the danger tone and moved to the foot of the menu. */
  destructive?: boolean;
  disabled?: boolean;
  /** Runs after the menu has closed. Ignored on a row with a submenu. */
  onSelect?: () => void;
  /** Rows of a submenu that opens beside this one on hover, ArrowRight or Enter. */
  submenu?: ContextMenuEntries;
};

export type ContextMenuSeparator = { type: "separator" };

/** A quiet heading over the rows after it: "Until something happens". */
export type ContextMenuHeader = { type: "header"; label: string };

export type ContextMenuEntry = ContextMenuItem | ContextMenuSeparator | ContextMenuHeader;

/** Rows as written, with `false`, `null` and `undefined` allowed so a row can be conditional inline. */
export type ContextMenuEntries = ReadonlyArray<ContextMenuEntry | false | null | undefined>;

export type ContextMenuSpec = {
  /** The first line: what the menu is about ("Northwind Foods", "3 selected"). */
  title?: string;
  /** The line under the title. */
  subtitle?: string;
  /** Names the menu for assistive technology when there is no title. */
  label?: string;
  entries: ContextMenuEntries;
};

export function isItem(entry: ContextMenuEntry): entry is ContextMenuItem {
  return entry.type === undefined || entry.type === "item";
}

/** Whether a row can take focus and be chosen. */
export function isActionable(entry: ContextMenuEntry): entry is ContextMenuItem {
  return isItem(entry) && entry.disabled !== true;
}

function present(entry: ContextMenuEntry | false | null | undefined): entry is ContextMenuEntry {
  return entry !== false && entry !== null && entry !== undefined;
}

/**
 * The rows a menu draws. Conditional rows that are off are dropped; the
 * destructive rows move to the foot of the menu, after a separator, so a
 * delete is never between two safe choices; and separators and headers that
 * would divide nothing — at either end, twice in a row, or a heading with no
 * rows under it — are dropped too.
 */
export function normalizeEntries(entries: ContextMenuEntries): ContextMenuEntry[] {
  const rows = entries.filter(present);
  const safe = rows.filter((entry) => !(isItem(entry) && entry.destructive === true));
  const destructive = rows.filter((entry) => isItem(entry) && entry.destructive === true);
  const ordered: ContextMenuEntry[] =
    destructive.length > 0 ? [...safe, { type: "separator" }, ...destructive] : safe;

  const result: ContextMenuEntry[] = [];
  for (const entry of ordered) {
    if (entry.type === "separator") {
      const last = result.at(-1);
      if (last === undefined || last.type === "separator") {
        continue;
      }
      if (last.type === "header") {
        result.pop();
        continue;
      }
    }
    result.push(entry);
  }

  while (result.length > 0) {
    const last = result[result.length - 1];
    if (last.type === "separator" || last.type === "header") {
      result.pop();
      continue;
    }
    break;
  }

  return result;
}

export function entryKey(entry: ContextMenuEntry, index: number): string {
  if (isItem(entry)) {
    return entry.id ?? entry.label;
  }

  return entry.type === "header" ? `header:${entry.label}` : `separator:${index}`;
}
