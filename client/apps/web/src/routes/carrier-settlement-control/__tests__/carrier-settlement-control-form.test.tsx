import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { seedPlan, unavailableFeatures } from "@/test/plan";
import type { CarrierSettlementControl } from "@/types/carrier-settlement-control";
import CarrierSettlementControlForm from "../_components/carrier-settlement-control-form";
import { savedCarrierSettlementControl } from "./carrier-settlement-control-fixture";

const served = vi.hoisted(() => ({
  control: null as unknown,
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/carrier-settlement-control", () => ({
  carrierSettlementControl: {
    get: Object.assign(
      () => ({ queryKey: ["carrier-settlement-control"], queryFn: async () => served.control }),
      { _def: ["carrier-settlement-control"] },
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
          <CarrierSettlementControlForm />
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
  served.control = savedCarrierSettlementControl;
  served.refusal = null;
  served.sent = [];
  cleanup();
  usePermissionStore.setState({ manifest: null });
});

describe("carrier settlement control form", () => {
  it("opens on the pay period, with invoice matching and approval beside it", async () => {
    served.control = savedCarrierSettlementControl;

    renderForm();
    await screen.findByText("Schedule");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Pay period",
      "Invoice matching",
      "Approval & posting",
    ]);
    expect(screen.getByRole("tab", { name: "Pay period" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  // The server only auto-accepts a match it made itself, so the choice means nothing while
  // invoices are not matched as they arrive.
  it("offers accepting matches within the tolerance only while invoices are matched as they arrive", async () => {
    served.control = savedCarrierSettlementControl;
    const user = userEvent.setup();

    renderForm("?tab=matching");
    const autoMatch = await screen.findByRole("switch", { name: /Match EDI invoices/ });
    expect(
      screen.queryByRole("switch", { name: /Accept matches within the tolerance/ }),
    ).toBeNull();

    await user.click(autoMatch);

    expect(
      await screen.findByRole("switch", { name: /Accept matches within the tolerance/ }),
    ).not.toBeChecked();
  });

  it("shows a saved auto-accept under matching that is already on", async () => {
    served.control = {
      ...savedCarrierSettlementControl,
      autoMatchInboundInvoices: true,
      autoAcceptWithinTolerance: true,
    } satisfies CarrierSettlementControl;

    renderForm("?tab=matching");

    expect(
      await screen.findByRole("switch", { name: /Accept matches within the tolerance/ }),
    ).toBeChecked();
  });

  it("turns accepting within the tolerance off when invoices stop being matched as they arrive", async () => {
    served.control = {
      ...savedCarrierSettlementControl,
      autoMatchInboundInvoices: true,
      autoAcceptWithinTolerance: true,
    } satisfies CarrierSettlementControl;
    const user = userEvent.setup();

    renderForm("?tab=matching");
    await user.click(await screen.findByRole("switch", { name: /Match EDI invoices/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.sent).toHaveLength(1));
    expect(served.sent[0]).toEqual({
      ...savedCarrierSettlementControl,
      autoMatchInboundInvoices: false,
      autoAcceptWithinTolerance: false,
    } satisfies CarrierSettlementControl);
  });

  it("leaves out matching invoices as they arrive on a plan without EDI", async () => {
    served.control = savedCarrierSettlementControl;
    seedPlan({}, unavailableFeatures("integrations.edi"));

    renderForm("?tab=matching");
    await screen.findByText("Matching");

    expect(screen.queryByRole("switch", { name: /Match EDI invoices/ })).toBeNull();
    expect(
      screen.getByRole("switch", { name: /Hold pay until the invoice is matched/ }),
    ).toBeInTheDocument();
  });

  it("saves the whole record, the tolerance in cents", async () => {
    served.control = savedCarrierSettlementControl;
    const user = userEvent.setup();

    renderForm("?tab=matching");
    const tolerance = await screen.findByRole("textbox", { name: /Variance tolerance/ });
    await user.tripleClick(tolerance);
    await user.keyboard("12.50");
    await user.click(screen.getByRole("switch", { name: /Match EDI invoices/ }));
    await user.click(
      await screen.findByRole("switch", { name: /Accept matches within the tolerance/ }),
    );
    await user.click(screen.getByRole("switch", { name: /Hold pay until the invoice is matched/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.sent).toHaveLength(1));
    expect(served.sent[0]).toEqual({
      ...savedCarrierSettlementControl,
      varianceToleranceMinor: 1250,
      autoMatchInboundInvoices: true,
      autoAcceptWithinTolerance: true,
      holdUntilInvoiceMatched: true,
    } satisfies CarrierSettlementControl);
  });
});

describe("a refused save", () => {
  it("opens and marks invoice matching when the server refuses the tolerance", async () => {
    served.control = savedCarrierSettlementControl;
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Variance tolerance cannot be negative",
      instance: "/graphql",
      code: "INVALID",
      errors: [
        {
          path: ["varianceToleranceMinor"],
          message: "Variance tolerance cannot be negative",
          code: "INVALID",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();

    renderForm("?tab=approval");
    await user.click(await screen.findByRole("switch", { name: /Post to the ledger on approval/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const matching = await screen.findByRole("tab", { name: /Invoice matching/, selected: true });
    expect(within(matching).getByText("Has fields to fix")).toBeInTheDocument();
    const tolerance = screen.getByRole("textbox", { name: /Variance tolerance/ });
    expect(
      within(tolerance.closest('[data-slot="setting-row"]')!).getByText(
        "Variance tolerance cannot be negative",
      ),
    ).toBeVisible();
  });
});
