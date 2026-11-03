import { documentContentEndpoint, documentPageImageEndpoint } from "@/services/document";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import { api } from "@trenova/shared/lib/api";

const PAGE_IMAGE = "image/webp";

/**
 * How a drawn page is cached: a page never changes under its key, and one no viewer shows is
 * dropped after five minutes.
 */
export const PAGE_IMAGE_CACHE = {
  staleTime: Number.POSITIVE_INFINITY,
  gcTime: 5 * 60 * 1000,
} as const;

/**
 * An image the API drew of a page, fetched with the session's credentials and held as a blob.
 * Anything that does not say it is an image is typed as one, so a mislabelled response is
 * only ever handed to an <img>, which draws nothing from it.
 */
export async function pageImage(endpoint: string, signal: AbortSignal): Promise<Blob> {
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

type PageImageQuery = {
  queryKey: readonly unknown[];
  queryFn: (context: { signal: AbortSignal }) => Promise<Blob>;
};

/**
 * The queries that draw one page of a file a viewer shows, and where the whole file downloads
 * from. A source with no thumbnail of its own shows its pages' full drawings in the rail.
 */
export type PageImageSource = {
  image: (page: number) => PageImageQuery;
  thumbnail?: (page: number) => PageImageQuery;
  downloadEndpoint: string;
};

/** A stored document's pages, as the server keeps them drawn. */
export function storedDocumentPages(documentId: string): PageImageSource {
  return {
    image: (page) => documentPage.image(documentId, page),
    downloadEndpoint: documentContentEndpoint(documentId, "download"),
  };
}
