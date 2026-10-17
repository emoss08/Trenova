import { describe, expect, it } from "vitest";
import { MODEL_READING_WINDOW_SECONDS, stillBeingRead } from "../document-reading";

const extractedAt = 1_792_000_000;

function reading(
  status: "Pending" | "Extracting" | "Extracted" | "Indexed" | "Failed",
  acceptanceStatus?: "not_attempted" | "pending" | "accepted" | "rejected",
  lastExtractedAt: number | null = extractedAt,
) {
  return {
    status,
    lastExtractedAt,
    structuredData: acceptanceStatus ? { aiDiagnostics: { acceptanceStatus } } : {},
  };
}

/**
 * Mirrors documentcontent.Content.StillBeingRead: a document is still being
 * read until its text is in and the model's reading of it, if one was asked
 * for, has landed or been given up on.
 */
describe("stillBeingRead", () => {
  it("is reading while the text is not in", () => {
    expect(stillBeingRead(reading("Pending"), extractedAt)).toBe(true);
    expect(stillBeingRead(reading("Extracting"), extractedAt)).toBe(true);
  });

  it("is reading while the model is, once the text is in", () => {
    expect(stillBeingRead(reading("Indexed", "pending"), extractedAt + 60)).toBe(true);
    expect(stillBeingRead(reading("Extracted", "pending", null), extractedAt)).toBe(true);
  });

  it("is done once the model's reading lands or is never asked for", () => {
    expect(stillBeingRead(reading("Indexed", "accepted"), extractedAt + 60)).toBe(false);
    expect(stillBeingRead(reading("Indexed", "rejected"), extractedAt + 60)).toBe(false);
    expect(stillBeingRead(reading("Indexed", "not_attempted"), extractedAt + 60)).toBe(false);
    expect(stillBeingRead(reading("Indexed"), extractedAt + 60)).toBe(false);
  });

  it("gives up on a model reading still pending past the server's window", () => {
    const content = reading("Indexed", "pending");

    expect(stillBeingRead(content, extractedAt + MODEL_READING_WINDOW_SECONDS - 1)).toBe(true);
    expect(stillBeingRead(content, extractedAt + MODEL_READING_WINDOW_SECONDS)).toBe(false);
  });

  it("is done with a document that could not be read", () => {
    expect(stillBeingRead(reading("Failed", "pending"), extractedAt)).toBe(false);
  });
});
