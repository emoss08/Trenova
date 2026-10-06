import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthPanel } from "./auth-panel";

const mocks = vi.hoisted(() => ({
  getVersion: vi.fn(),
}));

vi.mock("@/services/update", () => ({
  updateService: {
    getVersion: mocks.getVersion,
  },
}));

vi.mock("@/lib/edition", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/edition")>();
  return {
    ...actual,
    edition: {
      ...actual.edition,
      slots: { ...actual.edition.slots, AuthAmbient: () => <p>edition ambient</p> },
    },
  };
});

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <AuthPanel receipt={{ issued: false, rows: [{ key: "Identity" }] }} />
    </QueryClientProvider>,
  );
}

describe("AuthPanel", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.getVersion.mockResolvedValue({ version: "4.12.0", environment: "development" });
  });

  it("renders the edition's ambient slot under the headline", async () => {
    renderPanel();

    await waitFor(() => expect(screen.getByText("Network operational")).toBeInTheDocument());
    expect(screen.getByText("edition ambient")).toBeInTheDocument();
    expect(screen.getByText("v4.12.0")).toBeInTheDocument();
  });

  it("reports the network as unreachable when the version call fails", async () => {
    mocks.getVersion.mockRejectedValue(new Error("offline"));
    renderPanel();

    await waitFor(() => expect(screen.getByText("Network unreachable")).toBeInTheDocument());
  });
});
