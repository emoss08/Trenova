import { documentContentEndpoint, documentPageImageEndpoint } from "@/services/document";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import { api } from "@trenova/shared/lib/api";

const PAGE_IMAGE = "image/webp";

/**
 * An image the API drew of a page, fetched with the session's credentials and held as a blob.
 * Anything that does not say it is an image is typed as one, so a mislabelled response is
 * only ever handed to an <img>, which draws nothing from it.
 */
async function pageImage(endpoint: string, signal: AbortSignal): Promise<Blob> {
  const { blob } = await api.download(endpoint, "page.webp", { signal });
  return blob.type.startsWith("image/") ? blob : new Blob([blob], { type: PAGE_IMAGE });
}

export const documentPage = createQueryKeys("documentPage", {
  image: (documentId: string, page: number) => ({
    queryKey: [documentId, page],
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      pageImage(documentPageImageEndpoint(documentId, page), signal),
  }),
  original: (documentId: string) => ({
    queryKey: [documentId, "original"],
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      pageImage(documentContentEndpoint(documentId, "view"), signal),
  }),
  preview: (documentId: string) => ({
    queryKey: [documentId, "preview"],
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      pageImage(documentContentEndpoint(documentId, "preview"), signal),
  }),
});
