/** A content digest short enough to compare by eye: its algorithm and the first twelve characters. */
export function shortDigest(digest: string): string {
  const trimmed = digest.trim();
  const at = trimmed.indexOf(":");
  if (at < 0) {
    return trimmed.slice(0, 12);
  }
  return `${trimmed.slice(0, at)}:${trimmed.slice(at + 1, at + 13)}`;
}
