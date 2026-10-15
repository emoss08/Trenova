/**
 * A record's internal id as Trenova writes them: a short lowercase prefix,
 * an underscore and 26 characters of Crockford base32, such as
 * `inv_01M42BNHACKS99T13QKY88TVXV`. Ids are for tool calls; a person never
 * reads one. One inside a link's address — an artifact link, a URL — is
 * left alone.
 */
const ID = String.raw`(?<![\w:/.=?&#-])[a-z]{2,8}_[0-9A-HJKMNP-TV-Z]{26}(?!\w)`;
/** The marks a model wraps an id in: code ticks, bold. */
const WRAP = "[`*_]*";
/** What a model calls an id beside it: "ID", "id:", "invoiceId", "shipment ID". */
const LABEL = String.raw`(?:[A-Za-z]+\s)?[A-Za-z]*(?:ID|Id|id)\b:?\s*`;

/** "(ID **inv_…**)", "(`shp_…`)", "(invoiceId inv_…)": the whole aside goes. */
const ASIDE = new RegExp(String.raw`\s*\(\s*(?:${LABEL})?${WRAP}${ID}${WRAP}\s*\)`, "gu");
/** "ID **inv_…**" or ", id: ap_…" in a sentence: the label goes with it. */
const LABELLED = new RegExp(String.raw`,?\s*\b${LABEL}${WRAP}${ID}${WRAP}`, "gu");
/** Any id left on its own, with the comma or space before it. */
const BARE = new RegExp(String.raw`,?\s*${WRAP}${ID}${WRAP}`, "gu");
/**
 * A sentence, or a clause joined on with "and", that says only what an id is:
 * "Its ID is edip_….", "Acme Retail Group's ID: edip_…", ", and its ID is
 * edip_…". Without the id it says nothing, so it goes whole rather than
 * leaving "Its ID is." behind.
 */
const STATEMENT = new RegExp(
  String.raw`(,?[ \t]+and[ \t]+|[ \t]*)\b` +
    String.raw`(?:[Ii]ts|[Tt]he|[Tt]heir|[Tt]his|[Tt]hat|[Yy]our|[Oo]ur|[A-Z]\w*(?:[ \t]+[A-Z]\w*){0,3}['’]s)` +
    String.raw`(?:[ \t]+[\w'’]+){0,2}[ \t]+(?:[A-Za-z]*(?:ID|Id)|id)` +
    String.raw`(?:[ \t]+(?:is|was)[ \t]+|[ \t]*:[ \t]*)` +
    String.raw`${WRAP}${ID}${WRAP}([.!?]?)`,
  "gu",
);

function endsSentence(text: string, end: number, stop: string): boolean {
  const after = text.slice(end);
  const rest = after.replace(/^[ \t]+/u, "");
  if (stop) {
    return end === text.length || rest !== after || after.startsWith("\n");
  }

  return rest === "" || rest.startsWith("\n");
}

function startsSentence(text: string, start: number): boolean {
  const before = text.slice(0, start).replace(/[ \t]+$/u, "");

  return before === "" || /[.!?\n]$/u.test(before);
}

/**
 * Each statement that ends its sentence goes: a whole sentence with the space
 * after it, or a clause joined on with "and", whose sentence keeps its stop.
 */
function withoutStatements(text: string): string {
  let out = "";
  let last = 0;
  for (const match of text.matchAll(STATEMENT)) {
    const start = match.index;
    let end = start + match[0].length;
    const [, lead, stop] = match;
    if (!endsSentence(text, end, stop)) {
      continue;
    }
    if (lead.includes("and")) {
      end -= stop.length;
    } else if (!startsSentence(text, start)) {
      continue;
    } else if (start === 0 || text[start - 1] === "\n") {
      end += text.slice(end).length - text.slice(end).replace(/^[ \t]+/u, "").length;
      if (text[end] === "\n") {
        end += 1;
      }
    }
    out += text.slice(last, start);
    last = end;
  }

  return out + text.slice(last);
}

/**
 * The text with every internal record id taken out, and the words around
 * each closed up. The agents are told never to show one; this keeps one
 * that slips through, or one a summary carries for the agent, off the page.
 */
export function withoutRecordIds(text: string): string {
  if (!/[a-z]_[0-9A-HJKMNP-TV-Z]{26}/u.test(text)) {
    return text;
  }

  return withoutStatements(text)
    .replace(ASIDE, "")
    .replace(LABELLED, "")
    .replace(BARE, "")
    .replace(/[ \t]+([,.;:!?)])/gu, "$1")
    .replace(/\(\s*\)/gu, "")
    .replace(/[ \t]{2,}/gu, " ");
}
