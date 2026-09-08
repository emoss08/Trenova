import { delegationState } from "@trenova/shared/lib/org-structure";

const SECONDS_IN_DAY = 86_400;

// Structural shapes rather than the generated types, so the same maths serves
// the page and a fixture written from the schema by hand.
export type PositionLike = {
  id: string;
  code: string;
  title: string;
  status: string;
  department: string;
  isDrivingPosition: boolean;
  reportsToPositionId?: string | null;
};

export type HeadcountLike = {
  key: string;
  workers: number;
  drivers: number;
  terminated: number;
  /** Users holding the title through their membership; absent on older rows. */
  staff?: number;
};

export type PositionNode<P extends PositionLike = PositionLike> = {
  position: P;
  /** Active workers holding this position. */
  workers: number;
  drivers: number;
  terminated: number;
  /** Users holding this position through their membership in the organisation. */
  staff: number;
  /** Everyone in the position from either roster. */
  people: number;
  /** People in this position and every position under it. */
  rolledUp: number;
  depth: number;
  children: PositionNode<P>[];
};

export type PositionTree<P extends PositionLike = PositionLike> = {
  roots: PositionNode<P>[];
  /** Active workers with no position at all, who sit outside the chart. */
  unplaced: number;
};

/**
 * The org chart: positions hung from the one they report to, each carrying
 * its own headcount and the headcount beneath it. A position whose chain of
 * reports-to loops back on itself is hung from the top rather than dropped;
 * a chart that silently lost a branch would be worse than one with a loop.
 */
export function buildPositionTree<P extends PositionLike>(
  positions: readonly P[],
  byPosition: readonly HeadcountLike[],
): PositionTree<P> {
  const counts = new Map(byPosition.map((row) => [row.key, row]));
  const byId = new Map(positions.map((position) => [position.id, position]));
  const nodes = new Map<string, PositionNode<P>>();
  for (const position of positions) {
    const count = counts.get(position.id);
    const workers = count?.workers ?? 0;
    const staff = count?.staff ?? 0;
    nodes.set(position.id, {
      position,
      workers,
      drivers: count?.drivers ?? 0,
      terminated: count?.terminated ?? 0,
      staff,
      people: workers + staff,
      rolledUp: 0,
      depth: 0,
      children: [],
    });
  }

  const roots: PositionNode<P>[] = [];
  for (const node of nodes.values()) {
    const parentId = node.position.reportsToPositionId ?? null;
    if (parentId && byId.has(parentId) && parentId !== node.position.id && !loops(node, byId)) {
      nodes.get(parentId)?.children.push(node);
    } else {
      roots.push(node);
    }
  }

  const order = (a: PositionNode<P>, b: PositionNode<P>) =>
    Number(b.position.status === "Active") - Number(a.position.status === "Active") ||
    b.rolledUp - a.rolledUp ||
    a.position.title.localeCompare(b.position.title);

  const settle = (node: PositionNode<P>, depth: number): number => {
    node.depth = depth;
    let total = node.people;
    for (const child of node.children) total += settle(child, depth + 1);
    node.rolledUp = total;
    node.children.sort(order);
    return total;
  };
  for (const root of roots) settle(root, 0);
  roots.sort(order);

  return { roots, unplaced: counts.get("")?.workers ?? 0 };
}

function loops<P extends PositionLike>(node: PositionNode<P>, byId: Map<string, P>): boolean {
  const seen = new Set<string>([node.position.id]);
  let current = node.position.reportsToPositionId ?? null;
  while (current) {
    if (seen.has(current)) return true;
    seen.add(current);
    current = byId.get(current)?.reportsToPositionId ?? null;
  }
  return false;
}

/** The tree in reading order, top to bottom, for a list with indentation. */
export function flattenTree<P extends PositionLike>(
  roots: readonly PositionNode<P>[],
): PositionNode<P>[] {
  const out: PositionNode<P>[] = [];
  const walk = (node: PositionNode<P>) => {
    out.push(node);
    for (const child of node.children) walk(child);
  };
  for (const root of roots) walk(root);
  return out;
}

/** Positions still open that nobody on either roster holds. */
export function vacantPositions<P extends PositionLike>(
  positions: readonly P[],
  byPosition: readonly HeadcountLike[],
): P[] {
  const filled = new Set(
    byPosition.filter((row) => row.workers + (row.staff ?? 0) > 0).map((row) => row.key),
  );
  return positions
    .filter((position) => position.status === "Active" && !filled.has(position.id))
    .sort((a, b) => a.title.localeCompare(b.title));
}

/** Title, code or department, matched loosely enough for a half-typed title. */
export function matchesPositionSearch(
  position: Pick<PositionLike, "title" | "code" | "department">,
  query: string,
): boolean {
  const needle = query.trim().toLowerCase();
  if (!needle) return true;
  return [position.title, position.code, position.department].some((field) =>
    field.toLowerCase().includes(needle),
  );
}

