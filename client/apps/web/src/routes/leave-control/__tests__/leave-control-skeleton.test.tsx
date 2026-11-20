import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import LeaveControlForm from "../_components/leave-control-form";
import { LeaveControlSkeleton } from "../skeleton";
import { savedLeaveControl } from "./leave-control-fixture";

vi.mock("@/lib/queries/leave-control", () => ({
  leaveControl: {
    get: Object.assign(
      () => ({ queryKey: ["leave-control"], queryFn: async () => savedLeaveControl }),
      { _def: ["leave-control"] },
    ),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

type Outline = { tabs: number; sections: number; rows: number };

describe("LeaveControlSkeleton", () => {
  afterEach(cleanup);

  // The fallback has to be the page the form draws, or the layout jumps when the form lands.
  it("draws the form's sections and rows, with no tab row", async () => {
    const { container: skeleton } = render(
      <TestRouter>
        <LeaveControlSkeleton />
      </TestRouter>,
    );
    const drawn: Outline = {
      tabs: skeleton.querySelectorAll('[data-slot="tab-skeleton"]').length,
      sections: skeleton.querySelectorAll('[data-slot="settings-section-skeleton"]').length,
      rows: skeleton.querySelectorAll('[data-slot="field-skeleton"]').length,
    };
    expect(drawn).toEqual({ tabs: 0, sections: 4, rows: 7 });
    cleanup();

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { container: form } = render(
      <TestRouter>
        <QueryClientProvider client={queryClient}>
          <LeaveControlForm />
        </QueryClientProvider>
      </TestRouter>,
    );
    await screen.findByText("Leave year");

    expect({
      tabs: form.querySelectorAll('[role="tab"]').length,
      sections: form.querySelectorAll('[data-slot="settings-section"]').length,
      rows: form.querySelectorAll('[data-slot="setting-row"]').length,
    }).toEqual(drawn);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <LeaveControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
