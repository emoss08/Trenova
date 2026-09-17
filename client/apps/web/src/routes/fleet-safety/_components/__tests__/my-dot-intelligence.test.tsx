import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MyDotIntelligence } from "../my-dot-intelligence";

const mocks = vi.hoisted(() => ({
  allowed: true,
  fetchMyCarrierIntelligence: vi.fn(),
  fetchSettings: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed, isLoading: false }),
}));

vi.mock("@/lib/graphql/carrier-intelligence", () => ({
  MY_CARRIER_INTELLIGENCE_KEY: "my-carrier-intelligence",
  fetchMyCarrierIntelligence: mocks.fetchMyCarrierIntelligence,
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    carrierIntelSettings: {
      settings: () => ({
        queryKey: ["carrierIntelSettings", "settings"],
        queryFn: mocks.fetchSettings,
      }),
    },
  },
}));

afterEach(() => {
  vi.clearAllMocks();
  mocks.allowed = true;
});

function settings(configured: boolean) {
  return {
    carrierIntelControl: null,
    carrierIntelProvider: {
      configured,
      provider: configured ? "FMCSAQCMobile" : null,
      fallbackProvider: null,
      capabilities: [],
      sections: [],
    },
    carrierIntelRuleCatalog: [],
  };
}

function renderMyDot() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MyDotIntelligence />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("MyDotIntelligence", () => {
  it("asks for the organization's DOT number when none is on file", async () => {
    mocks.fetchMyCarrierIntelligence.mockResolvedValue({
      configured: false,
      dotNumber: null,
      snapshot: null,
    });
    mocks.fetchSettings.mockResolvedValue(settings(true));
    renderMyDot();

    expect(await screen.findByText("Your organization has no DOT number")).toBeInTheDocument();
    expect(screen.getByText("Open organization settings").closest("a")).toHaveAttribute(
      "href",
      "/admin/organization-settings",
    );
    expect(mocks.fetchMyCarrierIntelligence).toHaveBeenCalledWith(false, expect.anything());
  });

  it("links to integrations when no provider is connected", async () => {
    mocks.fetchMyCarrierIntelligence.mockResolvedValue({
      configured: true,
      dotNumber: "1234567",
      snapshot: null,
    });
    mocks.fetchSettings.mockResolvedValue(settings(false));
    renderMyDot();

    expect(
      await screen.findByText("No carrier intelligence provider is connected"),
    ).toBeInTheDocument();
    expect(screen.getByText("Open integrations").closest("a")).toHaveAttribute(
      "href",
      "/admin/integrations?category=CarrierCompliance",
    );
  });

  it("offers to pull the profile when it has never been fetched", async () => {
    mocks.fetchMyCarrierIntelligence.mockResolvedValue({
      configured: true,
      dotNumber: "1234567",
      snapshot: null,
    });
    mocks.fetchSettings.mockResolvedValue(settings(true));
    renderMyDot();

    expect(await screen.findByText("Your DOT profile has not been pulled yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Pull profile/ })).toBeInTheDocument();
  });

  it("is restricted without carrier intelligence read access", () => {
    mocks.allowed = false;
    renderMyDot();

    expect(screen.getByText("Your DOT profile is restricted")).toBeInTheDocument();
    expect(mocks.fetchMyCarrierIntelligence).not.toHaveBeenCalled();
  });
});
