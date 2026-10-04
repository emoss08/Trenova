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
 * The text with every internal record id taken out, and the words around
 * each closed up. The agents are told never to show one; this keeps one
 * that slips through, or one a summary carries for the agent, off the page.
 */
export function withoutRecordIds(text: string): string {
  if (!/[a-z]_[0-9A-HJKMNP-TV-Z]{26}/u.test(text)) {
    return text;
  }

  return text
    .replace(ASIDE, "")
    .replace(LABELLED, "")
    .replace(BARE, "")
    .replace(/[ \t]+([,.;:!?)])/gu, "$1")
    .replace(/\(\s*\)/gu, "")
    .replace(/[ \t]{2,}/gu, " ");
}
