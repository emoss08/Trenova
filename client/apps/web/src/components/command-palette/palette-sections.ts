import type { GlobalSearchGroup } from "@/services/global-search";
import { isGlobalSearchEntityType } from "@/services/global-search";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { fuzzyScore, rankByFuzzyScore, type ScoredField } from "@trenova/shared/lib/fuzzy-score";
import type { Notification } from "@trenova/shared/types/notification";
import { COMMAND_GROUP_ORDER, commandGroupLabel } from "./palette-commands";
import { PALETTE_ENTITIES } from "./palette-entities";
import {
  isRecordScope,
  recordItemKey,
  type PaletteAttention,
  type PaletteCommand,
  type PaletteItem,
  type PalettePage,
  type PaletteRecord,
  type PaletteScope,
  type PaletteSection,
} from "./palette-model";

export const MIN_REMOTE_QUERY_LENGTH = 2;
export const TOP_HIT_THRESHOLD = 0.9;

const HOME_RECENT_RECORDS = 5;
const HOME_RECENT_PAGES = 4;
const HOME_SUGGESTED = 6;
const SEARCH_PAGES = 6;
const SEARCH_COMMANDS = 6;
const SCOPED_LIMIT = 50;

export function pageFields(page: PalettePage): ScoredField[] {
  return [
    { text: page.title },
    { text: page.keywords.join(" "), weight: 0.8, subsequence: false },
    { text: page.trail, weight: 0.6, subsequence: false },
  ];
}

export function commandFields(command: PaletteCommand): ScoredField[] {
  return [
    { text: command.label },
    { text: command.keywords.join(" "), weight: 0.8, subsequence: false },
    { text: command.description, weight: 0.5, subsequence: false },
  ];
}

function pageItem(page: PalettePage, extra?: { pinned?: boolean; recent?: boolean }): PaletteItem {
  const prefix = extra?.pinned ? "pinned" : extra?.recent ? "recent-page" : "page";
  return { kind: "page", key: `${prefix}:${page.href}`, page, ...extra };
}

function commandItem(command: PaletteCommand): PaletteItem {
  return { kind: "command", key: `command:${command.id}`, command };
}

export function recordItem(record: PaletteRecord, recent = false): PaletteItem {
  const key = recordItemKey(record.entityType, record.id);
  return { kind: "record", key: recent ? `recent-${key}` : key, record, recent };
}

/** Search hits as palette records; a type the palette does not know is dropped, not guessed at. */
export function recordsFromGroups(
  groups: readonly GlobalSearchGroup[],
): Map<string, PaletteRecord[]> {
  const byType = new Map<string, PaletteRecord[]>();
  for (const group of groups) {
    if (!isGlobalSearchEntityType(group.entityType)) {
      continue;
    }
    const entityType = group.entityType;
    byType.set(
      entityType,
      group.hits.map((hit) => ({
        entityType,
        id: hit.id,
        title: hit.metadata?.proNumber || hit.title,
        subtitle: hit.subtitle,
        href: hit.href,
        metadata: hit.metadata ?? {},
      })),
    );
  }
  return byType;
}

function section(
  id: string,
  heading: string,
  items: PaletteItem[],
  extra?: Partial<PaletteSection>,
): PaletteSection {
  return { id, heading, items, ...extra };
}

function nonEmpty(sections: PaletteSection[]): PaletteSection[] {
  return sections.filter((entry) => entry.items.length > 0 || entry.loading || entry.error);
}

export interface HomeInput {
  recentRecords: readonly PaletteRecord[];
  attention: { rows: readonly PaletteAttention[]; loading: boolean };
  notifications: {
    items: readonly { notification: Notification; href: string | null }[];
    unread: number;
    loading: boolean;
  };
  pinnedPages: { pages: readonly PalettePage[]; loading: boolean };
  recentPages: readonly PalettePage[];
  suggested: readonly PaletteCommand[];
}

/**
 * What the palette shows before anything is typed: where the person just
 * was, what is waiting on them, and the few things they most often start.
 * Each section loads on its own, so a slow count never holds up the rest.
 */
export function buildHomeSections(input: HomeInput, t: TranslateFn): PaletteSection[] {
  const pinnedHrefs = new Set(input.pinnedPages.pages.map((page) => page.href));
  const recentPages = input.recentPages
    .filter((page) => !pinnedHrefs.has(page.href))
    .slice(0, HOME_RECENT_PAGES);

  return nonEmpty([
    section(
      "recent-records",
      t("Recent records"),
      input.recentRecords.slice(0, HOME_RECENT_RECORDS).map((record) => recordItem(record, true)),
    ),
    section(
      "attention",
      t("Needs attention"),
      input.attention.rows.map((attention) => ({
        kind: "attention",
        key: `attention:${attention.key}`,
        attention,
      })),
      { loading: input.attention.loading && input.attention.rows.length === 0, placeholderRows: 2 },
    ),
    section(
      "notifications",
      t("Unread notifications"),
      input.notifications.items.map(({ notification, href }) => ({
        kind: "notification",
        key: `notification:${notification.id}`,
        notification,
        href,
      })),
      {
        count: input.notifications.unread,
        loading: input.notifications.loading && input.notifications.items.length === 0,
        placeholderRows: 2,
      },
    ),
    section(
      "pinned",
      t("Pinned"),
      input.pinnedPages.pages.map((page) => pageItem(page, { pinned: true })),
      {
        loading: input.pinnedPages.loading && input.pinnedPages.pages.length === 0,
        placeholderRows: 2,
      },
    ),
    section(
      "recent-pages",
      t("Recently visited"),
      recentPages.map((page) => pageItem(page, { recent: true })),
    ),
    section(
      "suggested",
      t("Quick actions"),
      input.suggested.slice(0, HOME_SUGGESTED).map(commandItem),
    ),
  ]);
}

