/**
 * The wording several texts share, with the places they differ marked as
 * slots, so a batch of messages written from one template reads once with a
 * table of what fills each slot.
 *
 * The template is worked out from the texts alone, and every text's values
 * are checked by putting them back into the template: a text that does not
 * come out exactly as it was gets no values, and its reader is shown it
 * whole. So the template can be awkward, never wrong.
 */

export type TemplatePart = { kind: "text"; text: string } | { kind: "slot"; slot: number };

export type SharedTemplate = {
  parts: TemplatePart[];
  slots: number;
  /** Per text, in order: what fills each slot, or null when it must be shown whole. */
  values: (string[] | null)[];
};

/** Token pairs past this the comparison would cost more than reading the texts. */
const MAX_CELLS = 400_000;
/** Past this many slots a template is harder to read than the texts. */
const MAX_SLOTS = 8;
/** At least this share of a text must be the shared wording. */
const MIN_SHARED = 0.5;
/** Shared wording with fewer letters or digits than this between two slots joins them. */
const MIN_ANCHOR = 3;

const TOKEN = /[\p{L}\p{N}_]+|\s+|[^\p{L}\p{N}_\s]/gu;
const WORD_CHAR = /[\p{L}\p{N}]/gu;

function tokenize(text: string): string[] {
  return text.match(TOKEN) ?? [];
}

function wordChars(text: string): number {
  return text.match(WORD_CHAR)?.length ?? 0;
}

/** Which of `a`'s tokens a longest common subsequence with `b` keeps. */
function commonWith(a: readonly string[], b: readonly string[]): boolean[] | null {
  const n = a.length;
  const m = b.length;
  if ((n + 1) * (m + 1) > MAX_CELLS) {
    return null;
  }
  const width = m + 1;
  const lengths = new Uint16Array((n + 1) * width);
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lengths[i * width + j] =
        a[i] === b[j]
          ? lengths[(i + 1) * width + j + 1] + 1
          : Math.max(lengths[(i + 1) * width + j], lengths[i * width + j + 1]);
    }
  }

  const kept = Array.from({ length: n }, () => false);
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      kept[i] = true;
      i++;
      j++;
    } else if (lengths[(i + 1) * width + j] >= lengths[i * width + j + 1]) {
      i++;
    } else {
      j++;
    }
  }

  return kept;
}

function partsOf(tokens: readonly string[], fixed: readonly boolean[]): TemplatePart[] {
  const parts: TemplatePart[] = [];
  let slots = 0;
  for (let i = 0; i < tokens.length; i++) {
    const last = parts.at(-1);
    if (fixed[i]) {
      if (last?.kind === "text") {
        last.text += tokens[i];
      } else {
        parts.push({ kind: "text", text: tokens[i] });
      }
    } else if (last?.kind !== "slot") {
      parts.push({ kind: "slot", slot: slots++ });
    }
  }

  return parts;
}

/** Joins two slots whose shared wording between them is too thin to anchor on. */
function anchored(tokens: readonly string[], fixed: boolean[]): boolean[] {
  let start = -1;
  for (let i = 0; i <= tokens.length; i++) {
    const isFixed = i < tokens.length && fixed[i];
    if (isFixed && start < 0) {
      start = i;
    }
    if (!isFixed && start >= 0) {
      const between = start > 0 && i < tokens.length;
      if (between && wordChars(tokens.slice(start, i).join("")) < MIN_ANCHOR) {
        fixed.fill(false, start, i);
      }
      start = -1;
    }
  }

  return fixed;
}

function numberSlots(parts: TemplatePart[]): number {
  let slots = 0;
  for (const part of parts) {
    if (part.kind === "slot") {
      part.slot = slots++;
    }
  }

  return slots;
}

/** What fills each slot of `parts` in `text`, or null when the text does not come out of it. */
export function fillTemplate(parts: readonly TemplatePart[], text: string): string[] | null {
  const values: string[] = [];
  let at = 0;
  for (let k = 0; k < parts.length; k++) {
    const part = parts[k];
    if (part.kind === "text") {
      if (!text.startsWith(part.text, at)) {
        return null;
      }
      at += part.text.length;
      continue;
    }
    const next = parts[k + 1];
    if (next === undefined) {
      values.push(text.slice(at));
      at = text.length;
      continue;
    }
    const found = text.indexOf(next.kind === "text" ? next.text : "", at);
    if (found < 0) {
      return null;
    }
    values.push(text.slice(at, found));
    at = found;
  }
  if (at !== text.length) {
    return null;
  }

  return renderTemplate(parts, values) === text ? values : null;
}

export function renderTemplate(parts: readonly TemplatePart[], values: readonly string[]): string {
  return parts
    .map((part) => (part.kind === "text" ? part.text : (values[part.slot] ?? "")))
    .join("");
}

/** How many texts are tried as the one the others are compared with. */
const REFERENCES = 3;

/**
 * The template the texts share, or null when they share too little for one
 * to help: fewer than two texts, too long to compare, less than half shared,
 * too many slots, or fewer than two texts that come out of it. A text that
 * shares less than half its wording with the rest is left out of the
 * template and shown whole, so one message written differently does not
 * cost the others theirs.
 */
export function sharedTemplate(texts: readonly string[]): SharedTemplate | null {
  if (texts.length < 2) {
    return null;
  }
  for (let r = 0; r < Math.min(REFERENCES, texts.length); r++) {
    const template = templateFrom(texts, r);
    if (template !== null) {
      return template;
    }
  }

  return null;
}

function templateFrom(texts: readonly string[], referenceIndex: number): SharedTemplate | null {
  const reference = tokenize(texts[referenceIndex]);
  const referenceChars = wordChars(texts[referenceIndex]);
  if (reference.length === 0 || referenceChars === 0) {
    return null;
  }

  const fixed = Array.from({ length: reference.length }, () => true);
  let alike = 0;
  for (let t = 0; t < texts.length; t++) {
    if (t === referenceIndex) {
      continue;
    }
    const kept = commonWith(reference, tokenize(texts[t]));
    if (kept === null) {
      return null;
    }
    const keptChars = wordChars(reference.filter((_, i) => kept[i]).join(""));
    if (keptChars < MIN_SHARED * referenceChars) {
      continue;
    }
    alike++;
    for (let i = 0; i < fixed.length; i++) {
      fixed[i] = fixed[i] && kept[i];
    }
  }
  if (alike === 0) {
    return null;
  }

  const parts = partsOf(reference, anchored(reference, fixed));
  const slots = numberSlots(parts);
  const shared = parts.reduce(
    (sum, part) => sum + (part.kind === "text" ? wordChars(part.text) : 0),
    0,
  );
  if (slots > MAX_SLOTS || shared < MIN_SHARED * referenceChars) {
    return null;
  }

  const values = texts.map((text) => fillTemplate(parts, text));
  if (values.filter((filled) => filled !== null).length < 2) {
    return null;
  }

  return { parts, slots, values };
}
