import { describe, expect, it } from "vitest";
import { buildJsonPathIndex } from "../audit-json-path";

/** The JSON as the audit panel shows it, and the path each line copies (absent: none). */
function pathsByLine(value: unknown): Array<[string, string | undefined]> {
  const json = JSON.stringify(value, null, 2);
  const index = buildJsonPathIndex(json);
  return json.split("\n").map((line, i) => [line.trim(), index.get(i)]);
}

describe("buildJsonPathIndex", () => {
  it("maps each value and each opened object or array to its path", () => {
    expect(pathsByLine({ id: "shp_1", stops: [{ city: "Reno" }, 2], meta: { ok: true } })).toEqual([
      ["{", undefined],
      ['"id": "shp_1",', "$.id"],
      ['"stops": [', "$.stops"],
      ["{", "$.stops[0]"],
      ['"city": "Reno"', "$.stops[0].city"],
      ["},", undefined],
      ["2", "$.stops[1]"],
      ["],", undefined],
      ['"meta": {', "$.meta"],
      ['"ok": true', "$.meta.ok"],
      ["}", undefined],
      ["}", undefined],
    ]);
  });

  it("keeps the lines after an empty object or array, which is written on one line", () => {
    expect(pathsByLine({ tags: [], extra: {}, after: 1 })).toEqual([
      ["{", undefined],
      ['"tags": [],', "$.tags"],
      ['"extra": {},', "$.extra"],
      ['"after": 1', "$.after"],
      ["}", undefined],
    ]);
  });

  it("follows the text when the serializer drops a property or rewrites a value", () => {
    const at = new Date(Date.UTC(2026, 9, 13));
    expect(pathsByLine({ gone: undefined, at, after: 1 })).toEqual([
      ["{", undefined],
      ['"at": "2026-10-13T00:00:00.000Z",', "$.at"],
      ['"after": 1', "$.after"],
      ["}", undefined],
    ]);
  });

  it("maps nothing for a lone value or text that is not JSON", () => {
    expect(buildJsonPathIndex(JSON.stringify("text")).size).toBe(0);
    expect(buildJsonPathIndex("null").size).toBe(0);
    expect(buildJsonPathIndex("{not json").size).toBe(0);
  });
});
