const EMAIL_ADDRESS_PATTERN = /^[^\s@,;<>]+@[^\s@,;<>]+\.[^\s@,;<>]+$/;

/**
 * Reads a free-form recipient field the way the server does: addresses
 * separated by commas, semicolons, new lines or tabs, trimmed, lower-cased and
 * with repeats dropped.
 */
export function splitEmailList(raw: string | null | undefined): string[] {
  if (!raw) {
    return [];
  }
  const seen = new Set<string>();
  const addresses: string[] = [];
  for (const part of raw.split(/[,;\n\t]/)) {
    const address = part.trim().toLowerCase();
    if (address === "" || seen.has(address)) {
      continue;
    }
    seen.add(address);
    addresses.push(address);
  }
  return addresses;
}

export function isEmailAddress(value: string): boolean {
  return EMAIL_ADDRESS_PATTERN.test(value.trim());
}

/** The addresses in a recipient field that are not email addresses. */
export function invalidEmailAddresses(raw: string | null | undefined): string[] {
  return splitEmailList(raw).filter((address) => !isEmailAddress(address));
}
