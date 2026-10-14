import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

const get = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, get } };
});

const { edi } = await import("@/lib/queries/edi");
const { wireFileInspection, wireInboundFile } =
  await import("@/routes/desk/_components/artifacts/cards/edi-inbound-file/__tests__/fixtures");

beforeEach(() => {
  get.mockReset();
});

type WireMessage = {
  payloadSnapshot: { loadTender: { moves: { stops: Record<string, unknown>[] }[] } };
};

describe("EDI inbound file queries", () => {
  it("reads one file with its signal, keeping when its raw content was purged", async () => {
    get.mockResolvedValue(wireInboundFile("batch", { rawPurgedAt: 1_791_700_000 }));
    const file = await new QueryClient().fetchQuery(edi.inboundFile("ediinf_acme"));

    expect(get).toHaveBeenCalledWith("/edi/inbound-files/ediinf_acme/", {
      signal: expect.any(AbortSignal),
    });
    expect(file.rawPurgedAt).toBe(1_791_700_000);
    expect(file.messages?.map((message) => message.id)).toEqual([
      "edimsg_acme_204a",
      "edimsg_acme_204b",
      "edimsg_acme_997",
    ]);
  });

  it("reads a file that was never read as X12, whose envelope fields are blank", async () => {
    get.mockResolvedValue(wireInboundFile("quarantined"));
    const file = await new QueryClient().fetchQuery(edi.inboundFile("ediinf_bad"));

    expect(file.interchangeControlNumber).toBe("");
    expect(file.isaSenderId).toBe("");
    expect(file.transactionCount).toBe(0);
    expect(file.status).toBe("Quarantined");
  });

  it("reads a file from a mailbox no partner owns", async () => {
    get.mockResolvedValue(wireInboundFile("quarantined", { ediPartnerId: null, partner: null }));
    const file = await new QueryClient().fetchQuery(edi.inboundFile("ediinf_bad"));

    expect(file.ediPartnerId).toBeNull();
    expect(file.partner).toBeNull();
  });

  it("reads a file whose tender names a stop without a location code", async () => {
    const wire = wireInboundFile("simulator");
    const [message] = wire.messages as WireMessage[];
    const [stop] = message!.payloadSnapshot.loadTender.moves[0]!.stops;
    stop!.locationId = null;
    get.mockResolvedValue(wire);

    const file = await new QueryClient().fetchQuery(edi.inboundFile("ediinf_sim"));

    const parsed = file.messages?.[0]?.payloadSnapshot?.loadTender?.moves?.[0]?.stops?.[0];
    expect(parsed?.locationId).toBeNull();
    expect(parsed?.locationName).toBe("Simulator Shipper");
  });

  it("inspects one file's interchange with its signal", async () => {
    get.mockResolvedValue(wireFileInspection("batch"));
    const inspection = await new QueryClient().fetchQuery(edi.inboundFileInspection("ediinf_acme"));

    expect(get).toHaveBeenCalledWith("/edi/inbound-files/ediinf_acme/inspect/", {
      signal: expect.any(AbortSignal),
    });
    expect(inspection.groups.map((group) => group.functionalIdCode)).toEqual(["SM", "FA"]);
    expect(inspection.transactions).toHaveLength(3);
  });
});
