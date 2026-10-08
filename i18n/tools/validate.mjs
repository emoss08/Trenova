// validate.mjs checks a translation against its English source before it enters a catalog.
//
// A translation that drops a placeholder renders "{0}" or loses the number; one that renames
// a plural keyword falls back to the raw count; one that loses a rich-text tag renders the
// sentence without its link or bold. Each is invisible until a person reads that screen in
// that language, so merge refuses them up front.

const PLACEHOLDER = /\{(\d+)\}/g;
const PLURAL_HEAD = /\{(\d+),\s*(plural|select|selectordinal),/g;
const TAG = /<\/?([A-Za-z][\w-]*)\s*\/?>/g;

function counts(text, pattern, pick = (m) => m[1]) {
  const out = new Map();
  for (const match of text.matchAll(pattern)) {
    const key = pick(match);
    out.set(key, (out.get(key) ?? 0) + 1);
  }
  return out;
}

function sameCounts(a, b) {
  if (a.size !== b.size) return false;
  for (const [key, n] of a) if (b.get(key) !== n) return false;
  return true;
}

function describe(map) {
  return [...map.entries()].map(([k, n]) => (n > 1 ? `${k}×${n}` : k)).join(", ") || "none";
}

/**
 * translationProblems lists what a translation broke, or an empty array when it is sound.
 * Tags are only compared when the source has them, so a "<" in prose is never a tag.
 */
export function translationProblems(source, translated) {
  const problems = [];
  if (typeof translated !== "string" || translated.trim() === "") return ["is empty"];

  const wantPlaceholders = counts(source, PLACEHOLDER, (m) => m[0]);
  const gotPlaceholders = counts(translated, PLACEHOLDER, (m) => m[0]);
  if (!sameCounts(wantPlaceholders, gotPlaceholders)) {
    problems.push(`placeholders ${describe(gotPlaceholders)}, source has ${describe(wantPlaceholders)}`);
  }

  const wantPlurals = counts(source, PLURAL_HEAD, (m) => `{${m[1]}, ${m[2]}}`);
  const gotPlurals = counts(translated, PLURAL_HEAD, (m) => `{${m[1]}, ${m[2]}}`);
  if (!sameCounts(wantPlurals, gotPlurals)) {
    problems.push(`plural ${describe(gotPlurals)}, source has ${describe(wantPlurals)}`);
  }

  const wantTags = counts(source, TAG, (m) => m[0].replace(/\s+/g, ""));
  if (wantTags.size > 0) {
    const gotTags = counts(translated, TAG, (m) => m[0].replace(/\s+/g, ""));
    if (!sameCounts(wantTags, gotTags)) {
      problems.push(`tags ${describe(gotTags)}, source has ${describe(wantTags)}`);
    }
  }

  let depth = 0;
  for (const char of translated) {
    if (char === "{") depth += 1;
    else if (char === "}" && --depth < 0) break;
  }
  if (depth !== 0) problems.push("unbalanced braces");

  return problems;
}
