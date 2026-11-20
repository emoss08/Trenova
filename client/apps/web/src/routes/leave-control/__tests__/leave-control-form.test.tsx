import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { LeaveControl } from "@trenova/shared/types/worker-leave";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import LeaveControlForm from "../_components/leave-control-form";
import { savedLeaveControl } from "./leave-control-fixture";

const served = vi.hoisted(() => ({
  control: null as unknown,
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/leave-control", () => ({
  leaveControl: {
    get: Object.assign(
      () => ({ queryKey: ["leave-control"], queryFn: async () => served.control }),
      { _def: ["leave-control"] },
    ),
  },
}));

vi.mock("@/lib/graphql/worker-leave", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/worker-leave")>()),
  updateLeaveControl: async (record: LeaveControl) => {
    served.sent.push(record);
    return { ...record, version: record.version + 1 };
  },
}));

// The save runs the page's own save and hands a refusal to the form, as the real mutation does.
vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: ({
    form,
    mutationFn,
  }: {
    form: unknown;
    mutationFn: (values: unknown) => Promise<unknown>;
  }) => ({
    mutateAsync: async (values: unknown) => {
      if (!served.refusal) return mutationFn(values);
      const { handleMutationError } = await vi.importActual<
        typeof import("@/hooks/use-api-mutation")
      >("@/hooks/use-api-mutation");
      handleMutationError({ error: served.refusal, form: form as never });
      throw served.refusal;
    },
  }),
}));

function renderForm() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidate = vi.spyOn(queryClient, "invalidateQueries");

  render(
    <TestRouter>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <LeaveControlForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );

  return { invalidate };
}

function rowOf(element: HTMLElement) {
  return within(element.closest('[data-slot="setting-row"]') as HTMLElement);
}

const STACKING_WARNING =
  "An employee can take two years' leave back to back across the turn of the year.";

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

afterEach(() => {
  served.control = savedLeaveControl;
  served.refusal = null;
  served.sent = [];
  cleanup();
});

describe("leave control form", () => {
  it("draws the policy without a tab row", async () => {
    served.control = savedLeaveControl;

    renderForm();
    await screen.findByText("Leave year");

    expect(screen.queryAllByRole("tab")).toEqual([]);
    expect(screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent)).toEqual([
      "Leave year",
      "Entitlement",
      "Eligibility",
      "Certification",
    ]);
  });

  // Only the calendar year and the hire anniversary let two years' leave be taken in a row.
  it.each([
    ["CalendarYear", true],
    ["HireAnniversary", true],
    ["ForwardFromFirstUse", false],
    ["RollingBackward", false],
  ] as const)("warns about back-to-back leave under %s: %s", async (method, warns) => {
    served.control = { ...savedLeaveControl, measurementMethod: method };

    renderForm();
    await screen.findByText("Leave year");

    expect(screen.queryByText(STACKING_WARNING) !== null).toBe(warns);
  });

  it("saves the whole record, decimals as typed, and rereads every worker's leave", async () => {
    served.control = savedLeaveControl;
    const user = userEvent.setup();

    const { invalidate } = renderForm();
    const workweek = await screen.findByRole("textbox", { name: /Workweek/ });
    await user.clear(workweek);
    await user.type(workweek, "37.5");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.sent).toHaveLength(1));
    expect(served.sent[0]).toEqual({
      ...savedLeaveControl,
      workweekHours: "37.5",
    } satisfies LeaveControl);
    await waitFor(() => expect(invalidate).toHaveBeenCalledWith({ queryKey: ["worker-leave"] }));
  });

  it("refuses military caregiver leave shorter than the entitlement on its own row", async () => {
    served.control = savedLeaveControl;
    const user = userEvent.setup();

    renderForm();
    const weeks = await screen.findByRole("textbox", { name: /^Leave/ });
    await user.clear(weeks);
    await user.type(weeks, "30");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const caregiver = screen.getByRole("textbox", { name: /Military caregiver leave/ });
    expect(
      await rowOf(caregiver).findByText(
        "Military caregiver leave cannot be shorter than the ordinary entitlement",
      ),
    ).toBeVisible();
    expect(served.sent).toEqual([]);
  });
});

describe("a refused save", () => {
  it("puts the server's message on the field it names", async () => {
    served.control = savedLeaveControl;
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Certification must be given at least a day",
      instance: "/graphql",
      code: "INVALID",
      errors: [
        {
          path: ["certificationDueDays"],
          message: "Certification must be given at least a day",
          code: "INVALID",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();

    renderForm();
    const months = await screen.findByRole("textbox", { name: /Service/ });
    await user.clear(months);
    await user.type(months, "6");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const days = screen.getByRole("textbox", { name: /Time to return it/ });
    expect(
      await rowOf(days).findByText("Certification must be given at least a day"),
    ).toBeVisible();
  });
});
