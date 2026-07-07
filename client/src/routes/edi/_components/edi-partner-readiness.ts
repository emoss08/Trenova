import { EdiPartnerReadinessDocument } from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import type { ResultOf } from "@graphql-typed-document-node/core";

export type EDIPartnerReadinessState = ResultOf<
  typeof EdiPartnerReadinessDocument
>["ediPartnerReadiness"][number];

const BATCH_WINDOW_MS = 25;

type PendingRequest = {
  resolve: (state: EDIPartnerReadinessState | null) => void;
  reject: (error: unknown) => void;
};

let pendingIds = new Map<string, PendingRequest[]>();
let flushTimer: ReturnType<typeof setTimeout> | null = null;

async function flushBatch() {
  const batch = pendingIds;
  pendingIds = new Map();
  flushTimer = null;
  const partnerIds = Array.from(batch.keys());
  try {
    const result = await requestGraphQL({
      document: EdiPartnerReadinessDocument,
      operationName: "EdiPartnerReadiness",
      variables: { partnerIds },
    });
    const byId = new Map(result.ediPartnerReadiness.map((state) => [state.partnerId, state]));
    for (const [partnerId, requests] of batch) {
      const state = byId.get(partnerId) ?? null;
      for (const request of requests) request.resolve(state);
    }
  } catch (error) {
    for (const requests of batch.values()) {
      for (const request of requests) request.reject(error);
    }
  }
}

function fetchPartnerReadinessBatched(partnerId: string) {
  return new Promise<EDIPartnerReadinessState | null>((resolve, reject) => {
    const existing = pendingIds.get(partnerId) ?? [];
    existing.push({ resolve, reject });
    pendingIds.set(partnerId, existing);
    flushTimer ??= setTimeout(() => {
      void flushBatch();
    }, BATCH_WINDOW_MS);
  });
}

export function partnerReadinessQueryOptions(partnerId: string) {
  return {
    queryKey: ["edi-partner-readiness", partnerId] as const,
    queryFn: () => fetchPartnerReadinessBatched(partnerId),
    staleTime: 60_000,
  };
}
