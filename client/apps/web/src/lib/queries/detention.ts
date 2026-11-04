import { fetchDeskDetention } from "@/lib/graphql/detention-card";
import {
  fetchDetentionIntelligence,
  type DetentionIntelligenceInput,
} from "@/lib/graphql/detention-intelligence";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const detention = createQueryKeys("detention", {
  desk: () => ({
    queryKey: ["desk"],
    queryFn: async ({ signal }) => apiService.detentionService.desk({ signal }),
  }),
  byShipment: (shipmentId: string) => ({
    queryKey: ["byShipment", shipmentId],
    queryFn: async ({ signal }) => apiService.detentionService.byShipment(shipmentId, { signal }),
  }),
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchDeskDetention(id, { signal }),
  }),
  occurrence: (id: string) => ({
    queryKey: ["occurrence", id],
    queryFn: async ({ signal }) => apiService.detentionService.getOccurrence(id, { signal }),
  }),
  intelligence: (input: DetentionIntelligenceInput) => ({
    queryKey: ["intelligence", input.days, input.through ?? null, input.locationId ?? null],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDetentionIntelligence(input, { signal }),
  }),
  disputePacket: (id: string) => ({
    queryKey: ["disputePacket", id],
    queryFn: async ({ signal }) => apiService.detentionService.disputePacket(id, { signal }),
  }),
  policyById: (id: string) => ({
    queryKey: ["policyById", id],
    queryFn: async ({ signal }) => apiService.detentionPolicyService.getById(id, { signal }),
  }),
});
