import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

const get = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, get } };
});

const { edi } = await import("@/lib/queries/edi");
const { wireMappingPreview, wireTransfer } =
  await import("@/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/fixtures");

beforeEach(() => {
  get.mockReset();
});

describe("EDI transfer queries", () => {
  it("reads an inbound tender with its signal, which has no source shipment", async () => {
    get.mockResolvedValue(wireTransfer("mappingRequired"));
    const transfer = await new QueryClient().fetchQuery(edi.transfer("edilt_acme_88241"));

    expect(get).toHaveBeenCalledWith("/edi/transfers/edilt_acme_88241/", {
      signal: expect.any(AbortSignal),
    });
    expect(transfer.sourceShipmentId).toBeNull();
    expect(transfer.inboundMessageId).toBe("edimsg_acme_204b");
    expect(transfer.sourcePartner?.name).toBe("Acme Foods");
    expect(transfer.tenderPayload.totalChargeAmount).toBe("2880");
  });

  it("reads a tender we sent, which names its shipment and has no inbound message", async () => {
    get.mockResolvedValue(wireTransfer("outbound"));
    const transfer = await new QueryClient().fetchQuery(edi.transfer("edilt_s2610_1366"));

    expect(transfer.sourceShipmentId).toBe("shp_s2610_1366");
    expect(transfer.inboundMessageId).toBeNull();
    expect(transfer.targetBusinessUnitId).toBe("bu_lonestar");
    expect(transfer.tenderPayload.ratingDetail).toBeUndefined();
  });

  it("previews a tender's mappings with its signal", async () => {
    get.mockResolvedValue(wireMappingPreview());
    const preview = await new QueryClient().fetchQuery(edi.mappingPreview("edilt_acme_88241"));

    expect(get).toHaveBeenCalledWith("/edi/transfers/edilt_acme_88241/mapping-preview/", {
      signal: expect.any(AbortSignal),
    });
    expect(preview.unresolved.map((row) => row.sourceId)).toEqual(["HEB-SAT", "FSC"]);
  });

  it("lists the tenders one inbound file opened, with its signal", async () => {
    get.mockResolvedValue({
      results: [wireTransfer("approved"), wireTransfer("mappingRequired")],
      count: 2,
      next: null,
      prev: null,
    });
    const transfers = await new QueryClient().fetchQuery(edi.inboundFileTransfers("ediinf_acme"));

    expect(get).toHaveBeenCalledWith(
      "/edi/transfers/?direction=inbound&inboundFileId=ediinf_acme&limit=100",
      { signal: expect.any(AbortSignal) },
    );
    expect(transfers.map((row) => row.id)).toEqual(["edilt_acme_88240", "edilt_acme_88241"]);
    expect(get).toHaveBeenCalledTimes(1);
  });

  it("reads every page of tenders a large file opened, in the order the server lists them", async () => {
    const total = 250;
    const tender = (index: number) =>
      wireTransfer("approved", {
        id: `edilt_batch_${String(index).padStart(3, "0")}`,
        inboundMessageId: `edimsg_batch_${String(index).padStart(3, "0")}`,
      });
    get.mockImplementation(async (url: string) => {
      const offset = Number(new URL(url, "http://api").searchParams.get("offset") ?? 0);
      const results = Array.from(
        { length: Math.max(Math.min(100, total - offset), 0) },
        (_, index) => tender(offset + index),
      );
      return { results, count: total, next: null, prev: null };
    });

    const transfers = await new QueryClient().fetchQuery(edi.inboundFileTransfers("ediinf_batch"));

    expect(get.mock.calls.map(([url]) => url)).toEqual([
      "/edi/transfers/?direction=inbound&inboundFileId=ediinf_batch&limit=100",
      "/edi/transfers/?direction=inbound&inboundFileId=ediinf_batch&limit=100&offset=100",
      "/edi/transfers/?direction=inbound&inboundFileId=ediinf_batch&limit=100&offset=200",
    ]);
    expect(transfers).toHaveLength(total);
    expect(transfers.map((row) => row.id)).toEqual(
      Array.from({ length: total }, (_, index) => tender(index).id),
    );
  });
});
