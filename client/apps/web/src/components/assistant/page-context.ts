import { recordAtLocation } from "@/config/record-links";

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

/**
 * Older links name the open record in `entityId`; bookmarks and sent links
 * still carry it after every in-app link moved to the registry's parameters.
 */
const LEGACY_ENTITY_ID_PARAM = "entityId";
const MAX_ENTITY_ID_LENGTH = 100;

export function stripAppTitle(title: string): string {
  const trimmed = title.trim();
  const separator = trimmed.lastIndexOf(" | ");
  const stripped = separator === -1 ? trimmed : trimmed.slice(0, separator).trim();

  return stripped.slice(0, MAX_TITLE_LENGTH);
}

function recordAt(pathname: string, search: string): { entityType: string; entityId: string } {
  const record = recordAtLocation(pathname, search);
  if (record === null) {
    return { entityType: "", entityId: "" };
  }

  const entityId =
    record.entityId || (new URLSearchParams(search).get(LEGACY_ENTITY_ID_PARAM)?.trim() ?? "");
  return { entityType: record.entityType, entityId: entityId.slice(0, MAX_ENTITY_ID_LENGTH) };
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

  const { entityType, entityId } = recordAt(pathname, location.search);

  let path = pathname + location.search;
  if (path.length > MAX_PATH_LENGTH) {
    path =
      path.length > pathname.length
        ? pathname.slice(0, MAX_PATH_LENGTH)
        : path.slice(0, MAX_PATH_LENGTH);
  }

  return { path, entityType, entityId, title: stripAppTitle(location.title) };
}
