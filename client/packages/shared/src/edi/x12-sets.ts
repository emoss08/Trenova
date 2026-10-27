import { x12Separators } from "@trenova/shared/lib/highlight/x12";

/**
 * The transaction sets an X12 text holds (`204`, `210`), in the order their ST segments come,
 * each once, read with the interchange's own separators.
 */
export function x12TransactionSets(text: string): string[] {
  const { element, segment } = x12Separators(text);
  const sets: string[] = [];
  for (const raw of text.split(segment)) {
    const fields = raw.trim().split(element);
    if (fields[0] !== "ST") continue;
    const set = fields[1]?.trim() ?? "";
    if (set !== "" && !sets.includes(set)) sets.push(set);
  }
  return sets;
}
