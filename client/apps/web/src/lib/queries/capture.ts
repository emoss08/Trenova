import {
  fetchAvailableCaptureProfiles,
  fetchCaptureAgentRelease,
  fetchCaptureBatch,
  fetchCaptureBatchCount,
  fetchCaptureBatches,
  fetchCaptureDevices,
  fetchCapturePairing,
  fetchCaptureProfiles,
  fetchCaptureRequestsForTarget,
  fetchMyCaptureAccess,
  fetchMyCaptureDevices,
  type CaptureBatchFilter,
  type CaptureBatchPage,
  type CaptureDeviceStatus,
  type CaptureProfileStatus,
} from "@/lib/graphql/capture";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type Signal = { signal?: AbortSignal };

export const capture = createQueryKeys("capture", {
  access: () => ({
    queryKey: ["access"],
    queryFn: ({ signal }: Signal) => fetchMyCaptureAccess({ signal }),
  }),
  agentRelease: () => ({
    queryKey: ["agentRelease"],
    queryFn: ({ signal }: Signal) => fetchCaptureAgentRelease({ signal }),
  }),
  // The filter is part of the key: the stacks waiting on a person and the
  // ones already filed are different questions.
  batches: (filter: Omit<CaptureBatchFilter, "after">) => ({
    queryKey: [filter],
  }),
  batchCount: (filter: Omit<CaptureBatchFilter, "after" | "first">) => ({
    queryKey: [filter],
    queryFn: ({ signal }: Signal) => fetchCaptureBatchCount(filter, { signal }),
  }),
  batch: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: Signal) => fetchCaptureBatch(id, { signal }),
  }),
  requests: (targetType: string, targetId: string) => ({
    queryKey: [targetType, targetId],
    queryFn: ({ signal }: Signal) =>
      fetchCaptureRequestsForTarget({ targetType, targetId }, { signal }),
  }),
  myDevices: (status: CaptureDeviceStatus | null) => ({
    queryKey: [status ?? "all"],
    queryFn: ({ signal }: Signal) => fetchMyCaptureDevices(status, { signal }),
  }),
  devices: (status: CaptureDeviceStatus | null, query: string) => ({
    queryKey: [status ?? "all", query],
    queryFn: ({ signal }: Signal) =>
      fetchCaptureDevices({ status, query: query === "" ? null : query }, { signal }),
  }),
  availableProfiles: () => ({
    queryKey: ["available"],
    queryFn: ({ signal }: Signal) => fetchAvailableCaptureProfiles({ signal }),
  }),
  profiles: (status: CaptureProfileStatus | null, query: string) => ({
    queryKey: [status ?? "all", query],
    queryFn: ({ signal }: Signal) =>
      fetchCaptureProfiles({ status, query: query === "" ? null : query }, { signal }),
  }),
  pairing: (userCode: string) => ({
    queryKey: [userCode],
    queryFn: ({ signal }: Signal) => fetchCapturePairing(userCode, { signal }),
  }),
});

/**
 * The intake queue, paged by arrival. Each page's end cursor carries the sort,
 * so the next page continues where the last one stopped.
 */
export function captureBatchesQuery(filter: Omit<CaptureBatchFilter, "after">) {
  return {
    queryKey: capture.batches(filter).queryKey,
    queryFn: ({ pageParam, signal }: { pageParam?: unknown; signal?: AbortSignal }) =>
      fetchCaptureBatches(
        { ...filter, after: typeof pageParam === "string" ? pageParam : null },
        { signal },
      ),
    initialPageParam: null as string | null,
    getNextPageParam: (last: CaptureBatchPage): string | null =>
      last.hasNextPage ? last.endCursor : null,
  };
}