export interface RemoteState {
  groups: readonly GlobalSearchGroup[];
  loading: boolean;
  error: boolean;
  /** The query is long enough to have been sent. */
  ready: boolean;
}

export interface SearchInput {
  query: string;
  scope: PaletteScope;
  question: string | null;
  remote: RemoteState;
  pages: readonly PalettePage[];
  commands: readonly PaletteCommand[];
  recentRecords: readonly PaletteRecord[];
}

function groupPagesByModule(pages: readonly PalettePage[]): PaletteSection[] {
  const byModule = new Map<string, PalettePage[]>();
  for (const page of pages) {
    const list = byModule.get(page.module) ?? [];
    list.push(page);
    byModule.set(page.module, list);
  }
  return Array.from(byModule, ([module, modulePages]) =>
    section(
      `module:${module}`,
      module,
      modulePages.map((page) => pageItem(page)),
    ),
  );
}

function groupCommands(commands: readonly PaletteCommand[], t: TranslateFn): PaletteSection[] {
  return COMMAND_GROUP_ORDER.map((group) =>
    section(
      `commands:${group}`,
      commandGroupLabel(group, t),
      commands.filter((command) => command.group === group).map(commandItem),
    ),
  );
}

function recordSections(
  remote: RemoteState,
  scope: PaletteScope,
  t: TranslateFn,
): PaletteSection[] {
  if (!remote.ready) {
    return [];
  }
  const types = isRecordScope(scope)
    ? [scope]
    : (Object.keys(PALETTE_ENTITIES) as (keyof typeof PALETTE_ENTITIES)[]);
  if (remote.error) {
    return [section("records-error", t("Records"), [], { error: true })];
  }
  if (remote.loading && remote.groups.length === 0) {
    return [section("records-loading", t("Records"), [], { loading: true, placeholderRows: 3 })];
  }

  const byType = recordsFromGroups(remote.groups);
  return types.map((type) =>
    section(
      `records:${type}`,
      t(PALETTE_ENTITIES[type].pluralLabel),
      (byType.get(type) ?? []).map((record) => recordItem(record)),
    ),
  );
}

/**
 * What the palette shows for a query, in a fixed order a person can learn:
 * the one obvious answer first when there is one, then records, pages and
 * commands. The palette does its own ranking so the order never shifts
 * under the cursor the way a DOM-sorting filter does.
 */
export function buildSearchSections(input: SearchInput, t: TranslateFn): PaletteSection[] {
  const { query, scope } = input;
  const trimmed = query.trim();

  if (scope === "pages") {
    const ranked = rankByFuzzyScore(trimmed, input.pages, pageFields);
    return nonEmpty(
      trimmed === ""
        ? groupPagesByModule(ranked)
        : [
            section(
              "pages",
              t("Pages"),
              ranked.slice(0, SCOPED_LIMIT).map((page) => pageItem(page)),
            ),
          ],
    );
  }

  if (scope === "commands") {
    return nonEmpty(groupCommands(rankByFuzzyScore(trimmed, input.commands, commandFields), t));
  }

  if (isRecordScope(scope)) {
    if (trimmed.length < MIN_REMOTE_QUERY_LENGTH) {
      return nonEmpty([
        section(
          "recent-records",
          t("Recent"),
          input.recentRecords
            .filter((record) => record.entityType === scope)
            .map((record) => recordItem(record, true)),
        ),
      ]);
    }
    return nonEmpty(recordSections(input.remote, scope, t));
  }

  const sections: PaletteSection[] = [];
  if (input.question !== null) {
    sections.push(
      section("ask", t("Assistant"), [{ kind: "ask", key: "ask", question: input.question }]),
    );
  }

  const pages = rankByFuzzyScore(trimmed, input.pages, pageFields);
  const commands = rankByFuzzyScore(trimmed, input.commands, commandFields);

  const topPage = pages[0];
  const topCommand = commands[0];
  const topPageScore = topPage ? fuzzyScore(trimmed, pageFields(topPage)) : 0;
  const topCommandScore = topCommand ? fuzzyScore(trimmed, commandFields(topCommand)) : 0;
  let topHit: PaletteItem | null = null;
  if (Math.max(topPageScore, topCommandScore) >= TOP_HIT_THRESHOLD) {
    topHit =
      topPageScore >= topCommandScore && topPage ? pageItem(topPage) : commandItem(topCommand!);
    sections.push(section("top-hit", t("Top hit"), [topHit]));
  }

  sections.push(...recordSections(input.remote, scope, t));
  sections.push(
    section(
      "pages",
      t("Pages"),
      pages
        .map((page) => pageItem(page))
        .filter((item) => item.key !== topHit?.key)
        .slice(0, SEARCH_PAGES),
    ),
    section(
      "commands",
      t("Commands"),
      commands
        .map(commandItem)
        .filter((item) => item.key !== topHit?.key)
        .slice(0, SEARCH_COMMANDS),
    ),
  );

  return nonEmpty(sections);
}

/** Every item in the order it is drawn, for keyboard shortcuts and the preview. */
export function flattenSections(sections: readonly PaletteSection[]): PaletteItem[] {
  return sections.flatMap((entry) => entry.items);
}
