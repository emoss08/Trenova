import type { CaptureBatchDetail, EditCaptureItemsInput } from "@/lib/graphql/capture";

/** Where the pages a person left out of every document collect. */
export const LOOSE = "loose";

/**
 * One proposed document while a person rearranges a stack: its pages in order.
 * The key is the item it started as, or where it was carved from, so the
 * editor keeps its place and its form while pages move around it.
 */
export type LayoutGroup = { key: string; pageIds: string[] };

/**
 * How a stack's open pages divide into documents, as the person is editing it.
 *
 * Only open documents are here. A filed document is fixed on the server and
 * is drawn beside the editor, never inside it, so nothing here can move one of
 * its pages. `loose` holds the pages in no document — separators, blank pages
 * and whatever the person set aside — in scan order, because that is the order
 * someone looks for a page they dropped by mistake.
 */
export type PageLayout = {
  groups: LayoutGroup[];
  loose: string[];
  rotations: Readonly<Record<string, number>>;
  /** Each page's place in the stack, for ordering what is set aside. */
  sequence: Readonly<Record<string, number>>;
};

function openItemStatus(status: CaptureBatchDetail["items"][number]["status"]): boolean {
  return status === "Proposed" || status === "Failed";
}

/** The layout a batch is in now, as the server last saw it. */
export function layoutFromBatch(batch: CaptureBatchDetail): PageLayout {
  const sequence: Record<string, number> = {};
  const rotations: Record<string, number> = {};
  for (const page of batch.pages) {
    sequence[page.id] = page.sequence;
    rotations[page.id] = normalizeRotation(page.rotation);
  }

  const placed = new Set<string>();
  const groups: LayoutGroup[] = [];
  for (const item of [...batch.items].sort((a, b) => a.position - b.position)) {
    if (item.status === "Discarded") {
      continue;
    }
    for (const id of item.pageIds) {
      placed.add(id);
    }
    if (openItemStatus(item.status)) {
      groups.push({ key: item.id, pageIds: [...item.pageIds] });
    }
  }

  const loose = batch.pages
    .filter((page) => !placed.has(page.id))
    .sort((a, b) => a.sequence - b.sequence)
    .map((page) => page.id);

  return { groups, loose, rotations, sequence };
}

export function normalizeRotation(degrees: number): number {
  return ((degrees % 360) + 360) % 360;
}

function bySequence(layout: PageLayout) {
  return (a: string, b: string) => (layout.sequence[a] ?? 0) - (layout.sequence[b] ?? 0);
}

/** Drops documents left with no pages. A document is its pages. */
function compact(layout: PageLayout, groups: LayoutGroup[]): PageLayout {
  return { ...layout, groups: groups.filter((group) => group.pageIds.length > 0) };
}

function without(layout: PageLayout, pageId: string): PageLayout {
  return {
    ...layout,
    groups: layout.groups.map((group) => ({
      ...group,
      pageIds: group.pageIds.filter((id) => id !== pageId),
    })),
    loose: layout.loose.filter((id) => id !== pageId),
  };
}

/**
 * Starts a new document after the given page: it and the pages before it stay,
 * the pages after it become the next document. Splitting after a document's
 * last page does nothing, since there is nothing to split off.
 */
export function splitAfter(layout: PageLayout, groupKey: string, pageId: string): PageLayout {
  const groups: LayoutGroup[] = [];
  for (const group of layout.groups) {
    const at = group.pageIds.indexOf(pageId);
    if (group.key !== groupKey || at < 0 || at === group.pageIds.length - 1) {
      groups.push(group);
      continue;
    }
    groups.push({ key: group.key, pageIds: group.pageIds.slice(0, at + 1) });
    groups.push({
      key: `${group.key}:${group.pageIds[at + 1]}`,
      pageIds: group.pageIds.slice(at + 1),
    });
  }

  return { ...layout, groups };
}

/** Joins a document with the one after it, the way two stapled stacks are one. */
export function mergeWithNext(layout: PageLayout, groupKey: string): PageLayout {
  const at = layout.groups.findIndex((group) => group.key === groupKey);
  if (at < 0 || at === layout.groups.length - 1) {
    return layout;
  }

  const groups = [...layout.groups];
  const [first, second] = [groups[at]!, groups[at + 1]!];
  groups.splice(at, 2, { key: first.key, pageIds: [...first.pageIds, ...second.pageIds] });

  return { ...layout, groups };
}

