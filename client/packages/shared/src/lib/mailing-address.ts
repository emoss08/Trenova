export type MailingAddress = {
  line1?: string | null;
  line2?: string | null;
  city?: string | null;
  /** The state as an envelope writes it: its abbreviation. */
  state?: string | null;
  postalCode?: string | null;
  country?: string | null;
};

function clean(value: string | null | undefined): string {
  return value?.trim() ?? "";
}

/**
 * A city with its state and postal code the way an envelope reads, "Dallas, TX 75201", leaving out
 * whatever is blank. The same rule as the server's `addressutils.CityStatePostal`, so a preview
 * prints the line the document will.
 */
export function cityStatePostal(
  city: string | null | undefined,
  state: string | null | undefined,
  postalCode: string | null | undefined,
): string {
  const left = clean(city);
  const right = [clean(state), clean(postalCode)].filter(Boolean).join(" ");
  if (!left) return right;
  if (!right) return left;
  return `${left}, ${right}`;
}

/** An address as the lines a printed document stacks, blanks left out (`addressutils.Mailing.Lines`). */
export function mailingAddressLines(address: MailingAddress): string[] {
  return [
    clean(address.line1),
    clean(address.line2),
    cityStatePostal(address.city, address.state, address.postalCode),
    clean(address.country),
  ].filter(Boolean);
}
