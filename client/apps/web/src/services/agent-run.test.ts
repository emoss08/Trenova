import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ downloadFromUrl: vi.fn() }));

vi.mock("@trenova/shared/lib/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/utils")>()),
  downloadFromUrl: mocks.downloadFromUrl,
}));

import { agentRunTranscriptUrl, downloadAgentRunTranscript } from "./agent-run";

/**
 * A run's transcript downloads the way a conversation's does: a same-origin
 * link the server names, so the session goes with it and nothing is fetched
 * into the page first.
 */
describe("agentRunTranscriptUrl", () => {
  beforeEach(() => {
    mocks.downloadFromUrl.mockReset();
  });

  it("points at the run's transcript on the API", () => {
    expect(agentRunTranscriptUrl("ar_01M3034Q2N7JD99RA1D8DGH1ZF")).toMatch(
      /\/agent-runs\/ar_01M3034Q2N7JD99RA1D8DGH1ZF\/transcript\/$/u,
    );
  });

  it("escapes an id that is not a plain identifier", () => {
    expect(agentRunTranscriptUrl("a/b")).toContain("/agent-runs/a%2Fb/transcript/");
  });

  it("downloads through the shared helper without naming the file", () => {
    downloadAgentRunTranscript("ar_1");

    expect(mocks.downloadFromUrl).toHaveBeenCalledExactlyOnceWith(agentRunTranscriptUrl("ar_1"));
  });
});
