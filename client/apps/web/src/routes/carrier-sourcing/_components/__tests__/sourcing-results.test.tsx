import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import type { CarrierSourcingPage } from "@/lib/graphql/carrier-sourcing";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CarrierSourcingWorkspace } from "../carrier-sourcing-workspace";
import { SourcingResults } from "../sourcing-results";

const mocks = vi.hoisted(() => ({
  fetchCarrierIntelSettings: vi.fn(),
  searchCarrierSourcing: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-intelligence", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-intelligence")>()),
  fetchCarrierIntelSettings: mocks.fetchCarrierIntelSettings,
}));
vi.mock("@/lib/graphql/carrier-sourcing", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-sourcing")>()),
  searchCarrierSourcing: mocks.searchCarrierSourcing,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

afterEach(() => {
  vi.clearAllMocks();
});

function profile(overrides: Partial<CarrierIntelProfile> = {}): CarrierIntelProfile {
  return {
    coverage: ["Identity", "Authority", "Fleet"],
    identity: {
      dotNumber: "1234567",
      docketPrefix: "MC",
      docketNumber: "765432",
      legalName: "Blue Ridge Freight LLC",
      dbaName: null,
      ein: null,
      usdotStatus: "Active",
      entityType: "Carrier",
      carrierOperation: "Interstate",
      dotAddedAt: null,
      dotAgeDays: 2400,
      physicalAddress: {
        line1: "1 Main St",
        city: "Asheville",
        state: "NC",
        postalCode: "28801",
        country: "US",
        undelivered: false,
      },
      mailingAddress: null,
    },
    authority: {
      common: {
        status: "Active",
        pending: false,
        underReview: false,
        revocationPending: false,
        grantedAt: null,
        ageDays: 1800,
      },
      contract: null,
      broker: null,
      totalRevocations: 0,
      lastRevocationAt: null,
      history: null,
    },
    insurance: null,
    safety: null,
    basics: null,
    inspections: null,
    crashes: null,
    fleet: {
      powerUnits: 42,
      drivers: 50,
      cdlDrivers: 50,
      ownedTractors: null,
      termLeasedTractors: null,
      ownedTrailers: null,
      termLeasedTrailers: null,
      trailers: null,
      trucks: null,
    },
    equipment: null,
    contacts: null,
    operations: null,
    changeHistory: null,
    network: null,
    lanes: null,
    benchmarks: null,
    ...overrides,
  };
}

function page(items: CarrierSourcingPage["items"]): CarrierSourcingPage {
  return { total: items.length, provider: "CarrierOK", items };
}

function Providers({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter>
        <MemoryRouter>{children}</MemoryRouter>
      </NuqsTestingAdapter>
    </QueryClientProvider>
  );
}

function renderResults(results: CarrierSourcingPage) {
  const onImport = vi.fn();
  render(
    <Providers>
      <SourcingResults
        page={results}
        isLoading={false}
        isFetching={false}
        error={null}
        offset={0}
        pageSize={25}
        canImport
        ruleLabels={{}}
        onImport={onImport}
        onPageChange={vi.fn()}
        onRetry={vi.fn()}
      />
    </Providers>,
  );
  return { onImport };
}

describe("SourcingResults", () => {
  it("links a carrier already in Trenova to its panel instead of offering an import", () => {
    renderResults(
      page([
        {
          providerRef: "ref_1",
          dotNumber: "1234567",
          legalName: "Blue Ridge Freight LLC",
          existingCarrierId: "car_existing",
          laneMatches: 12,
          riskLevel: "Low",
          score: 102,
          findings: [],
          profile: profile(),
        },
        {
          providerRef: "ref_2",
          dotNumber: "7654321",
          legalName: "Piedmont Haulers",
          existingCarrierId: null,
          laneMatches: 0,
          riskLevel: "Elevated",
          score: 62,
          findings: [
            {
              code: "authority.too_new",
              category: "Authority",
              action: "Block",
              severity: "High",
              message: "Operating authority is younger than 180 days",
              unverifiable: false,
              unconfirmed: false,
              overridden: false,
              overrideId: null,
              overrideExpiresAt: null,
            },
          ],
          profile: profile({
            identity: {
              ...profile().identity!,
              dotNumber: "7654321",
              legalName: "Piedmont Haulers",
            },
          }),
        },
      ]),
    );

    const [existing, prospect] = screen.getAllByTestId("sourcing-result");

    const link = within(existing).getByTestId("existing-carrier-link");
    expect(link.textContent).toContain("Already in Trenova");
    expect(link.getAttribute("href")).toBe(
      "/dispatch/carriers?panelType=edit&panelEntityId=car_existing&tab=intelligence",
    );
    expect(within(existing).queryByRole("button", { name: /^import$/i })).toBeNull();
    expect(within(existing).getByText("Blue Ridge Freight LLC")).toBeTruthy();
    expect(within(existing).getByText(/12 lane load matches/i)).toBeTruthy();

    expect(within(prospect).queryByTestId("existing-carrier-link")).toBeNull();
    expect(within(prospect).getByRole("button", { name: /^import$/i })).toBeTruthy();
    expect(within(prospect).getByText("Operating authority is younger than 180 days")).toBeTruthy();
  });

  it("explains an empty page when the provider matched carriers the filters removed", () => {
    render(
      <Providers>
        <SourcingResults
          page={{ total: 40, provider: "CarrierOK", items: [] }}
          isLoading={false}
          isFetching={false}
          error={null}
          offset={0}
          pageSize={25}
          canImport
          ruleLabels={{}}
          onImport={vi.fn()}
          onPageChange={vi.fn()}
          onRetry={vi.fn()}
        />
      </Providers>,
    );

    expect(screen.getByText("Nothing matches")).toBeTruthy();
    expect(screen.getByRole("button", { name: /next/i })).toBeTruthy();
  });
});

describe("CarrierSourcingWorkspace", () => {
  it("shows the unsupported search state when the provider cannot search", async () => {
    mocks.fetchCarrierIntelSettings.mockResolvedValue({
      carrierIntelControl: {},
      carrierIntelRuleCatalog: [],
      carrierIntelProvider: {
        configured: true,
        provider: "FMCSAQCMobile",
        fallbackProvider: null,
        capabilities: ["LookupFMCSA"],
        sections: ["Identity", "Authority", "Safety"],
      },
    });

    render(
      <Providers>
        <CarrierSourcingWorkspace />
      </Providers>,
    );

    const unsupported = await screen.findByTestId("sourcing-search-unsupported");
    expect(within(unsupported).getByText("Search not supported by FMCSA QCMobile")).toBeTruthy();
    expect(within(unsupported).getByRole("button", { name: /look up by number/i })).toBeTruthy();
    expect(screen.queryByRole("form", { name: /carrier search/i })).toBeNull();
    expect(mocks.searchCarrierSourcing).not.toHaveBeenCalled();
  });

  it("offers the search form when the provider supports search", async () => {
    mocks.fetchCarrierIntelSettings.mockResolvedValue({
      carrierIntelControl: {},
      carrierIntelRuleCatalog: [],
      carrierIntelProvider: {
        configured: true,
        provider: "CarrierOK",
        fallbackProvider: "FMCSAQCMobile",
        capabilities: ["Search", "Autocomplete", "LookupFull", "LookupLite"],
        sections: [],
      },
    });

    render(
      <Providers>
        <CarrierSourcingWorkspace />
      </Providers>,
    );

    expect(await screen.findByRole("form", { name: /carrier search/i })).toBeTruthy();
    expect(screen.queryByTestId("sourcing-search-unsupported")).toBeNull();
  });
});
