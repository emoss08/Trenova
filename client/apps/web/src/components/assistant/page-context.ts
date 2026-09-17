/**
 * What the person is looking at, in the shape the server accepts.
 *
 * The kind names mirror `KnownPageEntityTypes` on the server; a kind the
 * server does not list is rejected with a field error, so an unknown page
 * sends its path and nothing else.
 */
export type PageContext = {
  path: string;
  entityType: string;
  entityId: string;
  title: string;
};

const MAX_PATH_LENGTH = 500;
const MAX_TITLE_LENGTH = 200;

/** Route prefixes and the record kind their panels open, longest prefix first. */
const PAGE_ENTITY_ROUTES: readonly { prefix: string; entityType: string }[] = [
  { prefix: "/shipment-management/shipments", entityType: "shipment" },
  { prefix: "/shipment-management/orders", entityType: "order" },
  { prefix: "/shipment-management/recurring-shipments", entityType: "shipment" },
  { prefix: "/hr/workers", entityType: "worker" },
  { prefix: "/equipment/tractors", entityType: "tractor" },
  { prefix: "/equipment/trailers", entityType: "trailer" },
  { prefix: "/dispatch/carriers", entityType: "carrier" },
  { prefix: "/dispatch/locations", entityType: "location" },
  { prefix: "/dispatch/console", entityType: "shipment_move" },
  { prefix: "/billing/queue", entityType: "billing_queue_item" },
  { prefix: "/billing/invoices", entityType: "invoice" },
  { prefix: "/billing/rate-matrices", entityType: "rate_matrix" },
  { prefix: "/customers", entityType: "customer" },
  { prefix: "/documents", entityType: "document" },
  { prefix: "/admin/agent-control", entityType: "agent_definition" },
];

const ENTITY_ID_PARAMS = ["panelEntityId", "entityId"] as const;

export function stripAppTitle(title: string): string {
  const trimmed = title.trim();
  const separator = trimmed.lastIndexOf(" | ");
  const stripped = separator === -1 ? trimmed : trimmed.slice(0, separator).trim();

  return stripped.slice(0, MAX_TITLE_LENGTH);
}

function entityTypeFor(pathname: string): string {
  const match = PAGE_ENTITY_ROUTES.find(
    (route) => pathname === route.prefix || pathname.startsWith(route.prefix + "/"),
  );

  return match?.entityType ?? "";
}

function entityIdFrom(search: string): string {
  const params = new URLSearchParams(search);
  for (const key of ENTITY_ID_PARAMS) {
    const value = params.get(key)?.trim();
    if (value) {
      return value.slice(0, 100);
    }
  }

  return "";
}

export function derivePageContext(location: {
  pathname: string;
  search: string;
  title: string;
}): PageContext | null {
  const pathname = location.pathname.trim();
  if (!pathname.startsWith("/")) {
    return null;
  }

  const entityType = entityTypeFor(pathname);
  const entityId = entityType === "" ? "" : entityIdFrom(location.search);

  let path = pathname + location.search;
  if (path.length > MAX_PATH_LENGTH) {
    path =
      path.length > pathname.length
        ? pathname.slice(0, MAX_PATH_LENGTH)
        : path.slice(0, MAX_PATH_LENGTH);
  }

  return { path, entityType, entityId, title: stripAppTitle(location.title) };
}
