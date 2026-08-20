import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { RateAgreement, RateIncreasePlan } from "@trenova/shared/types/rate";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RateIncreaseDialog } from "../rate-increase-dialog";

const previewRateIncrease = vi.fn();
const applyRateIncrease = vi.fn();

vi.mock("@/services/api", () => ({
  apiService: {
    get rateAgreementService() {
      return { previewRateIncrease, applyRateIncrease };
    },
  },
}));

vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: ({ control, name }: { control: Control; name: string }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label="Customer"
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
  CarrierAutocompleteField: () => null,
}));

vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: () => null,
}));

const selected = [
  { id: "ragr_1", code: "ACME-2026", name: "Acme" },
  { id: "ragr_2", code: "BETA-2026", name: "Beta" },
] as unknown as RateAgreement[];

const plan: RateIncreasePlan = {
  effectiveFrom: 1_700_000_000,
  agreementCount: 2,
  skippedNoRate: 1,
  negativeCount: 0,
  lines: [
    {
      rateAgreementId: "ragr_1",
      agreementCode: "ACME-2026",
      agreementName: "Acme",
      ruleId: "rarl_1",
      laneKey: "ST:GA>ST:FL",
      label: "GA to FL",
      before: 2,
      after: 2.1,
      breakCount: 0,
    },
  ],
};

function renderDialog(agreements: RateAgreement[] = selected) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RateIncreaseDialog open onOpenChange={() => {}} selectedAgreements={agreements} />
    </QueryClientProvider>,
  );
}

async function setAmountAndPreview(value: string) {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText("Change"), value);
  await user.click(screen.getByRole("button", { name: /preview changes/i }));
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("RateIncreaseDialog", () => {
  it("refuses to apply anything nobody has previewed", () => {
    renderDialog();

    expect(screen.getByRole("button", { name: /apply increase/i })).toBeDisabled();
    expect(applyRateIncrease).not.toHaveBeenCalled();
  });

  it("previews the selection and shows every lane's before and after", async () => {
    previewRateIncrease.mockResolvedValue(plan);
    renderDialog();

    await setAmountAndPreview("5");

    expect(await screen.findByText(/1 lanes across 2 agreements move/i)).toBeInTheDocument();
    expect(screen.getByText("GA to FL")).toBeInTheDocument();
    expect(screen.getByText(/1 matrix-priced lanes are untouched/i)).toBeInTheDocument();

    expect(previewRateIncrease).toHaveBeenCalledWith(
      expect.objectContaining({
        agreementIds: ["ragr_1", "ragr_2"],
        adjustment: { percentChange: 5 },
      }),
    );
  });

  it("applies only after a preview, with the same scope", async () => {
    previewRateIncrease.mockResolvedValue(plan);
    applyRateIncrease.mockResolvedValue(plan);
    const user = userEvent.setup();
    renderDialog();

    await setAmountAndPreview("5");
    await screen.findByText(/1 lanes across 2 agreements move/i);

    await user.click(screen.getByRole("button", { name: /apply increase/i }));

    await waitFor(() =>
      expect(applyRateIncrease).toHaveBeenCalledWith(
        expect.objectContaining({ agreementIds: ["ragr_1", "ragr_2"] }),
      ),
    );
  });

  it("blocks a decrease that would push lanes below zero", async () => {
    previewRateIncrease.mockResolvedValue({ ...plan, negativeCount: 1 });
    renderDialog();

    await setAmountAndPreview("-99");

    expect(await screen.findByText(/below zero/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /apply increase/i })).toBeDisabled();
  });
});
