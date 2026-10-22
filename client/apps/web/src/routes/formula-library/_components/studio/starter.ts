// starter.ts holds the library a new one starts from, and how a library's header relates to the
// path templates import it by.

import { FORMULA_LIBRARY_NAME_PATTERN } from "@trenova/shared/types/formula-library";

/** The path a library is imported by: `org.` and its name. */
export function libraryPath(name: string): string {
  return `org.${name}`;
}

/**
 * A small library that checks and passes its test as written, named for the library: a
 * function with a default, and a test that pins what it returns.
 */
export function starterLibrary(name: string): string {
  const path = libraryPath(FORMULA_LIBRARY_NAME_PATTERN.test(name) ? name : "example");
  return `library "${path}"

/// A flat fee, unless the caller sets another.
fn flatFee(fee: Money = $150) -> Money {
  return fee
}

test "the flat fee defaults to $150" {
  expect flatFee() == $150
  expect flatFee(fee: $90) == $90
}
`;
}

/**
 * Whether the header names another module than the one templates import, as `org.<name>`:
 * the header is what the editor and the outline show, the record's name is the import path.
 */
export function headerMismatch(headerName: string | null | undefined, name: string): boolean {
  const trimmed = name.trim();
  if (!headerName || !FORMULA_LIBRARY_NAME_PATTERN.test(trimmed)) return false;
  return headerName !== libraryPath(trimmed);
}
