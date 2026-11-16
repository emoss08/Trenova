/**
 * The plan page in the customer's Trenova account with one add-on already switched on, ready to
 * confirm: the account's plan page with `?add=<add-on key>`. Any query the page already carries is
 * kept. Null when the address cannot be read, so no broken link is drawn.
 */
export function addOnUrl(manageUrl: string, addOnKey: string): string | null {
  try {
    const url = new URL(manageUrl);
    url.searchParams.set("add", addOnKey);
    return url.toString();
  } catch {
    return null;
  }
}
