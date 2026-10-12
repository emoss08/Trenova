import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";

/**
 * Which of a card's sections the person may read, by the resource each one
 * reads. A card opens for anyone who may read its own record; a section built
 * from another resource, such as a customer's receivables or its rate
 * agreements, is left out for someone who may not read that resource, and its
 * query is never sent. The card itself is never refused for it.
 *
 * Pass a module-level constant, so the answer is worked out again only when
 * the person's permissions change.
 */
export function useCardAccess<K extends string>(
  sections: Readonly<Record<K, string>>,
): Readonly<Record<K, boolean>> {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const manifest = usePermissionStore((state) => state.manifest);

  return useMemo(() => {
    const access = {} as Record<K, boolean>;
    for (const key of Object.keys(sections) as K[]) {
      access[key] = canRead(manifest !== null, hasPermission, sections[key]);
    }
    return access;
  }, [hasPermission, manifest, sections]);
}

/** Whether the person may read one resource; false for none, and until their permissions are known. */
export function useCanRead(resource: string | null): boolean {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const known = usePermissionStore((state) => state.manifest !== null);
  return resource !== null && canRead(known, hasPermission, resource);
}

function canRead(
  known: boolean,
  hasPermission: (resource: string, operation: number) => boolean,
  resource: string,
): boolean {
  return known && hasPermission(resource, Operation.Read);
}

/**
 * The documents rail every card with documents ends in reads documents, so it
 * shows only to someone who may read them, on the card and in its outline.
 */
export const CARD_DOCUMENT_SECTIONS = { documents: Resource.Document } as const;
