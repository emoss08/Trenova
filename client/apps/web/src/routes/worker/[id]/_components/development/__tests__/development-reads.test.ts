import { PENDING_DOCUMENTS_REFETCH_MS } from "@/lib/resource-documents";
import type { Query } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { trainingRecordsQuery, trainingSummaryQuery } from "../development-reads";

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
  it("read the records again while a certificate's preview is drawn, and stop once it is ready", () => {
    const options = trainingRecordsQuery("wrk_1");

    expect(intervalOf(options, [{ document: certificate("Pending") }, { document: null }])).toBe(
      PENDING_DOCUMENTS_REFETCH_MS,
    );
    expect(intervalOf(options, [{ document: certificate("Ready") }, { document: null }])).toBe(
      false,
    );
    expect(intervalOf(options, undefined)).toBe(false);
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
