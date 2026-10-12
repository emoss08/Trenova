/** A place as a list line names it: "Atlanta, GA", or whichever half is known. */
export function cityStateLabel(
  city: string | null | undefined,
  state: string | null | undefined,
): string | null {
  if (city && state) {
    return `${city}, ${state}`;
  }
  return city || state || null;
}
