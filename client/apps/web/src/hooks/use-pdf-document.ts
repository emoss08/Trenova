import { api } from "@trenova/shared/lib/api";
import { useQuery } from "@tanstack/react-query";

const PDF = "application/pdf";

/**
 * A PDF the API serves, fetched with the session's credentials like every other request and
 * held as a blob the browser's own viewer can open. The blob is always typed as a PDF,
 * whatever the response said: a blob URL is the app's own origin, and a file mislabelled as
 * HTML must never be rendered as a page here.
 */
export function usePdfDocument(endpoint: string | null) {
  return useQuery({
    queryKey: ["pdf-document", endpoint],
    queryFn: async ({ signal }) => {
      const { blob } = await api.download(endpoint as string, "document.pdf", { signal });
      return blob.type === PDF ? blob : new Blob([blob], { type: PDF });
    },
    enabled: endpoint !== null,
    staleTime: 5 * 60 * 1000,
    gcTime: 60 * 1000,
  });
}
