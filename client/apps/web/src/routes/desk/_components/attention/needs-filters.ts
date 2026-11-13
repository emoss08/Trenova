import { humanizeCode } from "@/components/assistant/readable-values";
import type { DeskAttentionFilter, DeskAttentionKind } from "@/lib/graphql/desk-attention";
import type { WatchtowerSeverity, WatchtowerSourceKind } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { parseAsArrayOf, parseAsBoolean, parseAsString } from "nuqs";
import { SEVERITIES, severityLabel } from "../watchtower/severity";
import type { NeedsLens } from "./lens";

/**
 * What Needs you is searched for and narrowed to, kept in the address, so a
 * link sent to someone opens the same list, a reload comes back to it, and
 * the briefing's links can name a kind or a severity. The words searched for
 * are the page's, not a lens's: they stay as the person goes from lens to
 * lens, and each lens is narrowed by them.
 */
export const NEEDS_FILTER_ADDRESS = {
  kinds: parseAsArrayOf(parseAsString, ",").withDefault([]),
  severities: parseAsArrayOf(parseAsString, ",").withDefault([]),
  q: parseAsString.withDefault(""),
  unseen: parseAsBoolean.withDefault(false),
  working: parseAsBoolean.withDefault(false),
};

/** The item open beside the list, kept in the address so a link opens on it. */
export const NEEDS_ITEM_ADDRESS = { item: parseAsString };

/** What the list is narrowed to, apart from its lens and whose cases it holds. */
export type NeedsFilters = {
  search: string;
  severities: readonly WatchtowerSeverity[];
  kinds: readonly WatchtowerSourceKind[];
  /** Only rows with something the reader has not opened yet. */
  unseen: boolean;
  /** Only rows an agent's run is working on right now. */
  working: boolean;
};

export type NeedsAddress = {
  kinds: string[];
  severities: string[];
  q: string;
  unseen: boolean;
  working: boolean;
};

/** The filters an address names. A severity it misspells is dropped; a kind is the server's to judge. */
export function filtersFromAddress(address: NeedsAddress): NeedsFilters {
  return {
    search: address.q.trim(),
    severities: SEVERITIES.filter((severity) => address.severities.includes(severity)),
    kinds: [...new Set(address.kinds)] as WatchtowerSourceKind[],
    unseen: address.unseen,
    working: address.working,
  };
}

/** The address a set of filters is kept as; a filter that is off leaves the address. */
export function addressOf(filters: NeedsFilters) {
  return {
    q: filters.search.trim() || null,
    severities: filters.severities.length > 0 ? [...filters.severities] : null,
    kinds: filters.kinds.length > 0 ? [...filters.kinds] : null,
    unseen: filters.unseen || null,
    working: filters.working || null,
  };
}

/** What a list with nothing narrowed looks like, for "Clear". */
export const NO_FILTERS: NeedsFilters = {
  search: "",
  severities: [],
  kinds: [],
  unseen: false,
  working: false,
};

/** How many filters are on. The search is not one: it has its own box. */
export function filterCount(filters: NeedsFilters): number {
  return (
    filters.severities.length +
    filters.kinds.length +
    (filters.unseen ? 1 : 0) +
    (filters.working ? 1 : 0)
  );
}

/** Whether the list shows less than its lens holds. */
export function isNarrowed(filters: NeedsFilters): boolean {
  return filters.search !== "" || filterCount(filters) > 0;
}

/** What the server is asked for: the lens, whose cases, and the filters. */
export function attentionFilter(
  lens: NeedsLens,
  mine: boolean,
  filters: NeedsFilters,
): DeskAttentionFilter {
  return {
    lens,
    mine,
    kinds: filters.kinds,
    severities: filters.severities,
    search: filters.search,
    unseen: filters.unseen,
    working: filters.working,
  };
}

/** A value added to a list that lacks it, or taken out of one that has it. */
export function toggled<T>(list: readonly T[], value: T): T[] {
  return list.includes(value) ? list.filter((entry) => entry !== value) : [...list, value];
}

/**
 * The kinds the filter offers under a lens: those with something open that
 * are filed under it (every kind under Everything), most first as the server
 * sorted them, and any kind already chosen, so it can always be taken off.
 */
export function offeredKinds(
  kinds: readonly DeskAttentionKind[],
  lens: NeedsLens,
  chosen: readonly WatchtowerSourceKind[],
): DeskAttentionKind[] {
  return kinds.filter(
    (entry) =>
      chosen.includes(entry.kind) ||
      (entry.count > 0 && (lens === "Everything" || entry.lens === lens)),
  );
}

/** A kind in the reader's words: the server's label, or the code spelled out until it answers. */
export function kindLabel(kinds: readonly DeskAttentionKind[], kind: WatchtowerSourceKind): string {
  return kinds.find((entry) => entry.kind === kind)?.label ?? humanizeCode(kind);
}

export type NeedsFilterToken =
  | { key: string; type: "severity"; severity: WatchtowerSeverity; label: string }
  | { key: string; type: "kind"; kind: WatchtowerSourceKind; label: string }
  | { key: string; type: "unseen"; label: string }
  | { key: string; type: "working"; label: string };

/** One token per filter that is on, in the order the menu lists them. */
export function filterTokens(
  t: TranslateFn,
  filters: NeedsFilters,
  kinds: readonly DeskAttentionKind[],
): NeedsFilterToken[] {
  return [
    ...filters.severities.map((severity) => ({
      key: `severity:${severity}`,
      type: "severity" as const,
      severity,
      label: severityLabel(t, severity),
    })),
    ...(filters.unseen
      ? [{ key: "unseen", type: "unseen" as const, label: t("Not opened yet") }]
      : []),
    ...(filters.working
      ? [{ key: "working", type: "working" as const, label: t("An agent is working") }]
      : []),
    ...filters.kinds.map((kind) => ({
      key: `kind:${kind}`,
      type: "kind" as const,
      kind,
      label: kindLabel(kinds, kind),
    })),
  ];
}

/** The filters with one token's filter taken off. */
export function withoutToken(filters: NeedsFilters, token: NeedsFilterToken): NeedsFilters {
  switch (token.type) {
    case "severity":
      return {
        ...filters,
        severities: filters.severities.filter((entry) => entry !== token.severity),
      };
    case "kind":
      return { ...filters, kinds: filters.kinds.filter((entry) => entry !== token.kind) };
    case "unseen":
      return { ...filters, unseen: false };
    default:
      return { ...filters, working: false };
  }
}
