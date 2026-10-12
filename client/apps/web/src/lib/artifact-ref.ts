/**
 * A reply names an artifact inside a sentence with a link whose address is
 * `artifact:<id>`. The Desk draws it as the artifact's badge, named by the
 * artifact's own title; anywhere else it reads as its words.
 */
export const ARTIFACT_HREF = "artifact:";

/** The artifact id a link names, or null for any other link. */
export function artifactRefId(href: string | undefined): string | null {
  if (!href?.startsWith(ARTIFACT_HREF)) {
    return null;
  }
  const id = href.slice(ARTIFACT_HREF.length);
  return /^[A-Za-z0-9_]+$/.test(id) ? id : null;
}