/**
 * The nodes a search keeps: a match, and every position above it so the
 * match still hangs where it belongs. Siblings that do not match are dropped.
 */
export function pruneTree<P extends PositionLike>(
  roots: readonly PositionNode<P>[],
  query: string,
): PositionNode<P>[] {
  if (!query.trim()) return [...roots];
  const keep = (node: PositionNode<P>): PositionNode<P> | null => {
    const children = node.children
      .map(keep)
      .filter((child): child is PositionNode<P> => child !== null);
    if (children.length > 0 || matchesPositionSearch(node.position, query)) {
      return { ...node, children };
    }
    return null;
  };
  return roots.map(keep).filter((node): node is PositionNode<P> => node !== null);
}

/** Every position under a node, at any depth. */
export function descendantIds<P extends PositionLike>(node: PositionNode<P>): Set<string> {
  const ids = new Set<string>();
  const walk = (current: PositionNode<P>) => {
    for (const child of current.children) {
      ids.add(child.position.id);
      walk(child);
    }
  };
  walk(node);
  return ids;
}

export function findNode<P extends PositionLike>(
  roots: readonly PositionNode<P>[],
  id: string,
): PositionNode<P> | null {
  for (const node of flattenTree(roots)) {
    if (node.position.id === id) return node;
  }
  return null;
}

/**
 * Whether a position may be hung from another. It may not report to itself
 * or to anything beneath it: either would close a loop the server refuses,
 * and refusing here says so before the drop rather than after.
 */
export function canReportTo<P extends PositionLike>(
  roots: readonly PositionNode<P>[],
  positionId: string,
  parentId: string | null,
): boolean {
  if (parentId === null) return true;
  if (parentId === positionId) return false;
  const node = findNode(roots, positionId);
  if (!node) return true;
  return !descendantIds(node).has(parentId);
}

/** The positions above one, nearest first, for expanding a search match into view. */
export function ancestorIds(
  positions: readonly Pick<PositionLike, "id" | "reportsToPositionId">[],
  id: string,
): string[] {
  const byId = new Map(positions.map((position) => [position.id, position]));
  const out: string[] = [];
  const seen = new Set<string>([id]);
  let current = byId.get(id)?.reportsToPositionId ?? null;
  while (current && !seen.has(current)) {
    out.push(current);
    seen.add(current);
    current = byId.get(current)?.reportsToPositionId ?? null;
  }
  return out;
}

export type DelegationLike = {
  startsAt: number;
  endsAt?: number | null;
  revokedAt?: number | null;
};

export type CoverSummary = {
  active: number;
  scheduled: number;
  /** In force with an end date inside the next seven days. */
  endingSoon: number;
  /** In force with no end date: cover somebody has to remember to call back. */
  openEnded: number;
};

const ENDING_SOON_DAYS = 7;

export function coverSummary(delegations: readonly DelegationLike[], now: number): CoverSummary {
  const summary: CoverSummary = { active: 0, scheduled: 0, endingSoon: 0, openEnded: 0 };
  for (const delegation of delegations) {
    const state = delegationState(delegation, now);
    if (state === "scheduled") {
      summary.scheduled += 1;
    } else if (state === "active") {
      summary.active += 1;
      if (delegation.endsAt == null) summary.openEnded += 1;
      else if (delegation.endsAt <= now + ENDING_SOON_DAYS * SECONDS_IN_DAY)
        summary.endingSoon += 1;
    }
  }
  return summary;
}

export type HeadcountSummary = {
  active: number;
  drivers: number;
  nonDriving: number;
  terminated: number;
  /** Users holding a title: the front office, counted apart from the roster. */
  staff: number;
  /** Everyone with a job here, on either roster. */
  people: number;
  driverShare: number;
};

export function headcountSummary(headcount: {
  activeTotal: number;
  driverTotal: number;
  terminated: number;
  staffTotal?: number;
}): HeadcountSummary {
  const nonDriving = Math.max(0, headcount.activeTotal - headcount.driverTotal);
  const staff = headcount.staffTotal ?? 0;
  return {
    active: headcount.activeTotal,
    drivers: headcount.driverTotal,
    nonDriving,
    terminated: headcount.terminated,
    staff,
    people: headcount.activeTotal + staff,
    driverShare: headcount.activeTotal > 0 ? headcount.driverTotal / headcount.activeTotal : 0,
  };
}

/** The biggest group in a breakdown, for a headline. */
export function largestGroup<T extends { workers: number }>(rows: readonly T[]): T | null {
  return rows.reduce<T | null>(
    (best, row) => (best === null || row.workers > best.workers ? row : best),
    null,
  );
}
