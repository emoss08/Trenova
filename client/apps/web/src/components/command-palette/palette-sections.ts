import { containsWholePhrase, foldPhrase } from "@/lib/whole-phrase";
import type { GlobalSearchGroup } from "@/services/global-search";
import { isGlobalSearchEntityType, type GlobalSearchEntityType } from "@/services/global-search";
import type { AssistantEntityRef } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { rankMatches, type RankedMatch } from "@trenova/shared/lib/fuzzy-score";
import { COMMAND_GROUP_ORDER, commandGroupLabel } from "./palette-commands";
import { paletteEntityText, PALETTE_RECORD_TYPES } from "./palette-entities";
import { commandFields, matchRecord, pageFields } from "./palette-fields";
import {
  isRecordScope,
  recordItemKey,
  type PaletteAttention,
  type PaletteCommand,
  type PaletteItem,
  type PaletteMatch,
  type PalettePage,
  type PaletteRecord,
  type PaletteScope,
  type PaletteSection,
} from "./palette-model";
import { subjectQuestions } from "./subject-questions";

export const MIN_REMOTE_QUERY_LENGTH = 2;
export const TOP_HIT_THRESHOLD = 0.9;
export const HOME_RECENT_RECORDS = 5;
export const RECORD_SECTION_CAP = 3;
export const PAGE_SECTION_CAP = 4;
export const COMMAND_SECTION_CAP = 3;
export const SCOPED_LIMIT = 50;

export interface RecentPaletteRecord {
  record: PaletteRecord;
  visitedAt: number;
}

export interface HomeInput {
  recentRecords: readonly RecentPaletteRecord[];
  attention: { rows: readonly PaletteAttention[]; loading: boolean };
  pinnedPages: { pages: readonly PalettePage[]; loading: boolean };
  quickActions: readonly PaletteCommand[];
}

export interface RemoteState {
  /** The answer on screen. While a newer search is in flight it is the last one that landed. */
  groups: readonly GlobalSearchGroup[];
  /** Nothing has landed for this search yet, so there is no answer to show in the meantime. */
  loading: boolean;
  error: boolean;
  /** The request is one worth sending: a long enough query, or a scope to browse. */
  ready: boolean;
}

export interface LocalRanking {
  pages: RankedMatch<PalettePage>[];
  commands: RankedMatch<PaletteCommand>[];
}

export interface SearchInput {
  query: string;
  /** What the records were searched for: the query, or a question's search words. */
  recordQuery: string;
  scope: PaletteScope;
  question: string | null;
  remote: RemoteState;
  ranking: LocalRanking;
  recentRecords: readonly RecentPaletteRecord[];
}

export interface SearchResults {
  sections: PaletteSection[];
  /** Nothing matched the query anywhere it was looked for. */
  empty: boolean;
}

function pageItem(page: PalettePage, extra?: { pinned?: boolean; match?: PaletteMatch }) {
  const prefix = extra?.pinned ? "pinned" : "page";
  return { kind: "page", key: `${prefix}:${page.href}`, page, ...extra } satisfies PaletteItem;
}

function commandItem(command: PaletteCommand, match?: PaletteMatch): PaletteItem {
  return { kind: "command", key: `command:${command.id}`, command, match };
}

export function recordItem(
  record: PaletteRecord,
  extra?: { recent?: boolean; visitedAt?: number; match?: PaletteMatch },
): PaletteItem {
  const key = recordItemKey(record.entityType, record.id);
  return { kind: "record", key: extra?.recent ? `recent-${key}` : key, record, ...extra };
}

