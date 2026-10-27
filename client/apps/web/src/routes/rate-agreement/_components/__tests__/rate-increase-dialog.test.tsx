import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { RateAgreementRow } from "@/lib/graphql/rate-tables";
import { rateIncreasePlanSchema, type RateIncreasePlan } from "@trenova/shared/types/rate";
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
] as unknown as RateAgreementRow[];

// The plan as `PlanRateIncrease` returns it: each lane's own input values before and after,
// and the names of the ones the increase scaled.
const plan: RateIncreasePlan = {
  effectiveFrom: 1_700_000_000,
  agreementCount: 2,
  skippedNoMoney: 1,
  lines: [
    {
      rateAgreementId: "ragr_1",
      agreementCode: "ACME-2026",
      agreementName: "Acme",
      ruleId: "rarl_1",
      laneKey: "ST:GA>ST:FL",
      label: "GA to FL",
      before: {
        ratePerMile: { $q: "2.00", u: "$/mi" },
        hazmat: true,
        breaks: [
          { from: { $q: "0", u: "lb" }, rate: { $q: "40", u: "$/cwt" }, minimum: null },
          { from: { $q: "500", u: "lb" }, rate: { $q: "30", u: "$/cwt" }, minimum: null },
        ],
      },
      after: {
        ratePerMile: { $q: "2.10", u: "$/mi" },
        hazmat: true,
        breaks: [
          { from: { $q: "0", u: "lb" }, rate: { $q: "42", u: "$/cwt" }, minimum: null },
          { from: { $q: "500", u: "lb" }, rate: { $q: "31.5", u: "$/cwt" }, minimum: null },
        ],
      },
      changed: ["breaks", "ratePerMile"],
    },
  ],
};

function renderDialog(agreements: RateAgreementRow[] = selected) {
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
    expect(screen.getByText(/1 lane sets no money term of its own/i)).toBeInTheDocument();
    expect(screen.getByText("Rate per mile")).toBeInTheDocument();
    expect(screen.getByText("$2.00/mi → $2.10/mi")).toBeInTheDocument();
    expect(screen.getByText("2 rows move")).toBeInTheDocument();

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

  it("lists only the inputs the increase moves, under the agreement that holds the lane", async () => {
    previewRateIncrease.mockResolvedValue(plan);
    renderDialog();

    await setAmountAndPreview("5");

    expect(await screen.findByText("Acme")).toBeInTheDocument();
    expect(screen.queryByText("Hazmat")).not.toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
  });

  it("refuses a decrease of 100% or more before asking the server", async () => {
    renderDialog();

    await setAmountAndPreview("-100");

    expect(
      await screen.findByText(
        "A decrease of 100% or more would take every rate it touches to nothing or below.",
      ),
    ).toBeInTheDocument();
    expect(previewRateIncrease).not.toHaveBeenCalled();
  });

  it("reads a plan with no lanes and says nothing in scope moves", async () => {
    previewRateIncrease.mockResolvedValue(
      rateIncreasePlanSchema.parse({ effectiveFrom: 1_700_000_000, lines: null }),
    );
    renderDialog();

    await setAmountAndPreview("5");

    expect(
      await screen.findByText("No lane in scope sets a money term this change could move."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /apply increase/i })).toBeDisabled();
  });
});
