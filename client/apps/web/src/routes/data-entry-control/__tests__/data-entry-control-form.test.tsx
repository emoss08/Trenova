import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import DataEntryControlForm from "../_components/data-entry-control-form";
import { savedDataEntryControl } from "./data-entry-control-fixture";

const served = vi.hoisted(() => ({
  refusal: null as unknown,
  sent: [] as unknown[],
}));

vi.mock("@/lib/queries/data-entry-control", () => ({
  dataEntryControl: {
    get: Object.assign(
      () => ({ queryKey: ["data-entry-control"], queryFn: async () => savedDataEntryControl }),
      { _def: ["data-entry-control"] },
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
          <DataEntryControlForm />
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

describe("data entry control form", () => {
  it("shows each kind of text with its saved format, in words, and no tab row", async () => {
    const { container } = renderForm();
    await screen.findByText("Case formatting");

    expect(screen.queryAllByRole("tab")).toHaveLength(0);
    expect(
      [...container.querySelectorAll('[data-slot="setting-row"]')].map(
        (row) => row.querySelector("label")?.textContent,
      ),
    ).toEqual(["Codes", "Names", "Email addresses", "Cities"]);
    expect(screen.getByRole("button", { name: "UPPER" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "lower" })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Title case" })).toHaveLength(2);
    expect(screen.queryByText("TitleCase")).toBeNull();
    expect(screen.queryByText("AsEntered")).toBeNull();
  });

  it("never offers title case for email addresses", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(await screen.findByRole("button", { name: "lower" }));

    expect(await screen.findAllByText("As entered")).not.toHaveLength(0);
    expect(screen.getAllByText("Title case")).toHaveLength(2);
  });

  it("sends the whole record with the format that changed", async () => {
    const user = userEvent.setup();
    renderForm();

    await choose(user, "UPPER", "As entered");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.sent).toHaveLength(1));
    expect(served.sent[0]).toEqual({ ...savedDataEntryControl, codeCase: "AsEntered" });
  });

  it("puts a refused format's reason on its own field", async () => {
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    served.refusal = new ApiRequestError(400, {
      type: "https://api.trenova.app/problems/validation-error",
      title: "Validation Failed",
      status: 400,
      detail: "City case is required",
      instance: "/api/v1/data-entry-controls/",
      code: "INVALID",
      errors: [
        {
          path: ["cityCase"],
          message: "City case is required",
          code: "REQUIRED",
          location: "body",
        },
      ],
    } as ConstructorParameters<typeof ApiRequestError>[1]);
    const user = userEvent.setup();
    const { container } = renderForm();

    await choose(user, "lower", "UPPER");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const cities = [...container.querySelectorAll('[data-slot="setting-row"]')].at(-1)!;
    await waitFor(() => expect(cities).toHaveTextContent("City case is required"));
  });
});
