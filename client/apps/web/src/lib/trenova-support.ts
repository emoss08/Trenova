const PAGE_PATTERN = /^\/[^\s?#\\]{0,253}$/;

/**
 * The page a support request is opened from, as the console takes it: the matched route's
 * pattern (`/shipments/$shipmentId`), never the address itself, so no record's id goes to
 * Trenova. An index route's trailing slash is dropped; anything the console would refuse is
 * left out.
 */
export function supportRequestPage(routePath: string | undefined): string | undefined {
  if (!routePath) {
    return undefined;
  }
  const page = routePath.length > 1 ? routePath.replace(/\/+$/, "") : routePath;
  return PAGE_PATTERN.test(page) ? page : undefined;
}

/**
 * The console's new request form for this instance, with the page the person was on added as
 * `page`. The form's address comes from the control plane, with the instance in it already.
 */
export function newSupportRequestUrl(formUrl: string, page: string | undefined): string {
  const url = new URL(formUrl);
  if (page) {
    url.searchParams.set("page", page);
  }
  return url.toString();
}
