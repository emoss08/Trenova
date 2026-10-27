import { edi } from "@/lib/queries/edi";
import type { QueryClient } from "@tanstack/react-query";

export async function invalidateEDIConnections(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: edi.connections._def }),
    queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] }),
    queryClient.invalidateQueries({ queryKey: ["edi-communication-profile-list"] }),
  ]);
}

export async function invalidateEDIPartners(queryClient: QueryClient) {
  await queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] });
}

export async function invalidateEDICommunicationProfiles(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["edi-communication-profile-list"] }),
    queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] }),
  ]);
}

export async function invalidateEDITransfers(queryClient: QueryClient, transferId?: string) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["edi-inbound-transfer-list"] }),
    queryClient.invalidateQueries({ queryKey: ["edi-outbound-transfer-list"] }),
    ...(transferId
      ? [
          queryClient.invalidateQueries({
            queryKey: edi.mappingPreview(transferId).queryKey,
          }),
        ]
      : []),
  ]);
}

export async function invalidateEDIMessages(queryClient: QueryClient, messageId?: string) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["edi-message-list"] }),
    queryClient.invalidateQueries({ queryKey: edi.messages._def }),
    ...(messageId
      ? [
          queryClient.invalidateQueries({ queryKey: edi.message(messageId).queryKey }),
          queryClient.invalidateQueries({
            queryKey: edi.messageInspection(messageId).queryKey,
          }),
        ]
      : []),
  ]);
}

export async function invalidateEDIInboundFiles(queryClient: QueryClient, fileId?: string) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["edi-inbound-file-list"] }),
    queryClient.invalidateQueries({ queryKey: ["edi-message-list"] }),
    queryClient.invalidateQueries({ queryKey: edi.messages._def }),
    ...(fileId
      ? [queryClient.invalidateQueries({ queryKey: edi.inboundFile(fileId).queryKey })]
      : []),
  ]);
}
