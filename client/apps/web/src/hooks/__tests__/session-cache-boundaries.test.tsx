import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useSwitchOrganization } from "../use-organization-switch";
import { useSignOut } from "../use-sign-out";

// The order of what happens as a session ends is the point: the cache must be gone before
// the next screen can draw from it.
const calls: string[] = [];

const mocks = vi.hoisted(() => ({
  logout: vi.fn(),
  setUser: vi.fn(),
  navigate: vi.fn(),
  clearSessionCache: vi.fn(),
  switchOrganization: vi.fn(),
  fetchManifest: vi.fn(),
  clearPermissions: vi.fn(),
}));

vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => mocks.navigate,
}));

vi.mock("@/lib/session-cache", () => ({ clearSessionCache: mocks.clearSessionCache }));

vi.mock("@/components/assistant/turn-readers", () => ({ releaseTurnReaders: vi.fn() }));

vi.mock("@trenova/shared/stores/auth-store", () => {
  const state = { logout: mocks.logout, setUser: mocks.setUser };
  const useAuthStore = (selector?: (s: typeof state) => unknown) =>
    selector ? selector(state) : state;
  useAuthStore.getState = () => state;
  return { useAuthStore };
});

vi.mock("@trenova/shared/stores/permission-store", () => ({
  usePermissionStore: {
    getState: () => ({
      fetchManifest: mocks.fetchManifest,
      clearPermissions: mocks.clearPermissions,
    }),
  },
}));

vi.mock("@/services/api", () => ({
  apiService: { userService: { switchOrganization: mocks.switchOrganization } },
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  calls.length = 0;
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.logout.mockImplementation(async () => void calls.push("logout"));
  mocks.clearSessionCache.mockImplementation(async () => void calls.push("clear cache"));
  mocks.navigate.mockImplementation((to: string) => void calls.push(`navigate ${to}`));
  mocks.fetchManifest.mockResolvedValue(undefined);
  mocks.switchOrganization.mockResolvedValue({ id: "usr_1", currentOrganizationId: "org_2" });
});

describe("ending or changing a session", () => {
  it("signing out clears the cache, on disk too, before showing the sign-in page", async () => {
    const { result } = renderHook(() => useSignOut(), { wrapper });

    await act(() => result.current());

    expect(calls).toEqual(["logout", "clear cache", "navigate /login"]);
  });

  it("switching organization clears the cache before opening the new organization", async () => {
    const { result } = renderHook(() => useSwitchOrganization(), { wrapper });

    act(() => result.current.mutate({ organizationId: "org_2" }));

    await waitFor(() => expect(calls).toEqual(["clear cache", "navigate /"]));
    expect(mocks.setUser).toHaveBeenCalledWith({ id: "usr_1", currentOrganizationId: "org_2" });
  });
});