export function askItem(question: string, subject: AssistantEntityRef | null = null): PaletteItem {
  const scope = subject ? `${subject.type}:${subject.id}:` : "";
  return { kind: "ask", key: `ask:${scope}${question}`, question, subject };
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

export interface FoundRecords {
  records: readonly PaletteRecord[];
  /** How many match in all, which the index can know beyond what it returned. */
  total: number;
}

const foundByGroups = new WeakMap<
  readonly GlobalSearchGroup[],
  ReadonlyMap<GlobalSearchEntityType, FoundRecords>
>();

/**
 * Search hits as palette records; a type the palette does not know is dropped,
 * not guessed at. One answer always gives the same records, so the list built
 * from it on every key hands each row the record it drew last time.
 */
export function recordsFromGroups(
  groups: readonly GlobalSearchGroup[],
): ReadonlyMap<GlobalSearchEntityType, FoundRecords> {
  const cached = foundByGroups.get(groups);
  if (cached) {
    return cached;
  }
  const byType = new Map<GlobalSearchEntityType, FoundRecords>();
  for (const group of groups) {
    if (!isGlobalSearchEntityType(group.entityType)) {
      continue;
    }
    const entityType = group.entityType;
    const records = group.hits.map((hit): PaletteRecord => ({
      entityType,
      id: hit.id,
      title: hit.metadata?.proNumber || hit.title,
      subtitle: hit.subtitle,
      href: hit.href,
      metadata: hit.metadata ?? {},
    }));
    byType.set(entityType, { records, total: Math.max(group.total ?? 0, records.length) });
  }
  foundByGroups.set(groups, byType);
  return byType;
}

/** Records a question carries to Nova at most, as the server takes them. */
export const MAX_ASK_MENTIONS = 10;
const MIN_NAMED_TITLE_LENGTH = 3;

/**
 * The records a search found that a question names outright: a PRO number, a
 * customer's or a driver's name, written in the question as the record carries
 * it. Only those go to Nova as records the person named. A record the index
 * found by a looser match, SEED-PAY-005 for "SEED-PAY-002", is left to Nova's
 * own lookups rather than handed over as though it had been asked about.
 */
export function recordsNamedIn(
  question: string,
  groups: readonly GlobalSearchGroup[],
): AssistantEntityRef[] {
  const named: AssistantEntityRef[] = [];
  const seen = new Set<string>();
  for (const [, found] of recordsFromGroups(groups)) {
    for (const record of found.records) {
      const key = recordItemKey(record.entityType, record.id);
      if (
        foldPhrase(record.title).length < MIN_NAMED_TITLE_LENGTH ||
        seen.has(key) ||
        !containsWholePhrase(question, record.title)
      ) {
        continue;
      }
      seen.add(key);
      named.push({ type: record.entityType, id: record.id, label: record.title });
      if (named.length === MAX_ASK_MENTIONS) {
        return named;
      }
    }
  }
  return named;
}

/**
 * Pages and commands ranked against the query. They are matched in the
 * browser, so they answer on the first key, and the same ranking feeds both
 * the list and the scope tabs' counts.
 */
export function rankLocal(
  query: string,
  pages: readonly PalettePage[],
  commands: readonly PaletteCommand[],
  t: TranslateFn,
): LocalRanking {
  const translate = (text: string) => t(text);
  return {
    pages: rankMatches(query, pages, (page) => pageFields(page, translate)),
    commands: rankMatches(query, commands, commandFields),
  };
}

interface RankedRecord {
  record: PaletteRecord;
  match: PaletteMatch | null;
  score: number;
}

/**
 * The index found these; the palette's matcher says which characters to mark
 * and orders them by the same rules as pages and commands. A hit the matcher
 * cannot place (the index forgives a typo) keeps its place after the ones it
 * can, and is shown without marks rather than dropped.
 */
function rankRecords(query: string, records: readonly PaletteRecord[]): RankedRecord[] {
  return records
    .map((record, index) => {
      const match = matchRecord(query, record);
      return { record, match, score: match?.score ?? 0, index };
    })
    .sort((a, b) => b.score - a.score || a.index - b.index)
    .map(({ record, match, score }) => ({ record, match, score }));
}

/**
 * What the palette shows before anything is typed: where the person just
 * was, what is waiting on them, the pages they keep, and the few things they
 * most often start. Each section loads on its own, so a slow count never
 * holds up the rest.
 */
export function buildHomeSections(input: HomeInput, t: TranslateFn): PaletteSection[] {
  return nonEmpty([
    section(
      "recent-records",
      t("Jump back in"),
      input.recentRecords
        .slice(0, HOME_RECENT_RECORDS)
        .map(({ record, visitedAt }) => recordItem(record, { recent: true, visitedAt })),
    ),
    section(
      "attention",
      t("Needs you"),
      input.attention.rows.map((attention) => ({
        kind: "attention",
        key: `attention:${attention.key}`,
        attention,
      })),
      { loading: input.attention.loading && input.attention.rows.length === 0, placeholderRows: 2 },
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
      "quick-actions",
      t("Quick actions"),
      input.quickActions.map((command) => commandItem(command)),
    ),
  ]);
}

function groupCommands(
  ranked: readonly RankedMatch<PaletteCommand>[],
  t: TranslateFn,
): PaletteSection[] {
  return COMMAND_GROUP_ORDER.map((group) =>
    section(
      `commands:${group}`,
      commandGroupLabel(group, t),
      ranked
        .filter(({ entry }) => entry.group === group)
        .map(({ entry, match }) => commandItem(entry, match)),
    ),
  );
}

function remoteStatusSections(remote: RemoteState, t: TranslateFn): PaletteSection[] | null {
  if (remote.error) {
    return [section("records-error", t("Records"), [], { error: true })];
  }
  if (remote.loading && remote.groups.length === 0) {
    return [section("records-loading", t("Records"), [], { loading: true, placeholderRows: 3 })];
  }
  return null;
}

/** One kind of record with nothing typed: what was opened last, then the rest of them. */
function browseRecordScope(
  scope: GlobalSearchEntityType,
  input: SearchInput,
  t: TranslateFn,
): PaletteSection[] {
  const recent = input.recentRecords.filter(({ record }) => record.entityType === scope);
  const recentIds = new Set(recent.map(({ record }) => record.id));
  const found = recordsFromGroups(input.remote.groups).get(scope);
  const rest = (found?.records ?? []).filter((record) => !recentIds.has(record.id));

  return nonEmpty([
    section(
      "recent-records",
      t("Recent"),
      recent.map(({ record, visitedAt }) => recordItem(record, { recent: true, visitedAt })),
    ),
    ...(remoteStatusSections(input.remote, t) ??
      (input.remote.ready
        ? [
            section(
              `records:${scope}`,
              paletteEntityText(scope, t).all,
              rest.map((record) => recordItem(record)),
              { count: found ? Math.max(found.total - recent.length, rest.length) : undefined },
            ),
          ]
        : [])),
  ]);
}

function fallbackAsk(query: string, empty: boolean, t: TranslateFn): PaletteSection {
  return section("ask-fallback", empty ? t("Try asking instead") : t("Nova"), [askItem(query)]);
}

function widenSection(query: string, t: TranslateFn): PaletteSection {
  return section("widen", t("Or"), [{ kind: "widen", key: "widen:all", query }]);
}

/**
 * What the palette shows for a query, in a fixed order a person can learn:
 * Nova first when it is a question, the one obvious answer when there is one,
 * then records, pages and commands, and always a way to ask instead. The
 * palette does its own ranking so the order never shifts under the cursor
 * the way a DOM-sorting filter does.
 */
export function buildSearchSections(input: SearchInput, t: TranslateFn): SearchResults {
  const { scope, ranking } = input;
  const query = input.query.trim();

  if (scope === "commands") {
    const sections = nonEmpty(groupCommands(ranking.commands, t));
    return { sections, empty: query !== "" && sections.length === 0 };
  }

  if (scope === "pages") {
    if (query === "") {
      return {
        sections: nonEmpty([
          section(
            "pages",
            t("All pages"),
            ranking.pages.map(({ entry }) => pageItem(entry)),
          ),
        ]),
        empty: false,
      };
    }
    const items = ranking.pages
      .slice(0, SCOPED_LIMIT)
      .map(({ entry, match }) => pageItem(entry, { match }));
    const empty = items.length === 0;
    return {
      sections: nonEmpty([
        section("pages", t("Pages"), items, { count: ranking.pages.length }),
        ...(empty ? [widenSection(query, t)] : []),
      ]),
      empty,
    };
  }

  if (isRecordScope(scope) && query === "") {
    return { sections: browseRecordScope(scope, input, t), empty: false };
  }

  const sections: PaletteSection[] = [];
  if (input.question !== null) {
    sections.push(section("nova", t("Nova"), [askItem(input.question)]));
  }

  if (isRecordScope(scope)) {
    const status = remoteStatusSections(input.remote, t);
    if (status) {
      return { sections: [...sections, ...status], empty: false };
    }
    const found = recordsFromGroups(input.remote.groups).get(scope);
    const ranked = rankRecords(input.recordQuery, found?.records ?? []);
    const empty = input.remote.ready && !input.remote.loading && ranked.length === 0;
    sections.push(
      section(
        `records:${scope}`,
        paletteEntityText(scope, t).plural,
        ranked
          .slice(0, SCOPED_LIMIT)
          .map(({ record, match }) => recordItem(record, { match: match ?? undefined })),
        { count: found?.total },
      ),
    );
    if (input.question === null) {
      sections.push(fallbackAsk(query, empty, t));
    }
    if (empty) {
      sections.push(widenSection(query, t));
    }
    return { sections: nonEmpty(sections), empty: empty && input.question === null };
  }

  const found = recordsFromGroups(input.remote.groups);
  const recordsByType = new Map(
    PALETTE_RECORD_TYPES.map((type) => [
      type,
      rankRecords(input.recordQuery, found.get(type)?.records ?? []),
    ]),
  );

  type Candidate = { item: PaletteItem; score: number };
  const candidates: Candidate[] = [];
  for (const type of PALETTE_RECORD_TYPES) {
    const best = recordsByType.get(type)?.[0];
    if (best?.match) {
      candidates.push({
        item: recordItem(best.record, { match: best.match }),
        score: best.score,
      });
    }
  }
  const topPage = ranking.pages[0];
  if (topPage) {
    candidates.push({
      item: pageItem(topPage.entry, { match: topPage.match }),
      score: topPage.match.score,
    });
  }
  const topCommand = ranking.commands[0];
  if (topCommand) {
    candidates.push({
      item: commandItem(topCommand.entry, topCommand.match),
      score: topCommand.match.score,
    });
  }
  const top = candidates.reduce<Candidate | null>(
    (best, candidate) => (best === null || candidate.score > best.score ? candidate : best),
    null,
  );
  const topHit = top && top.score >= TOP_HIT_THRESHOLD ? top.item : null;
  if (topHit) {
    sections.push(section("top-hit", t("Top hit"), [topHit]));
  }
  const notTop = (item: PaletteItem) => item.key !== topHit?.key;
  const countWithTop = (items: number, kind: (item: PaletteItem) => boolean) =>
    items + (topHit && kind(topHit) ? 1 : 0);

  const status = remoteStatusSections(input.remote, t);
  if (status) {
    sections.push(...status);
  } else {
    for (const type of PALETTE_RECORD_TYPES) {
      const items = (recordsByType.get(type) ?? [])
        .map(({ record, match }) => recordItem(record, { match: match ?? undefined }))
        .filter(notTop);
      const total = Math.max(found.get(type)?.total ?? 0, items.length);
      const count = countWithTop(
        total,
        (item) => item.kind === "record" && item.record.entityType === type,
      );
      sections.push(
        section(
          `records:${type}`,
          paletteEntityText(type, t).plural,
          items.slice(0, RECORD_SECTION_CAP),
          {
            count,
            seeAll: total > RECORD_SECTION_CAP ? type : undefined,
          },
        ),
      );
    }
  }

  const pageItems = ranking.pages
    .map(({ entry, match }) => pageItem(entry, { match }))
    .filter(notTop);
  sections.push(
    section("pages", t("Pages"), pageItems.slice(0, PAGE_SECTION_CAP), {
      count: countWithTop(pageItems.length, (item) => item.kind === "page"),
      seeAll: pageItems.length > PAGE_SECTION_CAP ? "pages" : undefined,
    }),
  );

  const commandItems = ranking.commands
    .map(({ entry, match }) => commandItem(entry, match))
    .filter(notTop);
  sections.push(
    section("commands", t("Commands"), commandItems.slice(0, COMMAND_SECTION_CAP), {
      count: countWithTop(commandItems.length, (item) => item.kind === "command"),
      seeAll: commandItems.length > COMMAND_SECTION_CAP ? "commands" : undefined,
    }),
  );

  const matchedSomething =
    topHit !== null ||
    pageItems.length > 0 ||
    commandItems.length > 0 ||
    [...recordsByType.values()].some((records) => records.length > 0);
  const settled = !input.remote.loading && !input.remote.error;
  const empty = settled && !matchedSomething && input.question === null;

  if (input.question === null && query !== "") {
    sections.push(fallbackAsk(query, empty, t));
  }

  return { sections: nonEmpty(sections), empty };
}

/**
 * Asking about one record: the questions worth asking about that kind of
 * record before anything is typed, and once one is answered the ones left
 * under "Ask something else". A typed question is the only row.
 */
export function buildSubjectSections(
  subject: AssistantEntityRef,
  query: string,
  answered: string | null,
  t: TranslateFn,
): PaletteSection[] {
  const typed = query.trim();
  if (typed !== "") {
    return answered === typed
      ? []
      : [section("ask", t("Ask Nova about {0}", subject.label), [askItem(typed, subject)])];
  }
  const questions = subjectQuestions(subject.type, t).filter((question) => question !== answered);
  return nonEmpty([
    section(
      "ask",
      answered === null ? t("Ask Nova about {0}", subject.label) : t("Ask something else"),
      questions.map((question) => askItem(question, subject)),
    ),
  ]);
}

export type ScopeCountMode = "search" | "commands";

/**
 * What each scope tab counts for the query. A record type's count is what the
 * answer on screen found: absent until the first answer lands, then held while
 * the next search is in flight, so the strip does not reflow on every key. Command
 * mode searched only commands, so the tabs it skipped read zero rather than a
 * count it never ran. A question searches records by its search words, so its
 * tabs count what that found.
 */
export function scopeCounts(
  ranking: LocalRanking,
  remote: RemoteState,
  mode: ScopeCountMode = "search",
): Partial<Record<PaletteScope, number>> {
  const counts: Partial<Record<PaletteScope, number>> = {
    pages: mode === "commands" ? 0 : ranking.pages.length,
    commands: ranking.commands.length,
  };
  let all = (counts.pages ?? 0) + ranking.commands.length;
  if (mode === "commands") {
    for (const type of PALETTE_RECORD_TYPES) {
      counts[type] = 0;
    }
  } else if (remote.ready && !remote.loading && !remote.error) {
    const found = recordsFromGroups(remote.groups);
    for (const type of PALETTE_RECORD_TYPES) {
      const total = found.get(type)?.total ?? 0;
      counts[type] = total;
      all += total;
    }
  }
  counts.all = all;
  return counts;
}

/** Every item in the order it is drawn, for keyboard shortcuts and the preview. */
export function flattenSections(sections: readonly PaletteSection[]): PaletteItem[] {
  return sections.flatMap((entry) => entry.items);
}
