/** How many models the editor lists at once. */
export const MODEL_PAGE_SIZE = 10;

/** A model this far behind the newest one the endpoint lists is no longer among its latest. */
export const LATEST_WINDOW_SECONDS = 365 * 86_400;

export type ModelGroup = "latest" | "older";

export type ModelLike = { id: string; displayName: string; createdAt?: number | null };

export type ModelSplit<T> = {
  /** False when no model carries a date, and every model is under `latest`. */
  split: boolean;
  latest: T[];
  older: T[];
};

export type ModelPage<T> = {
  items: T[];
  /** Zero-based, clamped to the pages there are. */
  page: number;
  pageCount: number;
  /** One-based position of the first and last model shown; both zero for an empty list. */
  from: number;
  to: number;
  total: number;
};

function dateOf(model: ModelLike): number | null {
  return typeof model.createdAt === "number" && model.createdAt > 0 ? model.createdAt : null;
}

function byNewest(a: ModelLike, b: ModelLike): number {
  const left = dateOf(a);
  const right = dateOf(b);
  if (left !== null && right !== null && left !== right) return right - left;
  if (left !== null && right === null) return -1;
  if (left === null && right !== null) return 1;
  return a.id.localeCompare(b.id);
}

/**
 * The endpoint's models as latest and older: latest is every model released within a
 * year of the newest one listed, so an endpoint that stopped updating still has a latest.
 * A model without a date is older; a list with no dates at all is not split.
 */
export function splitModels<T extends ModelLike>(options: readonly T[]): ModelSplit<T> {
  const sorted = [...options].sort(byNewest);
  const newest = sorted.length ? dateOf(sorted[0]!) : null;
  if (newest === null) {
    return { split: false, latest: sorted, older: [] };
  }
  const latest: T[] = [];
  const older: T[] = [];
  for (const option of sorted) {
    const created = dateOf(option);
    (created !== null && newest - created <= LATEST_WINDOW_SECONDS ? latest : older).push(option);
  }
  return { split: true, latest, older };
}

/** The models whose id or display name holds the search, ignoring case. */
export function searchModels<T extends ModelLike>(options: readonly T[], query: string): T[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return [...options];
  return options.filter(
    (option) =>
      option.id.toLowerCase().includes(needle) || option.displayName.toLowerCase().includes(needle),
  );
}

/** One page of the models, the page clamped to those there are. */
export function modelPage<T>(options: readonly T[], page: number): ModelPage<T> {
  const total = options.length;
  const pageCount = Math.max(1, Math.ceil(total / MODEL_PAGE_SIZE));
  const current = Math.min(Math.max(0, Math.trunc(page)), pageCount - 1);
  const start = current * MODEL_PAGE_SIZE;
  const items = options.slice(start, start + MODEL_PAGE_SIZE);
  return {
    items,
    page: current,
    pageCount,
    from: items.length ? start + 1 : 0,
    to: start + items.length,
    total,
  };
}

/** Where the list opens: on the group and page that show the selected model. */
export function initialModelView<T extends ModelLike>(
  split: ModelSplit<T>,
  selected: string,
): { group: ModelGroup; page: number } {
  const id = selected.trim();
  for (const group of ["latest", "older"] as const) {
    const index = split[group].findIndex((option) => option.id === id);
    if (id && index >= 0) {
      return { group, page: Math.floor(index / MODEL_PAGE_SIZE) };
    }
  }
  return { group: "latest", page: 0 };
}
