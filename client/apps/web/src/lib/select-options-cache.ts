import type { SelectOptionResource } from "@trenova/graphql/generated/graphql";
import type { QueryKey } from "@tanstack/react-query";

function keyMentionsResource(key: QueryKey, resource: SelectOptionResource): boolean {
  return key.some(
    (part) =>
      part === resource ||
      (typeof part === "object" &&
        part !== null &&
        (part as { resource?: unknown }).resource === resource),
  );
}

/**
 * Matches every cached query that reads a resource's select options — the
 * autocomplete's own search and selected-value lookups as well as
 * `useSelectOption`. Pass it to `invalidateQueries` after a mutation that
 * changes which options exist, so the pickers do not keep offering a row that
 * was just archived.
 *
 * It matches on the resource appearing anywhere in the key because the three
 * query keys involved put it in three different positions.
 */
export function selectOptionsQueryFilter(resource: SelectOptionResource) {
  return {
    predicate: ({ queryKey }: { queryKey: QueryKey }) => {
      const scope = queryKey[0];
      if (
        scope !== "autocomplete-search" &&
        scope !== "autocomplete-option" &&
        scope !== "select-option"
      ) {
        return false;
      }

      return keyMentionsResource(queryKey, resource);
    },
  };
}
