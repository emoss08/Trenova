import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { SettlementControl } from "@/types/settlement-control";
import SettlementControlForm from "../_components/settlement-control-form";
import { savedSettlementControl } from "./settlement-control-fixture";

const served = vi.hoisted(() => ({
  control: null as unknown,
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/settlement-control", () => ({
  settlementControl: {
    get: Object.assign(
      () => ({ queryKey: ["settlement-control"], queryFn: async () => served.control }),
      { _def: ["settlement-control"] },
    ),
  },
}));

// The save hands the refusal the server returned to the form, as the real mutation does.
vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: ({ form }: { form: unknown }) => ({
    mutateAsync: async (values: unknown) => {
      served.sent.push(values);
      if (!served.refusal) return;
      const { handleMutationError } = await vi.importActual<
        typeof import("@/hooks/use-api-mutation")
      >("@/hooks/use-api-mutation");
      handleMutationError({ error: served.refusal, form: form as never });
      throw served.refusal;
    },
  }),
}));

function renderForm(search = "") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <TestRouter search={search}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <SettlementControlForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

afterEach(() => {
  served.control = savedSettlementControl;
  served.refusal = null;
  served.sent = [];
  cleanup();
});

describe("settlement control form", () => {
  it("opens on the pay period, with review and advances beside it", async () => {
    served.control = savedSettlementControl;

    renderForm();
    await screen.findByText("Schedule");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Pay period",
      "Review & approval",
      "Advances & escrow",
    ]);
    expect(screen.getByRole("tab", { name: "Pay period" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  // The server skips the net pay check at a zero threshold, so how many settlements it
  // averages over means nothing then.
  it("asks how many settlements to average over only while the net pay check runs", async () => {
    served.control = savedSettlementControl;
    const user = userEvent.setup();

    renderForm("?tab=review");
    expect(await screen.findByRole("textbox", { name: /Average over/ })).toBeVisible();

    const threshold = screen.getByRole("textbox", { name: /Flag a change in net pay over/ });
    await user.clear(threshold);
    await user.type(threshold, "0");

    await waitFor(() => expect(screen.queryByRole("textbox", { name: /Average over/ })).toBeNull());
  });

  it("hides the average for a record saved with the check off", async () => {
    served.control = { ...savedSettlementControl, varianceThresholdPct: "0" };

    renderForm("?tab=review");
    await screen.findByText("Exceptions");

    expect(screen.queryByRole("textbox", { name: /Average over/ })).toBeNull();
  });

  it("saves the whole record, decimals as the strings they were typed", async () => {
    served.control = savedSettlementControl;
    const user = userEvent.setup();

    renderForm("?tab=deductions");
    const rate = await screen.findByRole("textbox", { name: /Yearly interest rate/ });
    await user.clear(rate);
    await user.type(rate, "4.25");
    await user.click(screen.getByRole("switch", { name: /Recover advances in full/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.sent).toHaveLength(1));
    expect(served.sent[0]).toEqual({
      ...savedSettlementControl,
      defaultEscrowInterestRate: "4.25",
      allowNegativeNet: true,
    } satisfies SettlementControl);
  });
});

describe("a refused save", () => {
  it("opens and marks the review tab when the server refuses the net pay threshold", async () => {
    served.control = savedSettlementControl;
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Variance threshold cannot be negative",
      instance: "/graphql",
      code: "INVALID",
      errors: [
        {
          path: ["varianceThresholdPct"],
          message: "Variance threshold cannot be negative",
          code: "INVALID",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();

    renderForm("?tab=deductions");
    await user.click(await screen.findByRole("switch", { name: /Recover advances in full/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const review = await screen.findByRole("tab", { name: /Review & approval/, selected: true });
    expect(within(review).getByText("Has fields to fix")).toBeInTheDocument();
    const threshold = screen.getByRole("textbox", { name: /Flag a change in net pay over/ });
    expect(
      within(threshold.closest('[data-slot="setting-row"]')!).getByText(
        "Variance threshold cannot be negative",
      ),
    ).toBeVisible();
  });
});
