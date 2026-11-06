import persistedDocuments from "@trenova/graphql/generated/persisted-documents.json";
import { WorkerTableDocument } from "@trenova/graphql/generated/graphql";
import { workerSchema } from "@trenova/shared/types/worker";
import { describe, expect, it } from "vitest";

const RELATIONS = new Set(["state", "fleetCode", "profile", "pto", "customFields"]);

function workerTableSdl(): string {
  const hash = WorkerTableDocument.__meta__?.hash;
  const sdl = hash ? (persistedDocuments as Record<string, string>)[hash] : undefined;
  if (!sdl) throw new Error("WorkerTable is not in the persisted safelist");
  return sdl;
}

describe("the worker roster row", () => {
  it("selects every field the edit form saves back, so a save never clears one", () => {
    const sdl = workerTableSdl();
    const missing = Object.keys(workerSchema.shape).filter(
      (field) => !RELATIONS.has(field) && !new RegExp(`\\b${field}\\b`).test(sdl),
    );

    expect(missing).toEqual([]);
  });
});
