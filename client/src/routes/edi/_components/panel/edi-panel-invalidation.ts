import { queries } from "@/lib/queries";
import type { QueryClient } from "@tanstack/react-query";

export async function invalidateEDIConnections(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: queries.edi.connections._def }),
    queryClient.invalidateQueries({ queryKey: queries.edi.partners._def }),
    queryClient.invalidateQueries({ queryKey: queries.edi.communicationProfiles._def }),
    queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] }),
  ]);
}

export async function invalidateEDIPartners(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: queries.edi.partners._def }),
    queryClient.invalidateQueries({ queryKey: queries.edi.partnerOptions._def }),
    queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] }),
  ]);
}
