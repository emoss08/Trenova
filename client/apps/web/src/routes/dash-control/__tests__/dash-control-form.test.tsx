import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { seedPlan, unavailableFeatures } from "@/test/plan";
import type { DashControl } from "@/types/dash-control";
import DashControlForm from "../_components/dash-control-form";
import { savedDashControl } from "./dash-control-fixture";

const served = vi.hoisted(() => ({
  control: null as unknown,
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/dash-control", () => ({
  dashControl: {
    get: Object.assign(
      () => ({ queryKey: ["dash-control", "get"], queryFn: async () => served.control }),
      { _def: ["dash-control", "get"] },
    ),
  },
}));

// The save records what was sent and hands a refusal to the form, as the real mutation does.
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
          <DashControlForm />
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
  served.control = savedDashControl;
  served.refusal = null;
  served.sent = [];
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

served.control = savedDashControl;

describe("dash control tabs", () => {
  it("gives a full plan every tab, loads first", async () => {
    renderForm();
    await screen.findByText("Assignments");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Loads",
      "Pay & expenses",
      "Profile & time off",
      "Reminders",
    ]);
  });

  it("leaves reminders out when the plan has neither credentials nor training", async () => {
    seedPlan({}, unavailableFeatures("workforce.compliance", "workforce.talent"));

    renderForm();
    await screen.findByText("Assignments");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Loads",
      "Pay & expenses",
      "Profile & time off",
    ]);
    expect(screen.queryByText("Remind drivers")).toBeNull();
  });

  it("keeps reminders when the plan has training but not credentials", async () => {
    seedPlan({}, unavailableFeatures("workforce.compliance"));

    renderForm();
    await screen.findByText("Assignments");

    expect(screen.getByRole("tab", { name: "Reminders" })).toBeInTheDocument();
  });
});

describe("dash control rows that wait on another", () => {
  it.each([
    ["Accept assignments", "Allow declines", ""],
    ["Detention alerts", "Threshold", ""],
    ["Pay per load", "Pay estimates", "?tab=pay"],
    ["Expense claims", "Require a receipt", "?tab=pay"],
    ["Contact details", "Approve changes", "?tab=profile"],
    ["Remind drivers", "How often", "?tab=reminders"],
  ])("opens %s's %s row with it", async (parent, child, search) => {
    const user = userEvent.setup();

    renderForm(search);
    await screen.findByText("Assignments");
    expect(screen.queryByText(child)).toBeNull();

    await user.click(screen.getByRole("switch", { name: new RegExp(parent) }));

    expect(await screen.findByText(child)).toBeInTheDocument();
  });

  it("asks for the round-up day only for a weekly round-up", async () => {
    served.control = { ...savedDashControl, sendCredentialReminders: true } satisfies DashControl;
    const user = userEvent.setup();

    renderForm("?tab=reminders");
    await screen.findByText("How often");
    expect(screen.queryByText("Round-up day")).toBeNull();

    await user.click(screen.getByRole("button", { name: "One notice per item" }));
    await user.click(screen.getAllByText("Weekly round-up").at(-1)!);

    expect(await screen.findByText("Round-up day")).toBeInTheDocument();
  });
});

describe("saving", () => {
  it("sends a hidden setting the way the server needs it", async () => {
    served.control = {
      ...savedDashControl,
      requireLoadAcknowledgment: true,
      allowLoadRefusals: true,
    } satisfies DashControl;
    const user = userEvent.setup();

    renderForm();
    await user.click(await screen.findByRole("switch", { name: /Accept assignments/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(served.sent).toEqual([
      { ...savedDashControl, requireLoadAcknowledgment: false, allowLoadRefusals: false },
    ]);
  });

  it("opens and marks Loads when the server refuses one of its settings", async () => {
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Stop actions are not available",
      instance: "/graphql",
      code: "INVALID",
      errors: [
        {
          path: ["allowStopActions"],
          message: "Stop actions are not available",
          code: "INVALID",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();

    renderForm("?tab=profile");
    await user.click(await screen.findByRole("switch", { name: /Time-off requests/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const loads = await screen.findByRole("tab", { name: /Loads/, selected: true });
    expect(within(loads).getByText("Has fields to fix")).toBeInTheDocument();
    expect(screen.getAllByText("Stop actions are not available").at(-1)).toBeVisible();
  });
});
