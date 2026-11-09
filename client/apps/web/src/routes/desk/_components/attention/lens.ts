import type { DeskAttentionGroup, DeskAttentionItem, DeskLens } from "@/lib/graphql/desk-attention";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** The lenses in the order the page shows them; the keys 1–4 pick them in this order. */
export const DESK_LENSES: readonly DeskLens[] = ["Everything", "Watchtower", "Triage", "Decisions"];

const PARAM: Record<DeskLens, string> = {
  Everything: "",
  Watchtower: "watchtower",
  Triage: "triage",
  Decisions: "decisions",
};

/** The lens a `?lens=` address names; anything else is Everything. */
export function lensFromParam(value: string | null): DeskLens {
  const found = DESK_LENSES.find((lens) => PARAM[lens] !== "" && PARAM[lens] === value);
  return found ?? "Everything";
}

export function lensParam(lens: DeskLens): string {
  return PARAM[lens];
}

export function lensLabel(t: TranslateFn, lens: DeskLens): string {
  switch (lens) {
    case "Watchtower":
      return t("Watchtower");
    case "Triage":
      return t("Triage");
    case "Decisions":
      return t("Decisions");
    default:
      return t("Everything");
  }
}

/** The lenses a row has signals in, each once, in page order. */
export function groupLenses(group: DeskAttentionGroup): DeskLens[] {
  const present = new Set(group.items.map((item) => item.deskLens));
  return DESK_LENSES.filter((lens) => lens !== "Everything" && present.has(lens));
}

/** What the row is called: the record it is about, else what its lead item says it is. */
export function groupName(t: TranslateFn, group: DeskAttentionGroup): string {
  const record = group.record;
  if (record?.readable && record.label !== "") {
    return record.subtitle !== "" ? t("{0} · {1}", record.label, record.subtitle) : record.label;
  }
  if (record) {
    return t("A record you can't open");
  }
  const lead = group.items[0];
  return lead?.record?.label || lead?.kindLabel || "";
}

/** The item the row leads with: the soonest deadline, which the server sorted first. */
export function leadItem(group: DeskAttentionGroup): DeskAttentionItem {
  return group.items[0];
}

/** Every item the page holds, row by row, in the order J and K walk them. */
export function attentionItems(groups: readonly DeskAttentionGroup[]): DeskAttentionItem[] {
  return groups.flatMap((group) => group.items);
}
