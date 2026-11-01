import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceShareService } from "./invoice-share";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const share = {
  id: "invsh_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  invoiceId: "inv_1",
  sharedWithId: "usr_dana",
  sharedById: "usr_me",
  note: "",
  tab: "overview",
  shareCount: 1,
  firstSharedAt: 1_789_000_000,
  lastSharedAt: 1_789_000_000,
  createdAt: 1_789_000_000,
  updatedAt: 1_789_000_000,
  sharedWith: { id: "usr_dana", name: "Dana Whitfield", emailAddress: "dana@example.com" },
};

function lastRequest() {
  return vi.mocked(requestGraphQL).mock.calls.at(-1)?.[0];
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("InvoiceShareService.list", () => {
  it("reads the invoice's shares through InvoiceShares", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ invoiceShares: [share] });

    const shares = await new InvoiceShareService().list("inv_1");

    expect(lastRequest()).toMatchObject({
      operationName: "InvoiceShares",
      variables: { invoiceId: "inv_1" },
    });
    expect(shares).toHaveLength(1);
    expect(shares[0].sharedWith?.name).toBe("Dana Whitfield");
    expect(shares[0].note).toBeNull();
  });

  it("resolves to an empty list when the invoice has not been shared", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ invoiceShares: [] });

    await expect(new InvoiceShareService().list("inv_1")).resolves.toEqual([]);
  });
});

describe("InvoiceShareService.share", () => {
  it("sends the recipients, trimmed note and tab and resolves to the parsed result", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      shareInvoice: {
        shares: [share],
        recipientCount: 1,
        emailsQueued: 1,
        emailStatus: "Queued",
      },
    });

    const result = await new InvoiceShareService().share("inv_1", {
      userIds: ["usr_dana"],
      note: "  Check the detention line  ",
      tab: "charges",
    });

    expect(lastRequest()).toMatchObject({
      operationName: "ShareInvoice",
      variables: {
        invoiceId: "inv_1",
        input: { userIds: ["usr_dana"], note: "Check the detention line", tab: "charges" },
      },
    });
    expect(result.emailStatus).toBe("Queued");
    expect(result.shares).toHaveLength(1);
  });
});

describe("InvoiceShareService.candidates", () => {
  it("searches the eligible teammates with the trimmed query", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ invoiceShareCandidates: [share.sharedWith] });

    const candidates = await new InvoiceShareService().candidates("inv_1", "  dana ");

    expect(lastRequest()).toMatchObject({
      operationName: "InvoiceShareCandidates",
      variables: { invoiceId: "inv_1", query: "dana", limit: 10 },
    });
    expect(candidates).toHaveLength(1);
    expect(candidates[0].name).toBe("Dana Whitfield");
  });

  it("leaves the query out when nothing has been typed", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ invoiceShareCandidates: [] });

    await expect(new InvoiceShareService().candidates("inv_1", "   ")).resolves.toEqual([]);
    expect(lastRequest()?.variables).toEqual({ invoiceId: "inv_1", query: undefined, limit: 10 });
  });
});
