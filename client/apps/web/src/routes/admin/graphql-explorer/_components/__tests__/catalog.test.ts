import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { OperationCatalog } from "@/types/graphql-catalog";

const CATALOG: OperationCatalog = {
  operationCount: 1,
  fragmentCount: 1,
  operations: [
    {
      name: "ShipmentTable",
      kind: "query",
      domain: "shipment",
      sourceFile: "src/operations/shipment/table.graphql",
      hash: "sha256:abc",
      rootFields: ["shipments"],
      variables: [],
      fragments: [],
      usages: [],
      sdl: "query ShipmentTable { shipments { id } }",
    },
  ],
  fragments: [
    {
      name: "ShipmentRowFields",
      typeCondition: "Shipment",
      domain: "shipment",
      sourceFile: "src/operations/shipment/table.graphql",
      fragments: [],
      usedByOperations: ["ShipmentTable"],
      usages: [],
      sdl: "fragment ShipmentRowFields on Shipment { id }",
    },
  ],
  types: {},
};

async function importCatalogModule() {
  vi.resetModules();
  return import("../catalog");
}

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function okResponse() {
  return { ok: true, status: 200, json: () => Promise.resolve(CATALOG) };
}

describe("loadCatalog", () => {
  it("fetches the catalog asset and indexes it by name", async () => {
    fetchMock.mockResolvedValue(okResponse());
    const { loadCatalog } = await importCatalogModule();

    const index = await loadCatalog();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toContain("operation-catalog");
    expect(index.catalog.operationCount).toBe(1);
    expect(index.operationsByName.get("ShipmentTable")?.kind).toBe("query");
    expect(index.fragmentsByName.get("ShipmentRowFields")?.typeCondition).toBe("Shipment");
  });

  it("fetches once no matter how many callers ask for it", async () => {
    fetchMock.mockResolvedValue(okResponse());
    const { loadCatalog } = await importCatalogModule();

    const [first, second] = await Promise.all([loadCatalog(), loadCatalog()]);
    const third = await loadCatalog();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(second).toBe(first);
    expect(third).toBe(first);
  });

  it("rejects on a non-ok response", async () => {
    fetchMock.mockResolvedValue({ ok: false, status: 503, json: () => Promise.resolve({}) });
    const { loadCatalog } = await importCatalogModule();

    await expect(loadCatalog()).rejects.toThrow("status 503");
  });

  it("does not cache a failure, so a retry re-fetches", async () => {
    fetchMock.mockRejectedValueOnce(new Error("offline")).mockResolvedValue(okResponse());
    const { loadCatalog } = await importCatalogModule();

    await expect(loadCatalog()).rejects.toThrow("offline");
    const index = await loadCatalog();

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(index.operationsByName.size).toBe(1);
  });
});

describe("parseSelectionParam", () => {
  it("only resolves names the loaded catalog actually knows", async () => {
    fetchMock.mockResolvedValue(okResponse());
    const { loadCatalog, parseSelectionParam } = await importCatalogModule();
    const index = await loadCatalog();

    expect(parseSelectionParam(index, "op:ShipmentTable")).toEqual({
      kind: "operation",
      name: "ShipmentTable",
    });
    expect(parseSelectionParam(index, "fr:ShipmentRowFields")).toEqual({
      kind: "fragment",
      name: "ShipmentRowFields",
    });
    expect(parseSelectionParam(index, "op:NotARealOperation")).toBeNull();
    expect(parseSelectionParam(index, null)).toBeNull();
  });
});
