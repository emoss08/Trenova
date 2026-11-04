import { apiService } from "@/services/api";
import { api } from "@trenova/shared/lib/api";
import type { Document } from "@trenova/shared/types/document";
import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  documentPreviewQuery,
  prefetchRecordPreview,
} from "../_components/preview/preview-queries";
import { record } from "./palette-test-fixtures";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("prefetchRecordPreview", () => {
  it("warms a document's record but never draws its page, which the server logs as a view", async () => {
    const getById = vi.spyOn(apiService.documentService, "getById").mockResolvedValue({
      id: "doc_1",
      fileType: "application/pdf",
      originalName: "Rate Confirmation.pdf",
    } as Document);
    const download = vi.spyOn(api, "download");
    const queryClient = new QueryClient();

    prefetchRecordPreview(
      queryClient,
      record({ entityType: "document", id: "doc_1", title: "Rate Confirmation.pdf" }),
    );
    await vi.waitFor(() =>
      expect(queryClient.getQueryData(documentPreviewQuery("doc_1").queryKey)).toBeDefined(),
    );
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(getById).toHaveBeenCalledWith("doc_1");
    expect(download).not.toHaveBeenCalled();
    expect(queryClient.getQueryCache().findAll({ queryKey: ["documentPage"] })).toEqual([]);
  });
});
