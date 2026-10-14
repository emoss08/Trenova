/** Text made safe to place inside a RegExp source, where it matches itself literally. */
export function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
