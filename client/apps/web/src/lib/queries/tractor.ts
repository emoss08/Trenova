import {
  fetchDeskTractor,
  fetchTractorInspections,
  fetchTractorPosition,
  type InspectionCursor,
} from "@/lib/graphql/tractor-card";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import { infiniteQueryOptions } from "@tanstack/react-query";

export const tractor = createQueryKeys("tractor", {
  card: (tractorId: string) => ({
    queryKey: ["card", tractorId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskTractor(tractorId, { signal }),
  }),
  position: (tractorId: string) => ({
    queryKey: ["position", tractorId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchTractorPosition(tractorId, { signal }),
  }),
  inspections: (tractorId: string) => ({
    queryKey: ["inspections", tractorId],
  }),
});

/** A tractor's inspections, newest first, a page at a time. */
export function tractorInspectionsQuery(tractorId: string, pageSize: number) {
  return infiniteQueryOptions({
    queryKey: [...tractor.inspections(tractorId).queryKey, pageSize],
    queryFn: ({ pageParam, signal }) =>
      fetchTractorInspections(tractorId, { size: pageSize, cursor: pageParam }, { signal }),
    initialPageParam: null as InspectionCursor | null,
    getNextPageParam: (last) => last.next ?? undefined,
  });
}
