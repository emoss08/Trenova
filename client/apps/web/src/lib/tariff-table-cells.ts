// tariff-table-cells.ts keeps the rate table cells the browser has loaded for Tariff tests, and
// lays them into the context the module reads (docs/engineering/rules-engine.md, "Rate tables and
// the on-demand protocol"). The organisation's context carries only the shapes of tables with an
// exact axis; `tariff.test` answers the key combinations a test read that the context did not
// carry, the Studio loads those cells from the server and runs the tests again. The cells are
// kept per rate table stamp, so publishing or archiving a matrix, which moves the stamp, drops
// them. The context holds each table's newest published version; the stamp names those versions.

import type {
  TariffContext,
  TariffNeed,
  TariffTableCell,
  TariffTableCells,
} from "@trenova/shared/types/tariff";

/**
 * How many times one source's tests load cells, as the server's test runs do: three loads of the
 * combinations they need, then the still-needed tables whole.
 */
export const TABLE_CELL_PASSES = 4;

type CachedTable = {
  cells: readonly TariffTableCell[];
  /** The exact-key combinations `cells` holds, by `comboKey`; null when it is the whole table. */
  loaded: ReadonlyMap<string, string[]> | null;
};

/** The cells loaded at one stamp, by table code. */
export type TableCellCache = {
  readonly stamp: string;
  readonly tables: ReadonlyMap<string, CachedTable>;
};

/** One combination of exact-axis keys as one string. */
export function comboKey(keys: readonly string[]): string {
  return keys.join("\u001f");
}

export function emptyTableCellCache(stamp: string): TableCellCache {
  return { stamp, tables: new Map() };
}

/**
 * The cache with the server's answer added. An answer read at another stamp starts a new cache:
 * cells from two stamps are never mixed. A table that came whole replaces what was kept of it.
 */
export function mergeTableCells(cache: TableCellCache, answer: TariffTableCells): TableCellCache {
  const base = answer.stamp === cache.stamp ? cache : emptyTableCellCache(answer.stamp);
  const tables = new Map(base.tables);
  for (const table of answer.tables) {
    const kept = tables.get(table.code);
    if (table.loaded === undefined) {
      tables.set(table.code, { cells: table.cells, loaded: null });
      continue;
    }
    if (kept?.loaded === null) continue;
    const fresh = table.loaded.filter((keys) => !kept?.loaded?.has(comboKey(keys)));
    if (fresh.length === 0) continue;
    const loaded = new Map(kept?.loaded ?? []);
    for (const keys of fresh) loaded.set(comboKey(keys), [...keys]);
    tables.set(table.code, { cells: [...(kept?.cells ?? []), ...table.cells], loaded });
  }
  return { stamp: base.stamp, tables };
}

/** The needs the cache cannot answer: those not loaded yet, each once. */
export function uncachedNeeds(cache: TableCellCache, needs: readonly TariffNeed[]): TariffNeed[] {
  const seen = new Set<string>();
  const out: TariffNeed[] = [];
  for (const need of needs) {
    const kept = cache.tables.get(need.table);
    const key = `${need.table}\u001e${comboKey(need.keys)}`;
    if (kept?.loaded === null || kept?.loaded.has(comboKey(need.keys)) || seen.has(key)) continue;
    seen.add(key);
    out.push(need);
  }
  return out;
}

/**
 * The context with the cached cells laid into its tables. A table the context already carries
 * whole (one with no exact axis) is left as it is; one it carries by combination gets the
 * cached combinations and their cells, or the whole table when that was loaded.
 */
export function withTableCells(context: TariffContext, cache: TableCellCache): TariffContext {
  if (cache.tables.size === 0) return context;
  return {
    ...context,
    tables: context.tables.map((table) => {
      const kept = cache.tables.get(table.code);
      if (!kept || table.loaded === undefined) return table;
      if (kept.loaded === null) return { ...table, cells: [...kept.cells], loaded: undefined };
      return {
        ...table,
        cells: [...(table.cells ?? []), ...kept.cells],
        loaded: [...table.loaded, ...kept.loaded.values()],
      };
    }),
  };
}
