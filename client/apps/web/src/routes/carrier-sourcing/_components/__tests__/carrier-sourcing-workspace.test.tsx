import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CarrierSourcingWorkspace } from "../carrier-sourcing-workspace";
import { buildProfile, TestProviders } from "./fixtures";

const mocks = vi.hoisted(() => ({
  fetchCarrierIntelSettings: vi.fn(),
  searchCarrierSourcing: vi.fn(),
  lookupCarrierIntelProspect: vi.fn(),
  fetchCarrierSourcingAutocomplete: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-intelligence", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-intelligence")>()),
  fetchCarrierIntelSettings: mocks.fetchCarrierIntelSettings,
}));
vi.mock("@/lib/graphql/carrier-sourcing", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-sourcing")>()),
  searchCarrierSourcing: mocks.searchCarrierSourcing,
  lookupCarrierIntelProspect: mocks.lookupCarrierIntelProspect,
  fetchCarrierSourcingAutocomplete: mocks.fetchCarrierSourcingAutocomplete,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

afterEach(() => {
  vi.clearAllMocks();
});

function settings(provider: {
  configured: boolean;
  provider: string | null;
  capabilities: string[];
}) {
  return {
    carrierIntelControl: {},
    carrierIntelRuleCatalog: [],
    carrierIntelProvider: { fallbackProvider: null, sections: [], ...provider },
  };
}

function renderWorkspace() {
  render(
    <TestProviders>
      <CarrierSourcingWorkspace />
    </TestProviders>,
  );
}

describe("CarrierSourcingWorkspace", () => {
  it("asks to connect a provider when none is configured", async () => {
    mocks.fetchCarrierIntelSettings.mockResolvedValue(
      settings({ configured: false, provider: null, capabilities: [] }),
    );
    renderWorkspace();

    expect(await screen.findByText("Connect a carrier data provider")).toBeTruthy();
    expect(screen.queryByRole("combobox")).toBeNull();
  });

  it("runs a number as a lookup and shows the carrier as the only result", async () => {
    const user = userEvent.setup();
    mocks.fetchCarrierIntelSettings.mockResolvedValue(
      settings({
        configured: true,
        provider: "CarrierOK",
        capabilities: ["Search", "Autocomplete", "LookupFull", "LookupLite"],
      }),
    );
    mocks.lookupCarrierIntelProspect.mockResolvedValue({
      existingCarrierId: null,
      snapshot: {
        dotNumber: "1234567",
        provider: "CarrierOK",
        depth: "Lite",
        notFound: false,
        riskLevel: "Low",
        findings: [],
        profile: buildProfile(),
        fetchedAt: 1_700_000_000,
        confirmedAt: null,
        effectiveAsOf: 1_700_000_000,
        sourceAsOf: null,
      },
    });
    renderWorkspace();

    const input = await screen.findByRole("combobox");
    expect(screen.getByRole("button", { name: /filter/i })).toBeTruthy();
    await user.type(input, "USDOT 1234567");
    await user.keyboard("{Escape}{Enter}");

    expect(await screen.findByText("Blue Ridge Freight LLC")).toBeTruthy();
    expect(mocks.lookupCarrierIntelProspect).toHaveBeenCalledWith(
      { dotNumber: "1234567", depth: "Lite" },
      expect.anything(),
    );
    expect(mocks.searchCarrierSourcing).not.toHaveBeenCalled();
    expect(screen.getAllByTestId("sourcing-result")).toHaveLength(1);
  });

  it("explains that a lookup-only provider cannot search by name", async () => {
    const user = userEvent.setup();
    mocks.fetchCarrierIntelSettings.mockResolvedValue(
      settings({ configured: true, provider: "FMCSAQCMobile", capabilities: ["LookupFMCSA"] }),
    );
    renderWorkspace();

    const input = await screen.findByRole("combobox");
    expect(screen.queryByRole("button", { name: /filter/i })).toBeNull();
    await user.type(input, "Blue Ridge Freight");
    await user.keyboard("{Escape}{Enter}");

    expect(await screen.findByTestId("sourcing-search-unsupported")).toBeTruthy();
    expect(screen.getByText("FMCSA QCMobile can't search by name")).toBeTruthy();
    expect(mocks.searchCarrierSourcing).not.toHaveBeenCalled();
    expect(mocks.fetchCarrierSourcingAutocomplete).not.toHaveBeenCalled();
  });

  describe("search", () => {
    function searchPage(filteredOut: number) {
      return {
        total: 40,
        filteredOut,
        provider: "CarrierOK",
        items: [
          {
            providerRef: "ref-1",
            dotNumber: "1234567",
            legalName: "Blue Ridge Freight LLC",
            existingCarrierId: null,
            laneMatches: 0,
            riskLevel: "Low",
            score: 1,
            findings: [],
            profile: buildProfile(),
          },
        ],
      };
    }

    function lastSearchInput() {
      const calls = mocks.searchCarrierSourcing.mock.calls;
      return calls[calls.length - 1][0];
    }

    async function searchByName(filteredOut = 0) {
      const user = userEvent.setup();
      mocks.fetchCarrierIntelSettings.mockResolvedValue(
        settings({
          configured: true,
          provider: "CarrierOK",
          capabilities: ["Search", "LookupLite"],
        }),
      );
      mocks.searchCarrierSourcing.mockResolvedValue(searchPage(filteredOut));
      renderWorkspace();

      const input = await screen.findByRole("combobox");
      await user.type(input, "Blue Ridge");
      await user.keyboard("{Escape}{Enter}");
      await screen.findByText("Blue Ridge Freight LLC");
      return user;
    }

    it("explains matches the filters hid with the server's count", async () => {
      await searchByName(12);

      expect(await screen.findByText(/40 carriers · 12 hidden by filters/)).toBeTruthy();
      expect(screen.queryByText(/shown/)).toBeNull();
    });

    it("asks the server to sort instead of reordering the page", async () => {
      const user = await searchByName();
      expect(lastSearchInput()).toMatchObject({ text: "Blue Ridge", sort: "BestMatch" });

      await user.click(screen.getByRole("button", { name: "Sort carriers" }));
      await user.click(await screen.findByRole("menuitemradio", { name: "Fleet size" }));

      await waitFor(() => expect(lastSearchInput()).toMatchObject({ sort: "FleetSizeDesc" }));
    });

    it("bounds the authority age on both ends", async () => {
      const user = await searchByName();

      await user.click(screen.getByRole("button", { name: /filter/i }));
      await user.click(await screen.findByRole("option", { name: "Authority age" }));
      await user.click(await screen.findByRole("option", { name: "1–3 yrs" }));

      await waitFor(() =>
        expect(lastSearchInput()).toMatchObject({
          minAuthorityAgeDays: 365,
          maxAuthorityAgeDays: 1094,
        }),
      );
    });
  });
});
