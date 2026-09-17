import type { CarrierEquipmentVerification } from "@/lib/graphql/carrier-intelligence";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EquipmentVerificationCard } from "./equipment-verification-card";
import { VerifyEquipmentDialog } from "./verify-equipment-dialog";

const mocks = vi.hoisted(() => ({
  permissions: new Set<string>(["read", "create", "approve"]),
  fetchCarrierEquipmentVerifications: vi.fn(),
  verifyCarrierEquipment: vi.fn(),
  overrideCarrierEquipmentVerification: vi.fn(),
}));

vi.mock("@/hooks/use-permission", async () => {
  const { Operation } = await import("@trenova/shared/types/permission");
  const names: Record<number, string> = {
    [Operation.Read]: "read",
    [Operation.Create]: "create",
    [Operation.Approve]: "approve",
  };
  return {
    usePermission: (_resource: string, operation: number) => ({
      allowed: mocks.permissions.has(names[operation] ?? ""),
      isLoading: false,
    }),
  };
});

vi.mock("@/components/autocomplete-fields", () => ({
  UsStateAutocompleteField: () => null,
}));

vi.mock("@/lib/graphql/carrier-intelligence", () => ({
  CARRIER_EQUIPMENT_VERIFICATIONS_KEY: "carrier-equipment-verifications",
  fetchCarrierEquipmentVerifications: mocks.fetchCarrierEquipmentVerifications,
  verifyCarrierEquipment: mocks.verifyCarrierEquipment,
  overrideCarrierEquipmentVerification: mocks.overrideCarrierEquipmentVerification,
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

afterEach(() => {
  vi.clearAllMocks();
  mocks.permissions = new Set(["read", "create", "approve"]);
});

function verification(
  overrides: Partial<CarrierEquipmentVerification> = {},
): CarrierEquipmentVerification {
  return {
    id: "cev_1",
    carrierAssignmentId: "ca_1",
    shipmentMoveId: "sm_1",
    carrierId: "car_1",
    expectedDotNumber: "1234567",
    unitType: "Tractor",
    vin: "1FUJGLDR5CLBP8834",
    plateNumber: null,
    plateState: null,
    unitNumber: null,
    result: "Mismatch",
    matchedDotNumbers: ["7654321"],
    matchedLegalName: "Other Carrier LLC",
    mismatchReason: "Registered to Other Carrier LLC, not Acme Freight",
    provider: "CarrierOK",
    verifiedById: "usr_1",
    verifiedAt: 1_800_000_000,
    overrideById: null,
    overrideReason: null,
    overriddenAt: null,
    cleared: false,
    createdAt: 1_800_000_000,
    detail: {
      vin: "1FUJGLDR5CLBP8834",
      unitType: "Tractor",
      category: "Truck Tractor",
      make: "Freightliner",
      model: "Cascadia",
      year: 2012,
      plateNumber: "ABC1234",
      plateState: "TX",
      unitNumber: "T-12",
    },
    ...overrides,
  };
}

function withClient(node: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{node}</QueryClientProvider>;
}

describe("EquipmentVerificationCard", () => {
  it("shows the matched DOT, legal name, mismatch reason and vehicle detail", () => {
    render(
      <EquipmentVerificationCard verification={verification()} canApprove onOverride={vi.fn()} />,
    );

    expect(screen.getByText("7654321")).toBeInTheDocument();
    expect(screen.getByText("Other Carrier LLC")).toBeInTheDocument();
    expect(
      screen.getByText("Registered to Other Carrier LLC, not Acme Freight"),
    ).toBeInTheDocument();
    expect(screen.getByText("2012 Freightliner Cascadia")).toBeInTheDocument();
  });

  it("offers an override on an open mismatch to someone who can approve", async () => {
    const user = userEvent.setup();
    const onOverride = vi.fn();
    render(
      <EquipmentVerificationCard
        verification={verification()}
        canApprove
        onOverride={onOverride}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Override" }));
    expect(onOverride).toHaveBeenCalledWith(expect.objectContaining({ id: "cev_1" }));
  });

  it("hides the override without approval, on a match, and once overridden", () => {
    const { rerender } = render(
      <EquipmentVerificationCard
        verification={verification()}
        canApprove={false}
        onOverride={vi.fn()}
      />,
    );
    expect(screen.queryByRole("button", { name: "Override" })).toBeNull();

    rerender(
      <EquipmentVerificationCard
        verification={verification({ result: "Match", mismatchReason: null })}
        canApprove
        onOverride={vi.fn()}
      />,
    );
    expect(screen.queryByRole("button", { name: "Override" })).toBeNull();

    rerender(
      <EquipmentVerificationCard
        verification={verification({
          overriddenAt: 1_800_000_500,
          overrideById: "usr_2",
          overrideReason: "Leased on",
        })}
        canApprove
        onOverride={vi.fn()}
      />,
    );
    expect(screen.queryByRole("button", { name: "Override" })).toBeNull();
    expect(screen.getByText("Overridden")).toBeInTheDocument();
  });
});

describe("VerifyEquipmentDialog", () => {
  it("refuses an invalid VIN without calling the provider", async () => {
    const user = userEvent.setup();
    mocks.fetchCarrierEquipmentVerifications.mockResolvedValue([]);
    render(
      withClient(
        <VerifyEquipmentDialog
          carrierAssignmentId="ca_1"
          carrierName="Acme Freight"
          open
          onOpenChange={vi.fn()}
        />,
      ),
    );

    await user.type(screen.getByPlaceholderText("17-character VIN"), "1FUJGLDR5CLBP883Q");
    await user.click(screen.getByRole("button", { name: "Verify" }));

    expect(await screen.findByText("VIN cannot contain the letters I, O or Q")).toBeInTheDocument();
    expect(mocks.verifyCarrierEquipment).not.toHaveBeenCalled();
  });

  it("verifies a valid VIN and shows the mismatch with an override", async () => {
    const user = userEvent.setup();
    mocks.fetchCarrierEquipmentVerifications.mockResolvedValue([]);
    mocks.verifyCarrierEquipment.mockResolvedValue(verification());
    render(
      withClient(
        <VerifyEquipmentDialog
          carrierAssignmentId="ca_1"
          carrierName="Acme Freight"
          open
          onOpenChange={vi.fn()}
        />,
      ),
    );

    await user.type(screen.getByPlaceholderText("17-character VIN"), "1fujgldr5clbp8834");
    await user.click(screen.getByRole("button", { name: "Verify" }));

    await waitFor(() => expect(mocks.verifyCarrierEquipment).toHaveBeenCalledTimes(1));
    expect(mocks.verifyCarrierEquipment).toHaveBeenCalledWith({
      carrierAssignmentId: "ca_1",
      unitType: "Tractor",
      vin: "1FUJGLDR5CLBP8834",
    });

    const latest = await screen.findByRole("region", { name: "Latest result" });
    expect(within(latest).getByText("Mismatch")).toBeInTheDocument();
    expect(within(latest).getByRole("button", { name: "Override" })).toBeInTheDocument();
  });

  it("lists prior verifications for the assignment", async () => {
    mocks.fetchCarrierEquipmentVerifications.mockResolvedValue([
      verification({ id: "cev_2", result: "Match", mismatchReason: null }),
    ]);
    render(
      withClient(
        <VerifyEquipmentDialog
          carrierAssignmentId="ca_1"
          carrierName="Acme Freight"
          open
          onOpenChange={vi.fn()}
        />,
      ),
    );

    const history = await screen.findByRole("list", { name: "Prior verifications" });
    expect(within(history).getByText("Match")).toBeInTheDocument();
    expect(mocks.fetchCarrierEquipmentVerifications).toHaveBeenCalledWith(
      "ca_1",
      expect.anything(),
    );
  });
});
