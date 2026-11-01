import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ediGraphQLKey, ediGraphQLResponse } from "@/test/edi-graphql";

const request = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/graphql")>();
  return { ...actual, requestGraphQL: request };
});

type Call = { operationName: string; variables: Record<string, unknown>; signal?: AbortSignal };
const keys = () =>
  request.mock.calls.map(([call]) => ediGraphQLKey(call.operationName, call.variables));

const { edi } = await import("@/lib/queries/edi");
const { wireMappingPreview, wireTransfer } =
  await import("@/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/fixtures");

beforeEach(() => {
  request.mockReset();
});

describe("EDI transfer queries", () => {
  it("reads an inbound tender with its signal, which has no source shipment", async () => {
    request.mockResolvedValue(
      ediGraphQLResponse("EdiTransferRecord", wireTransfer("mappingRequired")),
    );
    const transfer = await new QueryClient().fetchQuery(edi.transfer("edilt_acme_88241"));

    expect(request).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "EdiTransferRecord",
        variables: { id: "edilt_acme_88241" },
        signal: expect.any(AbortSignal),
      }),
    );
    expect(transfer.sourceShipmentId).toBeNull();
    expect(transfer.inboundMessageId).toBe("edimsg_acme_204b");
    expect(transfer.sourcePartner?.name).toBe("Acme Foods");
    expect(transfer.tenderPayload.totalChargeAmount).toBe("2880");
  });

  it("reads a tender we sent, which names its shipment and has no inbound message", async () => {
    request.mockResolvedValue(ediGraphQLResponse("EdiTransferRecord", wireTransfer("outbound")));
    const transfer = await new QueryClient().fetchQuery(edi.transfer("edilt_s2610_1366"));

    expect(transfer.sourceShipmentId).toBe("shp_s2610_1366");
    expect(transfer.inboundMessageId).toBeNull();
    expect(transfer.targetBusinessUnitId).toBe("bu_lonestar");
    expect(transfer.tenderPayload.ratingDetail).toBeUndefined();
  });

  it("previews a tender's mappings with its signal", async () => {
    request.mockResolvedValue(
      ediGraphQLResponse("EdiTransferMappingPreview", wireMappingPreview()),
    );
    const preview = await new QueryClient().fetchQuery(edi.mappingPreview("edilt_acme_88241"));

    expect(request).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "EdiTransferMappingPreview",
        variables: { id: "edilt_acme_88241" },
        signal: expect.any(AbortSignal),
      }),
    );
    expect(preview.unresolved.map((row) => row.sourceId)).toEqual(["HEB-SAT", "FSC"]);
  });

  it("lists the tenders one inbound file opened, with its signal", async () => {
    request.mockResolvedValue(
      ediGraphQLResponse("EdiInboundFileTransfers", {
        results: [wireTransfer("approved"), wireTransfer("mappingRequired")],
      }),
    );
    const transfers = await new QueryClient().fetchQuery(edi.inboundFileTransfers("ediinf_acme"));

    expect(request).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "EdiInboundFileTransfers",
        variables: { inboundFileId: "ediinf_acme", input: { first: 100, after: null } },
        signal: expect.any(AbortSignal),
      }),
    );
    expect(transfers.map((row) => row.id)).toEqual(["edilt_acme_88240", "edilt_acme_88241"]);
    expect(request).toHaveBeenCalledTimes(1);
  });

  it("reads every page of tenders a large file opened, in the order the server lists them", async () => {
    const total = 250;
    const tender = (index: number) =>
      wireTransfer("approved", {
        id: `edilt_batch_${String(index).padStart(3, "0")}`,
        inboundMessageId: `edimsg_batch_${String(index).padStart(3, "0")}`,
      });
    const cursorAt = (offset: number) => `page-${offset}`;
    request.mockImplementation(async ({ variables }: Call) => {
      const after = (variables.input as { after: string | null }).after;
      const offset = after ? Number(after.replace("page-", "")) : 0;
      const results = Array.from(
        { length: Math.max(Math.min(100, total - offset), 0) },
        (_, index) => tender(offset + index),
      );
      const hasNextPage = offset + results.length < total;
      return ediGraphQLResponse("EdiInboundFileTransfers", {
        results,
        hasNextPage,
        endCursor: hasNextPage ? cursorAt(offset + results.length) : "",
      });
    });

    const transfers = await new QueryClient().fetchQuery(edi.inboundFileTransfers("ediinf_batch"));

    expect(keys()).toEqual([
      "EdiInboundFileTransfers:ediinf_batch:",
      "EdiInboundFileTransfers:ediinf_batch:page-100",
      "EdiInboundFileTransfers:ediinf_batch:page-200",
    ]);
    expect(transfers).toHaveLength(total);
    expect(transfers.map((row) => row.id)).toEqual(
      Array.from({ length: total }, (_, index) => tender(index).id),
    );
  });
});
