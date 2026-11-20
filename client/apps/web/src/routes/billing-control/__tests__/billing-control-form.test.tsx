import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import BillingControlForm from "../_components/billing-control-form";
import { savedBillingControl } from "./billing-control-fixture";

const served = vi.hoisted(() => ({
  control: null as unknown,
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/billing-control", () => ({
  billingControl: {
    get: Object.assign(
      () => ({ queryKey: ["billing-control"], queryFn: async () => served.control }),
      { _def: ["billing-control"] },
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

function serve(changes: Record<string, unknown> = {}) {
  served.control = { ...savedBillingControl, ...changes };
}

function renderForm(search = "") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <TestRouter search={search}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <BillingControlForm />
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
  served.refusal = null;
  served.sent = [];
  cleanup();
});

describe("billing control automation", () => {
  it("asks how often and how many only once shipments move to the queue on their own", async () => {
    serve();
    const user = userEvent.setup();

    renderForm("?tab=automation");
    expect(await screen.findByText("Ready to bill")).toBeInTheDocument();
    expect(screen.queryByText("Shipments per run")).toBeNull();

    const queue = screen
      .getByText("Move to the billing queue")
      .closest('[data-slot="setting-row"]');
    await user.click(within(queue as HTMLElement).getByRole("button"));
    await user.click(screen.getAllByText("Automatic when ready").at(-1)!);

    expect(await screen.findByText("How often")).toBeInTheDocument();
    expect(screen.getByText("Shipments per run")).toBeInTheDocument();
  });

  it("puts posting back to review when drafts go back to being made by hand", async () => {
    serve({
      invoiceDraftCreationMode: "AutomaticWhenTransferred",
      autoInvoiceBatchSize: 25,
      invoicePostingMode: "AutomaticWhenNoBlockingExceptions",
    });
    const user = userEvent.setup();

    renderForm("?tab=automation");
    expect(await screen.findByText("Drafts per run")).toBeInTheDocument();
    await choose(user, "Automatic when transferred", "Manual only");

    expect(
      await screen.findByRole("button", { name: "Manual review required" }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Drafts per run")).toBeNull();
  });
});

describe("billing control approval", () => {
  it("says an item waits for Nova's agreement whenever the server reads that mode", async () => {
    serve({ queueAutoApproveMode: "WhenAgentAgrees" });

    renderForm("?tab=automation");

    expect(
      await screen.findByText(
        "An item is approved without a biller once all five checks clear and Nova agrees with the charges.",
      ),
    ).toBeInTheDocument();
  });

  it("says every item waits for a biller when the server reads the mode as off", async () => {
    serve({ queueAutoApproveMode: "Off" });

    renderForm("?tab=automation");

    expect(
      await screen.findByText("Every item waits for a biller to approve it."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Off" })).toBeInTheDocument();
  });
});

describe("billing control exceptions", () => {
  it("offers skipping review, then its tolerance, only while rate validation requires review", async () => {
    serve();
    const user = userEvent.setup();

    renderForm("?tab=exceptions");
    expect(await screen.findByText("Checks")).toBeInTheDocument();
    expect(screen.queryByText("Skip review for small differences")).toBeNull();
    expect(screen.queryByText("Send for review to")).toBeNull();

    const rate = screen.getByText("Rate validation").closest('[data-slot="setting-row"]');
    await user.click(within(rate as HTMLElement).getByRole("button"));
    await user.click(screen.getAllByText("Require review").at(-1)!);

    expect(await screen.findByText("Skip review for small differences")).toBeInTheDocument();
    expect(screen.getByText("Send for review to")).toBeInTheDocument();
    expect(screen.queryByText("Tolerance")).toBeNull();

    await choose(user, "Disabled", "Bypass review within tolerance");
    expect(await screen.findByText("Tolerance")).toBeInTheDocument();
  });
});

describe("billing control collections", () => {
  it("asks for the reminder days once reminders are on, and keeps them as numbers", async () => {
    serve();
    const user = userEvent.setup();

    renderForm("?tab=collections");
    expect(await screen.findByText("Payment reminders")).toBeInTheDocument();
    expect(screen.queryByLabelText(/Remind on days past due/)).toBeNull();

    await user.click(screen.getByRole("switch", { name: /Send payment reminders/ }));
    const days = await screen.findByLabelText(/Remind on days past due/);
    expect(days).toHaveValue("1, 15, 30");

    await user.clear(days);
    await user.type(days, "45, 7, 21");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(served.sent).toHaveLength(1);
    expect(served.sent[0]).toMatchObject({
      paymentRemindersEnabled: true,
      paymentReminderDays: [7, 21, 45],
    });
  });

  it("says when the reminder days are not whole numbers", async () => {
    serve({ paymentRemindersEnabled: true });
    const user = userEvent.setup();

    renderForm("?tab=collections");
    const days = await screen.findByLabelText(/Remind on days past due/);
    await user.clear(days);
    await user.type(days, "1, 2.5");

    expect(
      screen.getByText("Enter whole days separated by commas, such as 1, 15, 30"),
    ).toBeInTheDocument();
  });

  it("opens Automation from the note that raised late charges wait for posting", async () => {
    serve({ lateChargeAssessmentMode: "Automatic" });
    const user = userEvent.setup();

    renderForm("?tab=collections");
    await user.click(await screen.findByRole("button", { name: "Open Automation" }));

    expect(await screen.findByRole("tab", { name: /Automation/, selected: true })).toBeVisible();
  });

  it("sends hidden draft settings the way the server requires them", async () => {
    serve({ autoInvoiceBatchSize: 40, notifyOnAutoInvoiceCreation: true });
    const user = userEvent.setup();

    renderForm("?tab=collections");
    await user.click(await screen.findByRole("switch", { name: /Send monthly statements/ }));
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(served.sent[0]).toMatchObject({
      monthlyStatementsEnabled: true,
      autoInvoiceBatchSize: null,
      notifyOnAutoInvoiceCreation: false,
    });
  });
});

describe("a refused save", () => {
  it("opens and marks Rating when the server refuses the fallback template", async () => {
    serve();
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Choose how unrated shipments are handled",
      instance: "/api/v1/billing-controls/",
      code: "INVALID",
      errors: [
        {
          path: ["unratedShipmentDisposition"],
          message: "Choose how unrated shipments are handled",
          code: "INVALID",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();

    renderForm("?tab=collections");
    await user.click(await screen.findByRole("switch", { name: /Send monthly statements/ }));
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    const rating = await screen.findByRole("tab", { name: /Rating/, selected: true });
    expect(within(rating).getByText("Has fields to fix")).toBeInTheDocument();
    expect(screen.getAllByText("Choose how unrated shipments are handled").length).toBeGreaterThan(
      0,
    );
  });

  it("opens Automation when a setting there fails the form's own check", async () => {
    serve({ billingQueueTransferMode: "AutomaticWhenReady", billingQueueTransferBatchSize: 0 });
    const user = userEvent.setup();

    renderForm("?tab=collections");
    await user.click(await screen.findByRole("switch", { name: /Send monthly statements/ }));
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByRole("tab", { name: /Automation/, selected: true })).toBeVisible();
    expect(served.sent).toEqual([]);
    expect(screen.getByText("Move at least one shipment per batch")).toBeInTheDocument();
  });
});
