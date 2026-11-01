import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ediGraphQLResponse } from "@/test/edi-graphql";

const request = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/graphql")>();
  return { ...actual, requestGraphQL: request };
});

const { edi } = await import("@/lib/queries/edi");
const { wireInspection, wireMessage } =
  await import("@/routes/desk/_components/artifacts/cards/edi-message/__tests__/fixtures");

beforeEach(() => {
  request.mockReset();
});

describe("EDI message queries", () => {
  it("reads one message with its signal, keeping the invoice it bills and a tender's purpose", async () => {
    request.mockResolvedValue(
      ediGraphQLResponse(
        "EdiMessageRecord",
        wireMessage({
          invoiceId: "inv_1",
          transactionSet: "204",
          payloadSnapshot: { transactionSet: "204", purposeCode: "04", loadTender: { bol: "B-1" } },
        }),
      ),
    );
    const message = await new QueryClient().fetchQuery(edi.message("edimsg_1"));

    expect(request).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "EdiMessageRecord",
        variables: { id: "edimsg_1" },
        signal: expect.any(AbortSignal),
      }),
    );
    expect(message.invoiceId).toBe("inv_1");
    expect(message.standard).toBe("X12");
    expect(message.payloadSnapshot?.purposeCode).toBe("04");
  });

  it("reads the message's findings as its validation errors", async () => {
    request.mockResolvedValue({
      ediMessage: {
        message: wireMessage(),
        diagnosticCount: 1,
        diagnostics: [
          {
            severity: "Error",
            code: "E1",
            segmentId: "B10",
            elementPosition: null,
            path: null,
            message: "Missing reference",
            suggestedFix: null,
          },
        ],
      },
    });
    const message = await new QueryClient().fetchQuery(edi.message("edimsg_1"));

    expect(message.diagnosticCount).toBe(1);
    expect(message.validationErrors).toEqual([
      expect.objectContaining({ code: "E1", segmentId: "B10", elementPosition: 0 }),
    ]);
  });

  it("inspects one message with its signal", async () => {
    request.mockResolvedValue(ediGraphQLResponse("EdiMessageInspection", wireInspection()));
    const inspected = await new QueryClient().fetchQuery(edi.messageInspection("edimsg_1"));

    expect(request).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "EdiMessageInspection",
        variables: { id: "edimsg_1" },
        signal: expect.any(AbortSignal),
      }),
    );
    expect(inspected.inspection.segments).toHaveLength(10);
  });
});
