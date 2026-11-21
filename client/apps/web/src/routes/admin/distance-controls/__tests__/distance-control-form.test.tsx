import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import DistanceControlForm from "../_components/distance-control-form";
import { savedDistanceControl } from "./distance-control-fixture";

const served = vi.hoisted(() => ({
  control: null as unknown,
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/distance-control", () => ({
  distanceControl: {
    get: Object.assign(
      () => ({ queryKey: ["distance-control"], queryFn: async () => served.control }),
      { _def: ["distance-control"] },
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

function renderForm() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <TestRouter>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <DistanceControlForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

const rowLabels = (container: HTMLElement) =>
  [...container.querySelectorAll('[data-slot="setting-row"]')].map(
    (row) => row.querySelector("label")?.textContent,
  );

const rowNamed = (container: HTMLElement, label: string) =>
  [...container.querySelectorAll('[data-slot="setting-row"]')].find(
    (row) => row.querySelector("label")?.textContent === label,
  )!;

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

afterEach(() => {
  served.control = savedDistanceControl;
  served.refusal = null;
  served.sent = [];
  cleanup();
});

served.control = savedDistanceControl;

describe("distance control form", () => {
  it("draws every setting in one tab with no tab row, and the unit in words", async () => {
    const { container } = renderForm();
    await screen.findByText("Stored mileage");

    expect(screen.queryAllByRole("tab")).toHaveLength(0);
    expect(rowLabels(container)).toEqual([
      "Loaded moves",
      "Empty moves",
      "Arrival estimates",
      "Pay",
      "Billing",
      "Fuel",
      "Reuse stored mileage",
      "Keep new mileage",
      "Match by city",
      "Capture jurisdiction miles",
      "Distance unit",
    ]);
    expect(screen.getByRole("button", { name: "Miles" })).toBeInTheDocument();
    expect(screen.queryByText("dp_practical")).toBeNull();
  });

  it("offers city matching only while stored mileage is reused or kept", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(await screen.findByRole("switch", { name: /Reuse stored mileage/ }));
    expect(screen.getByRole("switch", { name: /Match by city/ })).toBeInTheDocument();

    await user.click(screen.getByRole("switch", { name: /Keep new mileage/ }));
    await waitFor(() => expect(screen.queryByRole("switch", { name: /Match by city/ })).toBeNull());

    await user.click(screen.getByRole("switch", { name: /Keep new mileage/ }));
    expect(await screen.findByRole("switch", { name: /Match by city/ })).toBeInTheDocument();
  });

  it("sends the whole record with the setting that changed", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(await screen.findByRole("switch", { name: /Capture jurisdiction miles/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.sent).toHaveLength(1));
    expect(served.sent[0]).toEqual({ ...savedDistanceControl, captureJurisdictionMiles: true });
  });

  it("keeps city matching as saved while it is hidden", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(await screen.findByRole("switch", { name: /Reuse stored mileage/ }));
    await user.click(screen.getByRole("switch", { name: /Keep new mileage/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.sent).toHaveLength(1));
    expect(served.sent[0]).toEqual({
      ...savedDistanceControl,
      storeMileage: false,
      autoCreateStoredMileage: false,
      postalCodeFallbackToCity: true,
    });
  });

  it("refuses to send a record with a purpose that has no profile, and says so on its row", async () => {
    served.control = { ...savedDistanceControl, fuelDistanceProfileId: "" };
    const user = userEvent.setup();
    const { container } = renderForm();

    await user.click(await screen.findByRole("switch", { name: /Capture jurisdiction miles/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() =>
      expect(rowNamed(container, "Fuel")).toHaveTextContent("Fuel distance profile is required"),
    );
    expect(served.sent).toEqual([]);
  });

  it("puts the server's refusal of a profile on its own row", async () => {
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "Distance profile must be an active PC*Miler profile",
      instance: "/api/v1/distance-controls/",
      code: "INVALID",
      errors: [
        {
          path: ["billingDistanceProfileId"],
          message: "Distance profile must be an active PC*Miler profile",
          code: "INVALID",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();
    const { container } = renderForm();

    await user.click(await screen.findByRole("switch", { name: /Capture jurisdiction miles/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() =>
      expect(rowNamed(container, "Billing")).toHaveTextContent(
        "Distance profile must be an active PC*Miler profile",
      ),
    );
  });
});
