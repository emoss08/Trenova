import type { PaletteScope } from "./palette-model";

/** Anything an `@` mention can narrow the palette to: a kind of record, the pages, or the commands. */
export type SearchableEntityType = Exclude<PaletteScope, "all">;

export interface SearchEntityOption {
  key: SearchableEntityType;
  label: string;
  aliases: string[];
}

export const searchEntityOptions: SearchEntityOption[] = [
  {
    key: "shipment",
    label: "Shipments",
    aliases: ["s", "ship", "shipment", "shipments", "load", "loads"],
  },
  {
    key: "customer",
    label: "Customers",
    aliases: ["c", "cust", "customer", "customers"],
  },
  {
    key: "invoice",
    label: "Invoices",
    aliases: ["i", "inv", "invoice", "invoices", "bill", "bills"],
  },
  {
    key: "worker",
    label: "Workers",
    aliases: ["w", "worker", "workers", "driver", "drivers"],
  },
  {
    key: "document",
    label: "Documents",
    aliases: ["d", "doc", "docs", "document", "documents"],
  },
  {
    key: "pages",
    label: "Pages",
    aliases: ["page", "pages"],
  },
  {
    key: "commands",
    label: "Commands",
    aliases: ["cmd", "command", "commands"],
  },
];

export interface SearchMentionState {
  activeFilter: SearchableEntityType | null;
  mentionOpen: boolean;
  mentionText: string;
}

/**
 * The scopes an `@` mention may settle on: the ones the palette offers, so a
 * kind of record the person may not read is never a filter they can type.
 */
export type AvailableScopes = ReadonlySet<SearchableEntityType>;

function offered(available: AvailableScopes): SearchEntityOption[] {
  return searchEntityOptions.filter((option) => available.has(option.key));
}

export function resolveEntityAlias(
  value: string,
  available: AvailableScopes,
): SearchableEntityType | null {
  const normalized = value.trim().toLowerCase();
  if (!normalized) {
    return null;
  }

  const exactMatch =
    offered(available).find(
      (option) => option.label.toLowerCase() === normalized || option.aliases.includes(normalized),
    ) ?? null;

  return exactMatch?.key ?? null;
}

export function getSearchEntityOption(
  entityType: SearchableEntityType | null,
): SearchEntityOption | null {
  if (!entityType) {
    return null;
  }

  return searchEntityOptions.find((option) => option.key === entityType) ?? null;
}

export function getMentionState(input: string, available: AvailableScopes): SearchMentionState {
  const lastAtIndex = input.lastIndexOf("@");
  if (lastAtIndex < 0 || (lastAtIndex > 0 && !/\s/.test(input.charAt(lastAtIndex - 1)))) {
    return { activeFilter: null, mentionOpen: false, mentionText: "" };
  }

  const suffix = input.slice(lastAtIndex + 1);
  if (/\s/.test(suffix)) {
    return { activeFilter: null, mentionOpen: false, mentionText: "" };
  }

  const normalized = suffix.trim().toLowerCase();
  const activeFilter = resolveEntityAlias(normalized, available);

  return {
    activeFilter,
    mentionOpen: true,
    mentionText: normalized,
  };
}

export function filterMentionOptions(
  mentionText: string,
  available: AvailableScopes,
): SearchEntityOption[] {
  const normalized = mentionText.trim().toLowerCase();
  if (!normalized) {
    return offered(available);
  }

  return offered(available).filter(
    (option) =>
      option.label.toLowerCase().startsWith(normalized) ||
      option.aliases.some((alias) => alias.startsWith(normalized)),
  );
}

export function stripMentionToken(input: string): string {
  const lastAtIndex = input.lastIndexOf("@");
  if (lastAtIndex < 0) {
    return input.trim();
  }

  return input.slice(0, lastAtIndex).trim();
}

/**
 * The record type an `@` mention settles on once a space ends it: an exact
 * alias, or the only type the typed prefix can still mean ("@work" is
 * workers). An ambiguous or unknown prefix settles on nothing.
 */
export function resolveMentionCommit(
  value: string,
  available: AvailableScopes,
): SearchableEntityType | null {
  const exact = resolveEntityAlias(value, available);
  if (exact) {
    return exact;
  }
  if (value.trim() === "") {
    return null;
  }
  const candidates = filterMentionOptions(value, available);
  return candidates.length === 1 ? (candidates[0]?.key ?? null) : null;
}
