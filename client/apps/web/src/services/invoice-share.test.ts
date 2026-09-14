import { api } from "@trenova/shared/lib/api";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceShareService } from "./invoice-share";

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
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

afterEach(() => {
  vi.clearAllMocks();
});

describe("InvoiceShareService.list", () => {
  it("resolves to an empty list when the invoice has not been shared", async () => {
    vi.mocked(api.get).mockResolvedValue({ shares: [] });

    const shares = await new InvoiceShareService().list("inv_1");

    expect(api.get).toHaveBeenCalledWith("/billing/invoices/inv_1/shares/");
    expect(shares).toEqual([]);
  });

  it("resolves to the parsed shares", async () => {
    vi.mocked(api.get).mockResolvedValue({ shares: [share] });

    const shares = await new InvoiceShareService().list("inv_1");

    expect(shares).toHaveLength(1);
    expect(shares[0].sharedWith?.name).toBe("Dana Whitfield");
    expect(shares[0].note).toBeNull();
  });

  it("resolves to an empty list when the server sends a null list", async () => {
    vi.mocked(api.get).mockResolvedValue({ shares: null });

    await expect(new InvoiceShareService().list("inv_1")).resolves.toEqual([]);
  });
});

describe("InvoiceShareService.share", () => {
  it("posts the recipients, trimmed note and tab and resolves to the parsed result", async () => {
    vi.mocked(api.post).mockResolvedValue({
      shares: [share],
      recipientCount: 1,
      emailsQueued: 1,
      emailStatus: "Queued",
    });

    const result = await new InvoiceShareService().share("inv_1", {
      userIds: ["usr_dana"],
      note: "  Check the detention line  ",
      tab: "charges",
    });

    expect(api.post).toHaveBeenCalledWith("/billing/invoices/inv_1/shares/", {
      userIds: ["usr_dana"],
      note: "Check the detention line",
      tab: "charges",
    });
    expect(result.emailStatus).toBe("Queued");
    expect(result.shares).toHaveLength(1);
  });
});

describe("InvoiceShareService.candidates", () => {
  it("searches the eligible teammates and resolves to the parsed results", async () => {
    vi.mocked(api.get).mockResolvedValue({
      results: [share.sharedWith],
      count: 1,
      next: null,
      prev: null,
    });

    const candidates = await new InvoiceShareService().candidates("inv_1", "  dana ");

    expect(api.get).toHaveBeenCalledWith(
      "/billing/invoices/inv_1/shares/candidates/?limit=10&offset=0&query=dana",
      { signal: undefined },
    );
    expect(candidates).toHaveLength(1);
    expect(candidates[0].name).toBe("Dana Whitfield");
  });

  it("leaves the query out when nothing has been typed", async () => {
    vi.mocked(api.get).mockResolvedValue({ results: [], count: 0, next: null, prev: null });

    await expect(new InvoiceShareService().candidates("inv_1", "   ")).resolves.toEqual([]);
    expect(api.get).toHaveBeenCalledWith(
      "/billing/invoices/inv_1/shares/candidates/?limit=10&offset=0",
      { signal: undefined },
    );
  });
});
