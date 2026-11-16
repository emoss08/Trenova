import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useBackgroundWrites } from "../use-background-writes";

const mocks = vi.hoisted(() => ({ fetchCurrentSupportSession: vi.fn() }));

vi.mock("@/lib/graphql/support-access", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/support-access")>()),
  fetchCurrentSupportSession: mocks.fetchCurrentSupportSession,
}));

function renderWrites() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(() => useBackgroundWrites(), { wrapper });
}

const session = (scope: "ReadOnly" | "Full") => ({
  staffName: "Jane Doe",
  staffEmail: "jane.doe@trenova.app",
  scope,
  endsAt: 1_800_003_600,
});

describe("useBackgroundWrites", () => {
  beforeEach(() => {
    mocks.fetchCurrentSupportSession.mockReset();
  });

  it("waits until it knows whose session it is", () => {
    mocks.fetchCurrentSupportSession.mockReturnValue(new Promise(() => undefined));

    const { result } = renderWrites();

    expect(result.current).toBe("wait");
  });

  it("sends for a person's session", async () => {
    mocks.fetchCurrentSupportSession.mockResolvedValue(null);

    const { result } = renderWrites();

    await waitFor(() => expect(result.current).toBe("send"));
  });

  it("keeps in memory what a read-only support session would save", async () => {
    mocks.fetchCurrentSupportSession.mockResolvedValue(session("ReadOnly"));

    const { result } = renderWrites();

    await waitFor(() => expect(result.current).toBe("keep"));
  });

  it("sends for a full support session, whose writes the server judges", async () => {
    mocks.fetchCurrentSupportSession.mockResolvedValue(session("Full"));

    const { result } = renderWrites();

    await waitFor(() => expect(result.current).toBe("send"));
  });

  it("sends as before when the session cannot be read", async () => {
    mocks.fetchCurrentSupportSession.mockRejectedValue(new Error("offline"));

    const { result } = renderWrites();

    await waitFor(() => expect(result.current).toBe("send"));
  });
});
