import { beforeEach, describe, expect, it, vi } from "vitest";
import { OrganizationService } from "./organization";

const api = vi.hoisted(() => ({
  upload: vi.fn(),
  delete: vi.fn(),
  put: vi.fn(),
}));

vi.mock("@trenova/shared/lib/api", async (importActual) => ({
  ...(await importActual<typeof import("@trenova/shared/lib/api")>()),
  api,
}));

// What the TMS answers for an organization a new instance's bootstrap made: its name, a
// time zone and state, no street address, and the bootstrap's stand-ins elsewhere. The
// organization service's Update refuses to save it, but every read and the branding
// writes return it as it is.
const bootstrappedOrganization = {
  id: "org_01",
  version: 0,
  createdAt: 1794441464,
  updatedAt: 1794441464,
  businessUnitId: "bu_01",
  loginSlug: "acme-freight",
  name: "Acme Freight",
  scacCode: "TBDX",
  dotNumber: "0",
  logoUrl: "org_01/organization/logo/acme.png",
  iconUrl: "",
  addressLine1: "",
  addressLine2: "",
  city: "Pending",
  stateId: "us_01",
  postalCode: "00000",
  timezone: "America/New_York",
  taxId: "",
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};

describe("organization branding writes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns the saved logo for an organization whose address is not filled in", async () => {
    api.upload.mockResolvedValueOnce(bootstrappedOrganization);
    const file = new File(["png"], "acme.png", { type: "image/png" });

    const saved = await new OrganizationService().uploadBrandImage("org_01", "logo", file);

    expect(api.upload).toHaveBeenCalledWith(
      "/organizations/org_01/branding/logo",
      expect.any(FormData),
    );
    expect(saved.logoUrl).toBe("org_01/organization/logo/acme.png");
    expect(saved.addressLine1).toBe("");
  });

  it("returns the organization after a picture is removed, whatever its address holds", async () => {
    api.delete.mockResolvedValueOnce({ ...bootstrappedOrganization, logoUrl: "" });

    const saved = await new OrganizationService().deleteBrandImage("org_01", "logo");

    expect(saved.logoUrl).toBe("");
    expect(saved.city).toBe("Pending");
  });
});
