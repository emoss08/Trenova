import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentGeneralInformation, { BOLField } from "../shipment-general-information";

const { checkForDuplicateBOLs, getUIPolicy, getBillingProfile } = vi.hoisted(() => ({
  checkForDuplicateBOLs: vi.fn(),
  getUIPolicy: vi.fn(),
  getBillingProfile: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentService: {
      checkForDuplicateBOLs,
      getUIPolicy,
    },
    customerService: {
      getBillingProfile,
    },
  },
}));

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });
}

function TestForm() {
  const form = useForm({
    defaultValues: {
      id: undefined,
      bol: "",
      customerId: "",
    },
  });

  return (
    <QueryClientProvider client={createQueryClient()}>
      <FormProvider {...form}>
        <button
          type="button"
          onClick={() => {
            form.reset({
              // id: "shp_123" as Shipment["id"],
              bol: "BOL-123",
              customerId: "",
            });
          }}
        >
          Hydrate
        </button>
        <BOLField />
      </FormProvider>
    </QueryClientProvider>
  );
}

function TestSection({ customerId = "" }: { customerId?: string }) {
  const form = useForm({
    defaultValues: { id: undefined, bol: "", customerId },
  });

  return (
    <QueryClientProvider client={createQueryClient()}>
      <FormProvider {...form}>
        <ShipmentGeneralInformation />
      </FormProvider>
    </QueryClientProvider>
  );
}

describe("BOLField", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getUIPolicy.mockResolvedValue({ checkForDuplicateBols: true });
    getBillingProfile.mockResolvedValue(undefined);
    checkForDuplicateBOLs.mockResolvedValue({ valid: true });
  });

  it("sends the hydrated shipment id when checking duplicate BOLs", async () => {
    const user = userEvent.setup();

    render(<TestForm />);

    await user.click(screen.getByRole("button", { name: "Hydrate" }));

    await waitFor(
      () => {
        expect(checkForDuplicateBOLs).toHaveBeenCalledWith("BOL-123", undefined);
      },
      { timeout: 2000 },
    );

    expect(screen.queryByText(/BOL is already in use/i)).not.toBeInTheDocument();
  });

  // documentation.duplicateBol names `bol` at Block, but it forbids reusing a
  // number rather than demanding one, so the catalog leaves `bol` out of the
  // rule's requiredFields. This asserts the whole chain agrees: the fixture
  // carries the rule targeting bol, and the field still comes out optional.
  //
  // It fails if the catalog ever marks bol required, or if the descriptor
  // starts passing the resolved `required` down.
  it("is not required merely because a blocking rule names the bol field", async () => {
    getUIPolicy.mockResolvedValue({
      checkForDuplicateBols: true,
      profile: {
        profileId: "mpf_1",
        profileCode: "OTR",
        profileName: "OTR Truckload",
        serviceModel: "Truckload",
        equipmentClass: "DryVan",
        executionParty: "CompanyAsset",
        capabilities: ["Core"],
        rules: {
          "documentation.duplicateBol": {
            key: "documentation.duplicateBol",
            capability: "Core",
            label: "Duplicate Bill of Lading",
            enforcement: "Block",
            enabled: true,
            fields: ["bol"],
            parameters: {},
            provenance: {
              profileId: "mpf_1",
              profileCode: "OTR",
              profileName: "OTR Truckload",
              isOrgDefault: true,
              priority: 0,
              specificityScore: 0,
              matchedOn: ["organizationDefault"],
              ruleKey: "documentation.duplicateBol",
              capability: "Core",
              capabilityLabel: "Core Operations",
              enforcement: "Block",
              defaultEnforcement: "Block",
              overridden: false,
              rationale: "A reused bill of lading usually indicates a double-entered load.",
            },
          },
        },
        candidates: [],
        resolvedAt: 0,
      },
    });
    // The billing profile is what actually decides, and it does not require one.
    getBillingProfile.mockResolvedValue({
      enforceCustomerBillingReq: true,
      requireBOLNumber: false,
    });

    // Renders the whole section, not BOLField alone: the guard is on the
    // descriptor wiring in CapabilityFields, so the descriptor path has to run.
    render(<TestSection />);

    await screen.findByPlaceholderText("Enter BOL");
    await waitFor(() => expect(getUIPolicy).toHaveBeenCalled());

    // InputField routes `required` to a class on the label rather than to the
    // DOM input, so that class is the only observable signal. Asserting on
    // toBeRequired() here would pass no matter what the field decided.
    expect(screen.getByText("BOL")).not.toHaveClass("required");
  });

  it("still marks BOL required when the customer's billing profile demands one", async () => {
    getUIPolicy.mockResolvedValue({ checkForDuplicateBols: true, profile: null });
    getBillingProfile.mockResolvedValue({
      enforceCustomerBillingReq: true,
      requireBOLNumber: true,
    });

    render(<TestSection customerId="cus_1" />);

    await screen.findByPlaceholderText("Enter BOL");

    await waitFor(() => expect(screen.getByText("BOL")).toHaveClass("required"));
  });
});
