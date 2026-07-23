import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useSavedViewCounts } from "./use-view-counts";

const getShipmentSavedViewCountsGraphQLMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql/shipment", () => ({
  getShipmentSavedViewCountsGraphQL: getShipmentSavedViewCountsGraphQLMock,
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useSavedViewCounts", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("loads saved view counts with one analytics query", async () => {
    getShipmentSavedViewCountsGraphQLMock.mockResolvedValue({
      all: 10,
      transit: 4,
      atRisk: 2,
      unassigned: 3,
      deliveringToday: 1,
    });
    useAuthStore.setState({
      user: { timezone: "America/Chicago" } as User,
      isAuthenticated: true,
    });

    const { result } = renderHook(() => useSavedViewCounts(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current).toEqual({
        all: 10,
        transit: 4,
        "at-risk": 2,
        unassigned: 3,
        "delivering-today": 1,
      });
    });
    expect(getShipmentSavedViewCountsGraphQLMock).toHaveBeenCalledTimes(1);
    expect(getShipmentSavedViewCountsGraphQLMock).toHaveBeenCalledWith("America/Chicago");
  });
});
