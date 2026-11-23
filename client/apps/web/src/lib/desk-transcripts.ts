import { fetchDeskTranscriptExportDownload } from "@/lib/graphql/desk-transcripts";
import { api } from "@trenova/shared/lib/api";
import { downloadBlob, downloadFromUrl } from "@trenova/shared/lib/utils";

/**
 * Downloads another person's conversation as the Markdown file its owner's own
 * download produces, headed with whose it is. The server checks the Export
 * permission and records the download.
 */
export async function downloadDeskConversation(id: string): Promise<void> {
  const file = await api.download(
    `/desk-transcripts/${encodeURIComponent(id)}/download/`,
    "conversation.md",
  );
  downloadBlob(file.blob, file.fileName);
}

/** Fetches the one-minute link to the requester's own export and downloads it. */
export async function downloadDeskTranscriptExport(exportId: string): Promise<void> {
  const file = await fetchDeskTranscriptExportDownload(exportId);
  downloadFromUrl(file.url, file.fileName);
}
