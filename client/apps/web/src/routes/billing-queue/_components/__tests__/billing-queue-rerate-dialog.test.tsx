import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueRerateDialog } from "../billing-queue-rerate-dialog";

const { updateCharges, getDeclaredInputs } = vi.hoisted(() => ({
  updateCharges: vi.fn(),
  getDeclaredInputs: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    billingQueueService: { updateCharges },
    formulaTemplateService: { getDeclaredInputs },
  },
}));
vi.mock("@/components/autocomplete-fields", () => ({
  FormulaTemplateAutocompleteField: () => null,
}));

const DECLARATIONS: TariffInputDeclaration[] = [
  {
    name: "minimumCharge",
    doc: null,
    type: "Money",
    descriptor: { kind: "money" },
    required: false,
    default: { $q: "95", u: "$" },
    range: null,
  },
];

afterEach(cleanup);
beforeEach(() => {
  vi.clearAllMocks();
  getDeclaredInputs.mockResolvedValue(DECLARATIONS);
  updateCharges.mockResolvedValue({});
});

function renderDialog(currentInputs: Record<string, unknown>) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <BillingQueueRerateDialog
        open
        onOpenChange={() => {}}
        itemId="bqi_1"
        currentTemplateId="ft_ltl"
        currentInputs={currentInputs}
      />
    </QueryClientProvider>,
  );
}

describe("BillingQueueRerateDialog", () => {
  it("re-rates on the chosen tariff with the values typed for its inputs", async () => {
    const user = userEvent.setup();
    renderDialog({ minimumCharge: { $q: "95", u: "$" } });

    const field = await screen.findByDisplayValue("95");
    await user.clear(field);
    await user.type(field, "120");
    await user.click(screen.getByRole("button", { name: "Re-rate" }));

    await waitFor(() =>
      expect(updateCharges).toHaveBeenCalledWith("bqi_1", {
        formulaTemplateId: "ft_ltl",
        tariffInputs: { minimumCharge: { $q: "120", u: "$" } },
      }),
    );
  });

  it("drops a value for an input the tariff does not declare", async () => {
    const user = userEvent.setup();
    renderDialog({ rate: { $q: "2.35", u: "$/mi" } });

    await screen.findByPlaceholderText("Default: $95.00");
    await user.click(screen.getByRole("button", { name: "Re-rate" }));
    await waitFor(() =>
      expect(updateCharges).toHaveBeenCalledWith("bqi_1", {
        formulaTemplateId: "ft_ltl",
        tariffInputs: {},
      }),
    );
  });
});
