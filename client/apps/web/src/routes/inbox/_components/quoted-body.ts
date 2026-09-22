/**
 * A message body split the way a mail client shows it: what this sender
 * wrote, and the thread they replied on top of, folded away until asked for.
 *
 * The fold starts at a reply line ("On … wrote:"), a forwarded or original
 * message header, or a trailing run of ">" lines. A quote the sender answers
 * line by line is their message, so it stays. The boundaries are the same
 * ones the server's list preview stops at.
 */
export type QuotedBody = { own: string; quoted: string };

const REPLY_INTRO = /^On .{4,200} wrote:$/;
const HEADER_PREFIXES = [
  "-----Original Message-----",
  "---------- Forwarded message",
  "________________________________",
];

function isFoldStart(line: string): boolean {
  const trimmed = line.trim();

  return REPLY_INTRO.test(trimmed) || HEADER_PREFIXES.some((prefix) => trimmed.startsWith(prefix));
}

function isQuoteOrBlank(line: string): boolean {
  const trimmed = line.trim();

  return trimmed === "" || trimmed.startsWith(">");
}

function trimBlankEdges(lines: string[]): string {
  return lines
    .join("\n")
    .replace(/^\s*\n/, "")
    .trimEnd();
}

export function splitQuotedBody(body: string): QuotedBody {
  const lines = body.replace(/\r\n?/g, "\n").split("\n");

  let fold = lines.findIndex(isFoldStart);
  if (fold === -1) {
    fold = lines.length;
    while (fold > 0 && isQuoteOrBlank(lines[fold - 1])) {
      fold--;
    }
    if (!lines.slice(fold).some((line) => line.trim().startsWith(">"))) {
      fold = lines.length;
    }
  }

  const own = trimBlankEdges(lines.slice(0, fold));
  const quoted = trimBlankEdges(lines.slice(fold));
  if (own === "") {
    return { own: trimBlankEdges(lines), quoted: "" };
  }

  return { own, quoted };
}
