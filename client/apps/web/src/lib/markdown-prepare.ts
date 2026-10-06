/**
 * Readies a reply's markdown for the renderer, outside code only:
 *
 * - `\[…\]` and `\(…\)` become `$$…$$` and `$…$`, which the math parser
 *   reads, but only when they come in pairs: a lone `\[` is an escaped
 *   bracket and stays one.
 * - A dollar sign before a digit is money, not math: "$48,210.00 across six
 *   customers, $9,835.50 of it Acme" must never become an equation. It is
 *   escaped, so it reads as written.
 * - While a reply is still arriving, a `$$` block that has not closed yet is
 *   left as raw text rather than set as math that is about to change.
 */

const FENCE = /^ {0,3}(`{3,}|~{3,})/u;

/** Splits text into the parts that are code (fences and inline spans) and the parts that are not; joined, they are the text again. */
function segments(text: string): { code: boolean; text: string }[] {
  const out: { code: boolean; text: string }[] = [];
  let prose = "";
  let block = "";
  let fence: string | null = null;

  const flushProse = () => {
    if (prose === "") return;
    // Inline code spans: a run of backticks closed by the same run.
    let from = 0;
    for (const match of prose.matchAll(/(`+)[\s\S]*?\1/gu)) {
      out.push({ code: false, text: prose.slice(from, match.index) });
      out.push({ code: true, text: match[0] });
      from = match.index + match[0].length;
    }
    out.push({ code: false, text: prose.slice(from) });
    prose = "";
  };

  for (const line of text.split(/(?<=\n)/u)) {
    const marker = FENCE.exec(line);
    if (fence === null) {
      if (marker) {
        flushProse();
        fence = marker[1];
        block = line;
      } else {
        prose += line;
      }
      continue;
    }
    block += line;
    if (marker && marker[1][0] === fence[0] && marker[1].length >= fence.length) {
      out.push({ code: true, text: block });
      fence = null;
      block = "";
    }
  }
  flushProse();
  // A fence still open at the end is the code arriving so far.
  if (fence !== null) {
    out.push({ code: true, text: block });
  }

  return out;
}

function pairDelimiters(text: string, open: string, close: string, as: string): string {
  let out = "";
  let at = 0;
  for (;;) {
    const start = text.indexOf(open, at);
    if (start === -1) break;
    const end = text.indexOf(close, start + open.length);
    if (end === -1) break;
    out += text.slice(at, start) + as + text.slice(start + open.length, end) + as;
    at = end + close.length;
  }
  return out + text.slice(at);
}

/** A dollar sign before a digit, not escaped and not part of `$$`: money. */
const MONEY = /(?<![\\$])\$(?=\d)/gu;

function prepareProse(text: string, streaming: boolean, last: boolean): string {
  let next = pairDelimiters(text, "\\[", "\\]", "$$");
  next = pairDelimiters(next, "\\(", "\\)", "$");
  next = next.replace(MONEY, "\\$");
  if (streaming && last) {
    const blocks = [...next.matchAll(/(?<!\\)\$\$/gu)];
    if (blocks.length % 2 === 1) {
      const open = blocks.at(-1);
      if (open) {
        // Still arriving: the open block is shown as the raw text it is so
        // far, in its own box, and set as math once it closes.
        const raw = next.slice(open.index).replace(/`/gu, "\u02cb");
        next = `${next.slice(0, open.index)}\n\n\`\`\`${RAW_MATH_FENCE}\n${raw}\n\`\`\`\n`;
      }
    }
  }
  return next;
}

/** The code fence the renderer draws an unfinished math block in; see ai-markdown. */
const RAW_MATH_FENCE = "dk-math-raw";

export function prepareMarkdown(text: string, { streaming = false } = {}): string {
  if (!/[$\\]/u.test(text)) {
    return text;
  }

  return segments(text)
    .map((part, index, parts) =>
      part.code ? part.text : prepareProse(part.text, streaming, index === parts.length - 1),
    )
    .join("");
}
