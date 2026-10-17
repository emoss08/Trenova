import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { extractionCardsByDocument } from "../attachment-cards";

type Card = Pick<AssistantArtifact, "id" | "kind" | "payload" | "lineageSeq" | "updatedAt">;

function card(overrides: Partial<Card>): Card {
  return {
    id: "art_1",
    kind: "extraction",
    payload: { documentId: "doc_1" },
    lineageSeq: 1,
    updatedAt: 1_800_000_000,
    ...overrides,
  };
}

describe("extractionCardsByDocument", () => {
  it("names the newest version of each document's extraction", () => {
    const cards = extractionCardsByDocument([
      card({ id: "art_rules", lineageSeq: 1 }),
      card({ id: "art_model", lineageSeq: 2 }),
      card({ id: "art_other", payload: { documentId: "doc_2" } }),
    ]);

    expect(cards.get("doc_1")).toBe("art_model");
    expect(cards.get("doc_2")).toBe("art_other");
    expect(cards.size).toBe(2);
  });

  it("breaks a tie between versions by the later update", () => {
    const cards = extractionCardsByDocument([
      card({ id: "art_late", updatedAt: 1_800_000_500 }),
      card({ id: "art_early", updatedAt: 1_800_000_100 }),
    ]);

    expect(cards.get("doc_1")).toBe("art_late");
  });

  it("leaves out other kinds and cards that name no document", () => {
    const cards = extractionCardsByDocument([
      card({ id: "art_table", kind: "table_view" }),
      card({ id: "art_blank", payload: { documentId: "" } }),
      card({ id: "art_missing", payload: {} }),
      card({ id: "art_number", payload: { documentId: 7 } }),
    ]);

    expect(cards.size).toBe(0);
  });
});
