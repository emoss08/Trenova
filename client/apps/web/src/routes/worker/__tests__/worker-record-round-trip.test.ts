import persistedDocuments from "@trenova/graphql/generated/persisted-documents.json";
import { WorkerRecordDocument } from "@trenova/graphql/generated/documents/worker/record";
import { workerProfileSchema, workerSchema } from "@trenova/shared/types/worker";
import { describe, expect, it } from "vitest";
import { SERVER_OWNED_PROFILE_FIELDS } from "../[id]/_components/profile/worker-form-values";

const WORKER_RELATIONS = new Set(["state", "fleetCode", "profile", "pto", "customFields"]);
const PROFILE_RELATIONS = new Set(["licenseState"]);
/** The server keeps these on every update and never takes them from the request. */
const SERVER_OWNED = new Set<string>(SERVER_OWNED_PROFILE_FIELDS);

function workerRecordSdl(): string {
  const hash = WorkerRecordDocument.__meta__?.hash;
  const sdl = hash ? (persistedDocuments as Record<string, string>)[hash] : undefined;
  if (!sdl) throw new Error("WorkerRecord is not in the persisted safelist");
  return sdl;
}

function selectionOf(sdl: string, field: string): string {
  const start = new RegExp(`\\b${field}\\s*\\{`).exec(sdl);
  if (!start) throw new Error(`WorkerRecord selects no ${field}`);
  let depth = 0;
  for (let index = start.index + start[0].length - 1; index < sdl.length; index++) {
    if (sdl[index] === "{") depth++;
    if (sdl[index] === "}") depth--;
    if (depth === 0) return sdl.slice(start.index + start[0].length, index);
  }
  throw new Error(`WorkerRecord's ${field} selection is not closed`);
}

function withoutNested(selection: string): string {
  let depth = 0;
  let flat = "";
  for (const char of selection) {
    if (char === "{") depth++;
    if (depth === 0) flat += char;
    if (char === "}") depth--;
  }
  return flat;
}

function unselected(keys: string[], relations: Set<string>, selection: string): string[] {
  return keys.filter(
    (key) => !relations.has(key) && !new RegExp(`(^|\\s)${key}(\\s|$)`).test(selection),
  );
}

describe("the worker record read", () => {
  it("selects every worker field the profile form saves back, so a save never clears one", () => {
    const sdl = workerRecordSdl();
    const worker = withoutNested(selectionOf(sdl, "worker\\(id: \\$id\\)"));

    expect(unselected(Object.keys(workerSchema.shape), WORKER_RELATIONS, worker)).toEqual([]);
  });

  it("selects every profile field the profile form saves back, so a save never clears one", () => {
    const sdl = workerRecordSdl();
    const profile = withoutNested(selectionOf(sdl, "profile"));

    const saved = Object.keys(workerProfileSchema.shape).filter((key) => !SERVER_OWNED.has(key));

    expect(unselected(saved, PROFILE_RELATIONS, profile)).toEqual([]);
  });

  it("selects the relations the form shows by name", () => {
    const sdl = workerRecordSdl();

    expect(withoutNested(selectionOf(sdl, "state"))).toMatch(/\babbreviation\b/);
    expect(withoutNested(selectionOf(sdl, "fleetCode"))).toMatch(/\bcode\b/);
    expect(withoutNested(selectionOf(sdl, "licenseState"))).toMatch(/\babbreviation\b/);
  });
});
