import { escapeRegExp } from "@trenova/shared/lib/regexp";
import type { AssistantEntityRef } from "@/types/assistant";

export const CITE_HREF_PREFIX = "#palette-cite-";

export interface LinkedAnswer {
  /** The answer with each cited record's name turned into a link to its chip. */
  text: string;
  /** The record behind each link's address. */
  records: ReadonlyMap<string, AssistantEntityRef>;
}

const PROTECTED = /(`[^`\n]*`|\[[^\]\n]*\]\([^)\n]*\)|https?:\/\/\S+)/g;

/**
 * Marks where an answer names a record its tools returned. The records come
 * from the turn's tool results, never from reading the reply: the reply only
 * decides where each one is mentioned. A name is linked where it stands as
 * its own word, never inside code, a link the reply wrote, or a longer word,
 * and the longest name wins where two overlap ("Acme Foods" over "Acme").
 */
export function linkCitedRecords(
  text: string,
  records: readonly AssistantEntityRef[],
): LinkedAnswer {
  const named = records.filter((record) => record.label.trim().length >= 2);
  if (text === "" || named.length === 0) {
    return { text, records: new Map() };
  }

  const byLabel = new Map<string, AssistantEntityRef>();
  for (const record of named) {
    const label = record.label.trim();
    if (!byLabel.has(label)) {
      byLabel.set(label, record);
    }
  }
  const labels = [...byLabel.keys()].sort((a, b) => b.length - a.length);
  const pattern = new RegExp(
    `(?<![\\p{L}\\p{N}_-])(${labels.map(escapeRegExp).join("|")})(?![\\p{L}\\p{N}_-])`,
    "gu",
  );

  const hrefs = new Map<string, AssistantEntityRef>();
  const hrefFor = (record: AssistantEntityRef) => {
    const href = `${CITE_HREF_PREFIX}${record.type}-${record.id}`;
    hrefs.set(href, record);
    return href;
  };

  const linked = text
    .split(PROTECTED)
    .map((part, index) =>
      index % 2 === 1
        ? part
        : part.replace(pattern, (label: string) => {
            const record = byLabel.get(label);
            return record ? `[${label}](${hrefFor(record)})` : label;
          }),
    )
    .join("");

  return { text: linked, records: hrefs };
}
