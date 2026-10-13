import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

const get = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, get } };
});

const { queries } = await import("@/lib/queries");
const { wireInspection, wireMessage } =
  await import("@/routes/desk/_components/artifacts/cards/edi-message/__tests__/fixtures");

beforeEach(() => {
  get.mockReset();
});

describe("EDI message queries", () => {
  it("reads one message with its signal, keeping the invoice it bills and a tender's purpose", async () => {
    get.mockResolvedValue(
      wireMessage({
        invoiceId: "inv_1",
        transactionSet: "204",
        payloadSnapshot: { transactionSet: "204", purposeCode: "04", loadTender: { bol: "B-1" } },
      }),
    );
    const message = await new QueryClient().fetchQuery(queries.edi.message("edimsg_1"));

    expect(get).toHaveBeenCalledWith("/edi/messages/edimsg_1/", {
      signal: expect.any(AbortSignal),
    });
    expect(message.invoiceId).toBe("inv_1");
    expect(message.payloadSnapshot?.purposeCode).toBe("04");
  });

  it("inspects one message with its signal", async () => {
    get.mockResolvedValue(wireInspection());
    const inspected = await new QueryClient().fetchQuery(queries.edi.messageInspection("edimsg_1"));

    expect(get).toHaveBeenCalledWith("/edi/messages/edimsg_1/inspect/", {
      signal: expect.any(AbortSignal),
    });
    expect(inspected.inspection.segments).toHaveLength(10);
  });
});