/**
 * Puts a page at a place in a document, or among the pages set aside. Moving a
 * page out of the document it was in leaves that document without it, and a
 * document left empty is gone.
 */
export function movePage(
  layout: PageLayout,
  pageId: string,
  target: string,
  index: number,
): PageLayout {
  const known =
    layout.loose.includes(pageId) || layout.groups.some((group) => group.pageIds.includes(pageId));
  if (!known) {
    return layout;
  }

  const rest = without(layout, pageId);
  if (target === LOOSE) {
    return {
      ...compact(rest, rest.groups),
      loose: [...rest.loose, pageId].sort(bySequence(layout)),
    };
  }

  const exists = rest.groups.some((group) => group.key === target);
  if (!exists) {
    return layout;
  }

  const groups = rest.groups.map((group) => {
    if (group.key !== target) {
      return group;
    }
    const pageIds = [...group.pageIds];
    pageIds.splice(Math.max(0, Math.min(index, pageIds.length)), 0, pageId);
    return { ...group, pageIds };
  });

  return compact(rest, groups);
}

/** Sets a page aside: it goes into no document and stays on the batch. */
export function leaveOut(layout: PageLayout, pageId: string): PageLayout {
  return movePage(layout, pageId, LOOSE, 0);
}

/** Makes a document of one page that was set aside, at the end of the stack. */
export function newDocumentFrom(layout: PageLayout, pageId: string): PageLayout {
  if (!layout.loose.includes(pageId)) {
    return layout;
  }

  return {
    ...layout,
    loose: layout.loose.filter((id) => id !== pageId),
    groups: [...layout.groups, { key: `new:${pageId}`, pageIds: [pageId] }],
  };
}

/** Turns a page a quarter turn clockwise, or back with a negative turn. */
export function rotate(layout: PageLayout, pageId: string, quarterTurns: number): PageLayout {
  const current = layout.rotations[pageId];
  if (current === undefined) {
    return layout;
  }

  return {
    ...layout,
    rotations: { ...layout.rotations, [pageId]: normalizeRotation(current + quarterTurns * 90) },
  };
}

/** Whether the person changed anything the server would need to hear about. */
export function isDirty(layout: PageLayout, saved: PageLayout): boolean {
  if (layout.groups.length !== saved.groups.length) {
    return true;
  }
  for (let i = 0; i < layout.groups.length; i++) {
    const a = layout.groups[i]!.pageIds;
    const b = saved.groups[i]!.pageIds;
    if (a.length !== b.length || a.some((id, j) => id !== b[j])) {
      return true;
    }
  }

  return Object.keys(layout.rotations).some((id) => layout.rotations[id] !== saved.rotations[id]);
}

/**
 * The edit the server applies: every open document's pages in order, and the
 * rotation of every page whose rotation changed. Pages in no document are
 * simply not named, which is how the server knows they were set aside.
 */
export function toEditInput(
  layout: PageLayout,
  saved: PageLayout,
  version: number,
): EditCaptureItemsInput {
  const rotations = Object.keys(layout.rotations)
    .filter((pageId) => layout.rotations[pageId] !== saved.rotations[pageId])
    .map((pageId) => ({ pageId, rotation: layout.rotations[pageId]! }));

  return {
    version,
    items: layout.groups
      .filter((group) => group.pageIds.length > 0)
      .map((group) => ({ pageIds: [...group.pageIds] })),
    rotations,
  };
}

/**
 * Where a dragged page lands, from what it was dropped on: a page (it takes
 * that page's place) or a document's empty space (it goes at the end). Pages
 * set aside keep scan order, so a drop anywhere among them is the same drop.
 */
export function dropPosition(
  layout: PageLayout,
  overId: string,
): { target: string; index: number } | null {
  if (overId === LOOSE || layout.loose.includes(overId)) {
    return { target: LOOSE, index: 0 };
  }
  for (const group of layout.groups) {
    if (group.key === overId) {
      return { target: group.key, index: group.pageIds.length };
    }
    const index = group.pageIds.indexOf(overId);
    if (index >= 0) {
      return { target: group.key, index };
    }
  }

  return null;
}
