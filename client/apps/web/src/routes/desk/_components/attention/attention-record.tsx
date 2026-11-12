import type { DeskAttentionGroup } from "@/lib/graphql/desk-attention";

/** Where a record's case opens on the Desk. */
export function casePath(kind: string, id: string): string {
  return `/desk/r/${encodeURIComponent(kind)}/${encodeURIComponent(id)}`;
}

/**
 * What the Watchtower sends with a person it takes to a case: they came to
 * work it, so a case no agent has looked at is handed over as it opens. A
 * link from anywhere else (the rail, a card, a pasted address) only reads it.
 */
export const START_CASE = { startCase: true } as const;

/** True when the case was opened to be worked (router state from `START_CASE`). */
export function startsCase(state: unknown): boolean {
  return (
    typeof state === "object" &&
    state !== null &&
    (state as { startCase?: unknown }).startCase === true
  );
}

/** Where one item opens in Needs you, chosen in the list with its detail beside it. */
export function needsItemPath(itemId: string): string {
  return `/desk/needs?item=${encodeURIComponent(itemId)}`;
}

/**
 * Where a row of Needs you opens: the case of the record it is about, when
 * the reader may open that record, else the row's lead item in Needs you.
 */
export function attentionPath(group: DeskAttentionGroup): string {
  const record = group.record;
  if (record?.readable && record.label !== "") {
    return casePath(record.kind, record.id);
  }
  return needsItemPath(group.items[0]?.id ?? "");
}
