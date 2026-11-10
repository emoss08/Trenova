import type { DeskAttentionGroup } from "@/lib/graphql/desk-attention";

/** Where a record's case opens on the Desk. */
export function casePath(kind: string, id: string): string {
  return `/desk/r/${encodeURIComponent(kind)}/${encodeURIComponent(id)}`;
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
