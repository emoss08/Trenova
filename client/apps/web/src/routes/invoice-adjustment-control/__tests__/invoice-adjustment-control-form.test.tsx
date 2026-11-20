import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { InvoiceAdjustmentControl } from "@/types/invoice-adjustment-control";
import InvoiceAdjustmentControlForm from "../_components/invoice-adjustment-control-form";
import { savedInvoiceAdjustmentControl } from "./invoice-adjustment-control-fixture";

const served = vi.hoisted(() => ({
  control: null as unknown,
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/invoice-adjustment-control", () => ({
  invoiceAdjustmentControl: {
    get: Object.assign(
      () => ({ queryKey: ["invoice-adjustment-control"], queryFn: async () => served.control }),
      { _def: ["invoice-adjustment-control"] },
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
          <InvoiceAdjustmentControlForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

async function open(user: ReturnType<typeof userEvent.setup>, current: string) {
  await user.click(await screen.findByRole("button", { name: current }));
}

async function choose(user: ReturnType<typeof userEvent.setup>, current: string, next: string) {
  await open(user, current);
  await user.click(screen.getAllByText(next).at(-1)!);
}

function optionItem(label: string) {
  return screen.getAllByText(label).at(-1)!.closest('[role="option"]')!;
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

afterEach(() => {
  served.control = savedInvoiceAdjustmentControl;
  served.refusal = null;
  served.sent = [];
  cleanup();
});

describe("invoice adjustment control form", () => {
  it("opens on eligibility, with approval and credits beside it", async () => {
    served.control = savedInvoiceAdjustmentControl;

    renderForm();
    await screen.findByText("Invoices that can be adjusted");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Eligibility",
      "Approval",
      "Credits & rebills",
    ]);
    expect(screen.getByRole("tab", { name: "Eligibility", selected: true })).toBeInTheDocument();
  });

  // The server refuses every eligibility policy that adjusts without approval.
  it("never offers adjusting a paid or disputed invoice without approval", async () => {
    served.control = savedInvoiceAdjustmentControl;
    const user = userEvent.setup();

    renderForm();
    await open(user, "Not allowed");

    expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual([
      "Not allowed",
      "With approval",
    ]);
  });

  it("asks about closed periods only while adjustments book on the invoice's date", async () => {
    served.control = savedInvoiceAdjustmentControl;
    const user = userEvent.setup();

    renderForm();
    expect(await screen.findByText("When the period is closed")).toBeVisible();

    await choose(user, "Invoice date while its period is open", "Next open period");

    await waitFor(() => expect(screen.queryByText("When the period is closed")).toBeNull());
  });

  it("says why next-period posting cannot be chosen while nothing needs approval", async () => {
    served.control = {
      ...savedInvoiceAdjustmentControl,
      closedPeriodAdjustmentPolicy: "Disallow",
      standardAdjustmentApprovalPolicy: "None",
    } satisfies InvoiceAdjustmentControl;
    const user = userEvent.setup();

    renderForm();
    await screen.findByText("When the period is closed");
    await user.click(screen.getAllByRole("button", { name: "Not allowed" }).at(-1)!);

    const item = optionItem("Next open period, with approval");
    expect(item).toHaveAttribute("aria-disabled", "true");
    expect(
      within(item as HTMLElement).getByText(
        "Adjustments never need approval. Change that under Approval first.",
      ),
    ).toBeInTheDocument();
  });

  it("says why approval cannot be turned off while closed periods post with approval", async () => {
    served.control = savedInvoiceAdjustmentControl;
    const user = userEvent.setup();

    renderForm("?tab=approval");
    await open(user, "From an amount");

    const item = optionItem("Never");
    expect(item).toHaveAttribute("aria-disabled", "true");
    expect(
      within(item as HTMLElement).getByText(
        "Adjustments to a closed period post in the next open period with approval. Change that under Eligibility first.",
      ),
    ).toBeInTheDocument();
  });

  it("asks for an approval amount only while approval starts from one", async () => {
    served.control = savedInvoiceAdjustmentControl;
    const user = userEvent.setup();

    renderForm("?tab=approval");
    await screen.findByText("Adjustments");
    expect(screen.getAllByRole("textbox", { name: /Approval amount/ })).toHaveLength(2);

    await choose(user, "From an amount", "Always");
    await choose(user, "Approval from an amount", "Not allowed");

    await waitFor(() =>
      expect(screen.queryAllByRole("textbox", { name: /Approval amount/ })).toHaveLength(0),
    );
  });

  it("refuses to save an approval amount of zero", async () => {
    served.control = savedInvoiceAdjustmentControl;
    const user = userEvent.setup();

    renderForm("?tab=approval");
    const [amount] = await screen.findAllByRole("textbox", { name: /Approval amount/ });
    await user.clear(amount);
    await user.type(amount, "0");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(
      await within(amount.closest('[data-slot="setting-row"]') as HTMLElement).findByText(
        "Enter an amount above zero",
      ),
    ).toBeVisible();
    expect(served.sent).toEqual([]);
  });

  it("asks about the rerate tolerance only while changes in amount are reviewed", async () => {
    served.control = savedInvoiceAdjustmentControl;
    const user = userEvent.setup();

    renderForm("?tab=credits");
    expect(await screen.findByRole("textbox", { name: /Rerate tolerance/ })).toBeVisible();

    await choose(user, "When the amount changes", "Always");

    await waitFor(() =>
      expect(screen.queryByRole("textbox", { name: /Rerate tolerance/ })).toBeNull(),
    );
  });

  it("blocks credit beyond what is owed once unapplied credit is not allowed", async () => {
    served.control = {
      ...savedInvoiceAdjustmentControl,
      overCreditPolicy: "AllowWithApproval",
    } satisfies InvoiceAdjustmentControl;
    const user = userEvent.setup();

    renderForm("?tab=credits");
    expect(await screen.findByText("Credit beyond what is owed")).toBeVisible();

    await choose(user, "Allowed", "Not allowed");
    await waitFor(() => expect(screen.queryByText("Credit beyond what is owed")).toBeNull());
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.sent).toHaveLength(1));
    expect(served.sent[0]).toEqual({
      ...savedInvoiceAdjustmentControl,
      customerCreditBalancePolicy: "Disallow",
      overCreditPolicy: "Block",
    } satisfies InvoiceAdjustmentControl);
  });
});

describe("a refused save", () => {
  it("opens and marks Approval when the server refuses the approval amount", async () => {
    served.control = savedInvoiceAdjustmentControl;
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Standard adjustment approval threshold must be greater than zero",
      instance: "/api/v1/invoice-adjustment-controls/",
      code: "INVALID",
      errors: [
        {
          path: ["standardAdjustmentApprovalThreshold"],
          message: "Standard adjustment approval threshold must be greater than zero",
          code: "INVALID_OPERATION",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();

    renderForm("?tab=credits");
    await choose(user, "Current invoice only", "Current and replaced invoices");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const approval = await screen.findByRole("tab", { name: /Approval/, selected: true });
    expect(within(approval).getByText("Has fields to fix")).toBeInTheDocument();
    const [amount] = screen.getAllByRole("textbox", { name: /Approval amount/ });
    expect(
      within(amount.closest('[data-slot="setting-row"]') as HTMLElement).getByText(
        "Standard adjustment approval threshold must be greater than zero",
      ),
    ).toBeVisible();
  });
});
