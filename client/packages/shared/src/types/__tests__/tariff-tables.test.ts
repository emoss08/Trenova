import { describe, expect, it } from "vitest";
import { tariffCheckResultSchema, tariffContextSchema } from "../tariff";

describe("tariff context rate tables", () => {
  it("keeps a table's published version and a postal range cell's two ends", () => {
    const context = tariffContextSchema.parse({
      tables: [
        {
          code: "Lanes",
          name: "Lanes",
          version: 4,
          axes: [{ match: "Range", kind: "ZipRange", label: "Origin" }],
          valueType: "Money",
          cells: [{ at: [{ from: "100", to: "149" }], value: "12.5" }],
          loaded: [["10001"]],
        },
      ],
    });

    const [table] = context.tables;
    expect(table?.version).toBe(4);
    expect(table?.cells?.[0]?.at).toEqual([
      { key: undefined, from: "100", to: "149", min: undefined, max: undefined },
    ]);
    expect(table?.loaded).toEqual([["10001"]]);
  });

  it("reads a table with no version as unversioned", () => {
    const context = tariffContextSchema.parse({
      tables: [{ code: "Zones", axes: [], valueType: "Money", version: null }],
    });
    expect(context.tables[0]?.version).toBeUndefined();
  });
});

describe("tariff check rate tables", () => {
  const base = { ok: true, diagnostics: [], inputs: [] };

  it("lists each table the file reads with the version the context held", () => {
    expect(
      tariffCheckResultSchema.parse({
        ...base,
        tables: [
          { code: "Lanes", version: 4 },
          { code: "Zones", version: null },
        ],
      }).tables,
    ).toEqual([
      { code: "Lanes", version: 4 },
      { code: "Zones", version: null },
    ]);
  });

  it("reads a check that names no tables as reading none", () => {
    expect(tariffCheckResultSchema.parse(base).tables).toEqual([]);
  });
});
