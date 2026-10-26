import {
  getAvailableOperations,
  getAvailableResources,
  getPermissionBundles,
} from "@/lib/role-api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/** The registry changes only with a deploy, so a read of it holds for half an hour. */
export const PERMISSION_CATALOG_STALE_TIME = 30 * 60_000;

export const permissionCatalog = createQueryKeys("permissionCatalog", {
  resources: () => ({
    queryKey: ["resources"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => getAvailableResources({ signal }),
  }),
  operations: () => ({
    queryKey: ["operations"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => getAvailableOperations({ signal }),
  }),
  bundles: () => ({
    queryKey: ["bundles"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => getPermissionBundles({ signal }),
  }),
});
