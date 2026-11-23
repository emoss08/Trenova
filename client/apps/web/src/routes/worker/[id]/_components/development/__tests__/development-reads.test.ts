import { PENDING_DOCUMENTS_REFETCH_MS } from "@/lib/resource-documents";
import type { Query } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { fetchWorkerTrainingRecords } from "@/lib/graphql/worker-training";
import { trainingRecordsList, trainingSummaryQuery } from "../development-reads";

vi.mock("@/lib/graphql/worker-training", () => ({
  fetchWorkerTrainingRecords: vi.fn(),
  fetchWorkerTrainingSummary: vi.fn(),
  WORKER_TRAINING_KEY: "worker-training",
  WORKER_TRAINING_SUMMARY_KEY: "worker-training-summary",
}));

vi.mock("@/lib/graphql/performance-review", () => ({
  fetchWorkerPerformanceReviews: vi.fn(),
  WORKER_REVIEWS_KEY: "worker-reviews",
}));

const certificate = (previewStatus: string) => ({
  id: "doc_hazmat",
  fileName: "doc_hazmat.pdf",
  originalName: "hazmat-certificate.pdf",
  fileType: "application/pdf",
  fileSize: 2048,
  previewStatus,
  createdAt: 1,
});

/** The refetch interval a read's options give for the data it holds. */
function intervalOf(options: { refetchInterval?: unknown }, data: unknown) {
  const interval = options.refetchInterval as (query: Query) => number | false;
  return interval({ state: { data } } as unknown as Query);
}

describe("training reads", () => {
  it("read the records a page at a time, open and closed, under the training key", async () => {
    const list = trainingRecordsList("wrk_1");
    const signal = new AbortController().signal;

    expect(list.root).toBe("worker-training");
    await list.fetchPage({ first: 50, after: null, signal });
    await list.fetchPage({ first: 50, after: "c_50", signal });

    expect(vi.mocked(fetchWorkerTrainingRecords).mock.calls).toEqual([
      [{ workerId: "wrk_1", includeClosed: true, first: 50, after: undefined }, { signal }],
      [{ workerId: "wrk_1", includeClosed: true, first: 50, after: "c_50" }, { signal }],
    ]);
  });

  it("read the summary again while a slot's certificate preview is drawn", () => {
    const options = trainingSummaryQuery("wrk_1");

    expect(
      intervalOf(options, {
        items: [{ record: null }, { record: { document: certificate("Pending") } }],
      }),
    ).toBe(PENDING_DOCUMENTS_REFETCH_MS);
    expect(
      intervalOf(options, { items: [{ record: { document: certificate("Unsupported") } }] }),
    ).toBe(false);
  });
});
