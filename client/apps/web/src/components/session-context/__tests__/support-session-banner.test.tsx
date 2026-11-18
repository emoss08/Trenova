import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SupportSessionBanner } from "../support-session-banner";

const mocks = vi.hoisted(() => ({ fetchCurrentSupportSession: vi.fn() }));

vi.mock("@/lib/graphql/support-access", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/support-access")>()),
  fetchCurrentSupportSession: mocks.fetchCurrentSupportSession,
}));

function renderBanner() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TestRouter>
        <SupportSessionBanner />
      </TestRouter>
    </QueryClientProvider>,
  );
}

describe("SupportSessionBanner", () => {
  beforeEach(() => {
    mocks.fetchCurrentSupportSession.mockReset();
  });

  it("names the staff member and says it is a Trenova support session", async () => {
    mocks.fetchCurrentSupportSession.mockResolvedValue({
      staffName: "Jane Doe",
      staffEmail: "jane.doe@trenova.app",
      scope: "ReadOnly",
      endsAt: 1_800_003_600,
    });

    renderBanner();

    const banner = await screen.findByRole("status");
    expect(banner.textContent).toMatch(/^Trenova support session: Jane Doe, read-only, ends /);
    expect(screen.getByRole("button", { name: "End session" })).toBeTruthy();
  });

  it("says when the session has full access", async () => {
    mocks.fetchCurrentSupportSession.mockResolvedValue({
      staffName: "Jane Doe",
      staffEmail: "jane.doe@trenova.app",
      scope: "Full",
      endsAt: 1_800_003_600,
    });

    renderBanner();

    const banner = await screen.findByRole("status");
    expect(banner.textContent).toMatch(/^Trenova support session: Jane Doe, full access, ends /);
  });

  it("draws nothing in a session that is not Trenova support's", async () => {
    mocks.fetchCurrentSupportSession.mockResolvedValue(null);

    const { container } = renderBanner();

    await vi.waitFor(() => expect(mocks.fetchCurrentSupportSession).toHaveBeenCalled());
    expect(container.innerHTML).toBe("");
  });
});
