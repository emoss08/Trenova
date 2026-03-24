import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BOLField } from "../shipment-general-information";

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
});
