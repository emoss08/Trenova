export const searchableEntityTypes = ["shipment", "customer", "worker", "document"] as const;

export type SearchableEntityType = (typeof searchableEntityTypes)[number];

export interface SearchEntityOption {
  key: SearchableEntityType;
  label: string;
  aliases: string[];
}

export const searchEntityOptions: SearchEntityOption[] = [
  {
    key: "shipment",
    label: "Shipments",
    aliases: ["s", "ship", "shipment", "shipments"],
  },
  {
    key: "customer",
    label: "Customers",
    aliases: ["c", "cust", "customer", "customers"],
  },
  {
    key: "worker",
    label: "Workers",
    aliases: ["w", "worker", "workers"],
  },
  {
    key: "document",
    label: "Documents",
    aliases: ["d", "doc", "document", "documents"],
  },
];

export interface SearchMentionState {
  activeFilter: SearchableEntityType | null;
  mentionOpen: boolean;
  mentionText: string;
}

export function resolveEntityAlias(value: string): SearchableEntityType | null {
  const normalized = value.trim().toLowerCase();
  if (!normalized) {
    return null;
  }

  const exactMatch =
    searchEntityOptions.find(
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

export function getMentionState(input: string): SearchMentionState {
  const lastAtIndex = input.lastIndexOf("@");
  if (lastAtIndex < 0) {
    return { activeFilter: null, mentionOpen: false, mentionText: "" };
  }

  const suffix = input.slice(lastAtIndex + 1);
  if (suffix.includes(" ")) {
    return { activeFilter: null, mentionOpen: false, mentionText: "" };
  }

  const normalized = suffix.trim().toLowerCase();
  const activeFilter = resolveEntityAlias(normalized);

  return {
    activeFilter,
    mentionOpen: true,
    mentionText: normalized,
  };
}

export function filterMentionOptions(mentionText: string): SearchEntityOption[] {
  const normalized = mentionText.trim().toLowerCase();
  if (!normalized) {
    return searchEntityOptions;
  }

  return searchEntityOptions.filter(
    (option) =>
      option.label.toLowerCase().includes(normalized) ||
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
