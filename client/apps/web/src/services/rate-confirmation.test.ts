import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "@trenova/shared/lib/api";
import { RateConfirmationService } from "./rate-confirmation";

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

// Contract fixture mirroring PublicRateConfirmationView in
// services/tms/internal/core/services/rateconfirmationservice/public.go — every
// field is a non-omitted JSON key, strings may be empty, stops is always an array.
function publicView() {
  return {
    carrierName: "Knight Logistics LLC",
    companyName: "Acme Brokerage",
    shipmentProNumber: "PRO-10422",
    revisionLabel: "Revision 2",
    rateMethodLabel: "Flat Rate",
    baseRateLabel: "$1,500.00",
    totalCost: "$1,650.00",
    currency: "USD",
    paymentTermsLabel: "Net 30",
    stops: [
      {
        sequence: 1,
        type: "Pickup",
        name: "DC 12",
        address: "100 Main St, Dallas, TX 75201",
        window: "Jun 1, 08:00 – 12:00 UTC",
      },
      {
        sequence: 2,
        type: "Delivery",
        name: "Store 44",
        address: "9 Elm Ave, Tulsa, OK 74101",
        window: "",
      },
    ],
    status: "Sent",
    expiresAt: 1_760_000_000,
    confirmed: false,
  };
}

describe("RateConfirmationService public signing endpoints", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
    vi.mocked(api.post).mockReset();
  });

  it("fetches the preview from the public link endpoint with an encoded token", async () => {
    vi.mocked(api.get).mockResolvedValue(publicView());

    const service = new RateConfirmationService();
    const view = await service.getPublicRateConfirmation("raw/token+value");

    expect(api.get).toHaveBeenCalledWith("/rate-confirmation-links/raw%2Ftoken%2Bvalue/");
    expect(view.carrierName).toBe("Knight Logistics LLC");
    expect(view.stops).toHaveLength(2);
    expect(view.confirmed).toBe(false);
  });

  it("accepts a preview whose optional labels are empty strings", async () => {
    vi.mocked(api.get).mockResolvedValue({
      ...publicView(),
      companyName: "",
      paymentTermsLabel: "",
      baseRateLabel: "",
      stops: [],
      status: "Confirmed",
      confirmed: true,
    });

    const service = new RateConfirmationService();
    const view = await service.getPublicRateConfirmation("tok");

    expect(view.companyName).toBe("");
    expect(view.stops).toEqual([]);
    expect(view.confirmed).toBe(true);
  });

  it("posts the signer to the confirm endpoint", async () => {
    vi.mocked(api.post).mockResolvedValue({ status: "recorded" });

    const service = new RateConfirmationService();
    await service.confirmPublicRateConfirmation("raw/token", {
      signerName: "Jane Smith",
      signerTitle: "Operations Manager",
    });

    expect(api.post).toHaveBeenCalledWith("/rate-confirmation-links/raw%2Ftoken/confirm/", {
      signerName: "Jane Smith",
      signerTitle: "Operations Manager",
    });
  });

  it("sends an empty title when the signer omits it — the backend treats it as optional", async () => {
    vi.mocked(api.post).mockResolvedValue({ status: "recorded" });

    const service = new RateConfirmationService();
    await service.confirmPublicRateConfirmation("tok", {
      signerName: "Jane Smith",
      signerTitle: "",
    });

    expect(api.post).toHaveBeenCalledWith("/rate-confirmation-links/tok/confirm/", {
      signerName: "Jane Smith",
      signerTitle: "",
    });
  });
});
