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

/**
 * A reply with every artifact it made named in it. An artifact is opened from
 * the words of the reply, never from a list under it, so one the reply did
 * not name (an older reply, or a model that ignored the instruction) is
 * named at the end of its last sentence. After a table the badge starts a
 * line of its own, since a table row cannot hold it.
 */
export function withArtifactRefs(
  content: string,
  artifacts: readonly { id: string; title: string }[],
): string {
  const named = artifactRefIds(content);
  const refs = artifacts
    .filter((artifact) => !named.has(artifact.id))
    .map((artifact) => `[${artifact.title.replace(/[[\]]/g, "")}](${ARTIFACT_HREF}${artifact.id})`);
  if (refs.length === 0) {
    return content;
  }

  const body = content.trimEnd();
  if (body === "") {
    return refs.join(" ");
  }
  const lastLine = body.slice(body.lastIndexOf("\n") + 1).trimStart();
  const ownLine = lastLine.startsWith("|") || lastLine.startsWith("```");
  return `${body}${ownLine ? "\n\n" : " "}${refs.join(" ")}`;
}
