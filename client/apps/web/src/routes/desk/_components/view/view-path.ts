/** Where a composed view opens on the Desk's canvas: the conversation that made it and its name there. */
export function viewPath(threadId: string, slug: string): string {
  return `/desk/v/${encodeURIComponent(threadId)}/${encodeURIComponent(slug)}`;
}
