import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { AccountingControl } from "@/types/accounting-control";
import AccountingControlForm from "../_components/accounting-control-form";
import { savedAccountingControl } from "./accounting-control-fixture";

const served = vi.hoisted(() => ({ control: null as unknown, refusal: null as unknown }));

vi.mock("@/lib/queries/accounting-control", () => ({
  accountingControl: {
    get: Object.assign(
      () => ({ queryKey: ["accounting-control"], queryFn: async () => served.control }),
      { _def: ["accounting-control"] },
    ),
  },
}));

// The save hands the refusal the server returned to the form, as the real mutation does.
vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: ({ form }: { form: unknown }) => ({
    mutateAsync: async () => {
      if (!served.refusal) return;
      const { handleMutationError } = await vi.importActual<
        typeof import("@/hooks/use-api-mutation")
      >("@/hooks/use-api-mutation");
      handleMutationError({ error: served.refusal, form: form as never });
      throw served.refusal;
    },
  }),
}));

const automatic: AccountingControl = {
  ...savedAccountingControl,
  journalPostingMode: "Automatic",
  autoPostSourceEvents: [
    "InvoicePosted",
    "CreditMemoPosted",
    "DebitMemoPosted",
    "VendorBillPosted",
  ],
  defaultRevenueAccountId: "gla_revenue",
  defaultExpenseAccountId: "gla_expense",
  defaultArAccountId: "gla_ar",
  defaultApAccountId: "gla_ap",
};

function signIn(capabilities: OrganizationCapabilities) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Carrier Co", ...capabilities },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
}

function renderForm(search = "") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <TestRouter search={search}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <AccountingControlForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

async function choose(user: ReturnType<typeof userEvent.setup>, current: string, next: string) {
  await user.click(await screen.findByRole("button", { name: current }));
  await user.click(screen.getAllByText(next).at(-1)!);
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

afterEach(() => {
  served.control = savedAccountingControl;
  served.refusal = null;
  cleanup();
  useAuthStore.setState({ user: null, isAuthenticated: false });
});

describe("accounting control tabs", () => {
  it("gives an organization that runs a fleet every tab, posting first", async () => {
    served.control = savedAccountingControl;
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });

    renderForm();
    await screen.findByText("Recognition");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Posting",
      "Default accounts",
      "Driver pay",
      "Close & reconciliation",
      "Currency",
    ]);
  });

  it("leaves driver pay out for a brokerage, which settles no drivers", async () => {
    served.control = savedAccountingControl;
    signIn({ brokerageEnabled: true, assetOperationsEnabled: false });

    renderForm();
    await screen.findByText("Recognition");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Posting",
      "Default accounts",
      "Close & reconciliation",
      "Currency",
    ]);
    expect(screen.queryByText("Escrow interest expense account")).toBeNull();
  });
});

describe("accounting control driver pay", () => {
  it("offers the escrow adjustment account beside the other escrow accounts", async () => {
    served.control = savedAccountingControl;
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });

    renderForm("?tab=driverPay");

    expect(await screen.findByText("Escrow adjustment account")).toBeVisible();
    expect(
      screen.getByText(
        "Offsets escrow liability for manual escrow adjustments, kept apart from purchased transportation so corrections stay visible and reconcilable; adjustments cannot be recorded without it.",
      ),
    ).toBeVisible();
  });
});

describe("accounting control posting", () => {
  it("starts automatic posting with the events the recognition policies need", async () => {
    served.control = savedAccountingControl;
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });
    const user = userEvent.setup();

    renderForm();
    await choose(user, "Manual", "Automatic");

    const events = await screen.findByRole("group", { name: "Events that post journals" });
    const checked = within(events)
      .getAllByRole("checkbox")
      .filter(
        (box) => box.getAttribute("aria-checked") === "true" || (box as HTMLInputElement).checked,
      )
      .map((box) => box.id.replace("multi-checkbox-autoPostSourceEvents-", ""));
    expect(checked).toEqual([
      "InvoicePosted",
      "CreditMemoPosted",
      "DebitMemoPosted",
      "VendorBillPosted",
    ]);
  });

  it("groups the events by the ledger they post from", async () => {
    served.control = automatic;
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });

    renderForm();

    const events = await screen.findByRole("group", { name: "Events that post journals" });
    expect(
      within(events)
        .getAllByRole("group")
        .map((group) => group.getAttribute("aria-label")),
    ).toEqual(["Customers", "Vendors", "Driver settlements", "Carrier settlements", "Factoring"]);
    expect(within(events).getAllByRole("checkbox")).toHaveLength(21);
  });

  it("sets both policies when the basis changes, and swaps the events posting needs", async () => {
    served.control = automatic;
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });
    const user = userEvent.setup();

    renderForm();
    await choose(user, "Accrual", "Cash");

    expect(await screen.findByRole("button", { name: "On cash receipt" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "On cash disbursement" })).toBeInTheDocument();
    const events = screen.getByRole("group", { name: "Events that post journals" });
    const box = (label: string) => within(events).getByRole("checkbox", { name: label });
    await waitFor(() => expect(box("Customer payment posted")).toBeChecked());
    expect(box("Vendor payment posted")).toBeChecked();
    expect(box("Invoice posted")).not.toBeChecked();
    expect(box("Invoice posted")).toBeDisabled();
    expect(box("Vendor bill posted")).not.toBeChecked();
  });

  it("offers only the recognition policy the basis allows", async () => {
    served.control = savedAccountingControl;
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });
    const user = userEvent.setup();

    renderForm();
    await user.click(await screen.findByRole("button", { name: "On invoice post" }));

    expect(screen.queryByText("On cash receipt")).toBeNull();
  });
});

describe("a refused save", () => {
  // The server names an entry of the list (`autoPostSourceEvents[1]`); the tab that holds
  // the list opens, marked, and the list says why.
  it("opens and marks Posting when the server refuses one auto-post event", async () => {
    served.control = automatic;
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Duplicate journal source event",
      instance: "/api/v1/accounting-controls/",
      code: "INVALID",
      errors: [
        {
          path: ["autoPostSourceEvents", 1],
          message: "Duplicate journal source event",
          code: "DUPLICATE",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });
    const user = userEvent.setup();

    renderForm("?tab=close");
    await user.click(
      await screen.findByRole("switch", { name: /Require approval to close a period/ }),
    );
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const posting = await screen.findByRole("tab", { name: /Posting/, selected: true });
    expect(within(posting).getByText("Has fields to fix")).toBeInTheDocument();
    const events = screen.getByRole("group", { name: "Events that post journals" });
    expect(within(events.parentElement!).getByText("Duplicate journal source event")).toBeVisible();
  });
  it("opens and marks Driver pay when the server refuses the escrow adjustment account", async () => {
    served.control = automatic;
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Default escrow adjustment account does not exist in your organization",
      instance: "/api/v1/accounting-controls/",
      code: "INVALID",
      errors: [
        {
          path: ["defaultEscrowAdjustmentAccountId"],
          message: "Default escrow adjustment account does not exist in your organization",
          code: "INVALID_REFERENCE",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });
    const user = userEvent.setup();

    renderForm("?tab=close");
    await user.click(
      await screen.findByRole("switch", { name: /Require approval to close a period/ }),
    );
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const driverPay = await screen.findByRole("tab", { name: /Driver pay/, selected: true });
    expect(within(driverPay).getByText("Has fields to fix")).toBeInTheDocument();
    expect(
      screen.getAllByText("Default escrow adjustment account does not exist in your organization")
        .length,
    ).toBeGreaterThan(0);
  });
});
