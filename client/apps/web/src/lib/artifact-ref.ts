/**
 * A reply names an artifact inside a sentence with a link whose address is
 * `artifact:<id>`, which the server teaches the model to write. The Desk
 * draws it as the artifact's badge; anywhere else it reads as its words.
 */
export const ARTIFACT_HREF = "artifact:";

const ARTIFACT_LINK = /\]\(artifact:([A-Za-z0-9_]+)\)/g;

/** The artifact id a link names, or null for any other link. */
export function artifactRefId(href: string | undefined): string | null {
  if (!href?.startsWith(ARTIFACT_HREF)) {
    return null;
  }
  const id = href.slice(ARTIFACT_HREF.length);
  return /^[A-Za-z0-9_]+$/.test(id) ? id : null;
}

/** The artifacts a reply names in its text, so they are not listed again under it. */
export function artifactRefIds(content: string): Set<string> {
  return new Set(Array.from(content.matchAll(ARTIFACT_LINK), (match) => match[1]));
}

/** A line that holds nothing but artifact links: a badge set apart instead of in a sentence. */
const LONE_REFS = /^\s*(?:\[[^\]\n]*\]\(artifact:[A-Za-z0-9_]+\)[\s.,;:]*)+$/u;

/** A block of text that reads as a sentence rather than a heading, list, table, quote or code. */
function isProse(block: string): boolean {
  const first = block.trimStart().split("\n", 1)[0] ?? "";
  return first !== "" && !/^(#{1,6}\s|[|>]|[-*+]\s|\d+[.)]\s|```|~~~|\$\$|\\\[)/u.test(first);
}

/**
 * A reply with every artifact it made named inside its words. An artifact is
 * opened from a sentence of the reply, never from a line under it: a badge the
 * model set on a line of its own (after a table, say) and any artifact the
 * reply did not name are moved to the end of the reply's first sentence-like
 * paragraph, which is where its answer is. A reply with no such paragraph
 * keeps them at its end.
 */
export function withArtifactRefs(
  content: string,
  artifacts: readonly { id: string; title: string }[],
): string {
  const named = artifactRefIds(content);
  const missing = artifacts
    .filter((artifact) => !named.has(artifact.id))
    .map((artifact) => `[${artifact.title.replace(/[[\]]/g, "")}](${ARTIFACT_HREF}${artifact.id})`);

  const lines = content.trimEnd().split("\n");
  const lone: string[] = [];
  let fenced = false;
  const kept = lines.filter((line) => {
    if (/^\s*(```|~~~)/u.test(line)) fenced = !fenced;
    if (!fenced && LONE_REFS.test(line)) {
      lone.push(line.trim().replace(/[\s.,;:]+$/u, ""));
      return false;
    }
    return true;
  });
  const refs = [...lone, ...missing];
  if (refs.length === 0) {
    return content;
  }

  const body = kept.join("\n").trimEnd();
  if (body === "") {
    return refs.join(" ");
  }
  // Blocks with the blank lines between them kept, so a code block reads as written.
  const parts = body.split(/(\n{2,})/u);
  let inFence = false;
  let at = -1;
  for (let index = 0; index < parts.length; index += 2) {
    const fences = parts[index].match(/^\s*(```|~~~)/gmu)?.length ?? 0;
    if (!inFence && at === -1 && isProse(parts[index])) at = index;
    if (fences % 2 === 1) inFence = !inFence;
  }
  if (at === -1) {
    return `${body}\n\n${refs.join(" ")}`;
  }
  parts[at] = `${parts[at].trimEnd()} ${refs.join(" ")}`;
  return parts.join("");
}
