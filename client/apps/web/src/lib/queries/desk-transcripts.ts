import {
  fetchDeskTranscriptExportPreview,
  type DeskConversationFilterInput,
} from "@/lib/graphql/desk-transcripts";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const deskTranscripts = createQueryKeys("deskTranscripts", {
  exportPreview: (filter: DeskConversationFilterInput) => ({
    queryKey: [filter],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskTranscriptExportPreview(filter, { signal }),
  }),
});
