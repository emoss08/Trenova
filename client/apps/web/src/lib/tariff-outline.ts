// tariff-outline.ts reads the checker's outline the way every Studio view does: a `property` is a
// test (it runs on every change, blocks review when it fails and has a verdict dot), so whatever
// lists or counts tests takes both kinds.

import type { TariffOutlineItem } from "@trenova/shared/types/tariff";

/** Whether an outline entry is a `test` or a `property` block. */
export function isTestItem(item: Pick<TariffOutlineItem, "kind">): boolean {
  return item.kind === "test" || item.kind === "property";
}
