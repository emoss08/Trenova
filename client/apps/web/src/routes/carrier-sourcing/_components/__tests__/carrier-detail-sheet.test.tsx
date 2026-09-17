import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CarrierDetailSheet } from "../carrier-detail-sheet";
import { buildCandidate, buildFinding, buildProfile, TestProviders } from "./fixtures";

const mocks = vi.hoisted(() => ({ lookupCarrierIntelProspect: vi.fn() }));

vi.mock("@/lib/graphql/carrier-sourcing", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-sourcing")>()),
  lookupCarrierIntelProspect: mocks.lookupCarrierIntelProspect,
}));

afterEach(() => {
  vi.clearAllMocks();
});

const provider = {
  configured: true,
  provider: "CarrierOK",
  capabilities: ["Search", "LookupFull", "LookupLite"],
};

const seed = buildCandidate({
  findings: [buildFinding(), buildFinding({ code: "safety.unrated", action: "Warn" })],
  profile: buildProfile({
    insurance: {
      bipdOnFile: "500000",
      bipdRequired: "750000",
      cargoOnFile: null,
      cargoRequired: null,
      bondOnFile: null,
      bondRequired: null,
      pendingCancelAt: null,
      lastCanceledAt: null,
      cancelCount: 0,
      filings: [],
    },
  }),
});

function renderSheet(overrides: Partial<Parameters<typeof CarrierDetailSheet>[0]> = {}) {
  const onImport = vi.fn();
  render(
    <TestProviders>
      <CarrierDetailSheet
        target={{ dotNumber: seed.dotNumber, seed }}
        provider={provider}
        imported={{}}
        canImport
        ruleLabels={{ "authority.too_new": "Authority too new" }}
        onOpenChange={vi.fn()}
        onImport={onImport}
        {...overrides}
      />
    </TestProviders>,
  );
  return { onImport };
}

describe("CarrierDetailSheet", () => {
  it("leads with the decision and the key facts in human units", async () => {
    const user = userEvent.setup();
    const { onImport } = renderSheet();

    expect(await screen.findByText("Blue Ridge Freight LLC")).toBeTruthy();
    expect(screen.getByTestId("decision-headline").textContent).toContain(
      "1 issue blocks tendering",
    );
    expect(screen.getByText("Authority too new")).toBeTruthy();
    expect(screen.getAllByText("18 yrs").length).toBeGreaterThan(0);
    expect(screen.getAllByText("$500,000.00").length).toBeGreaterThan(0);
    expect(screen.getByText("of $750,000.00 required")).toBeTruthy();
    expect(screen.getByRole("button", { name: /pull full profile · ~\$3\.00\/mo/i })).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Import carrier" }));
    expect(onImport).toHaveBeenCalledWith(seed);
    expect(mocks.lookupCarrierIntelProspect).not.toHaveBeenCalled();
  });

  it("links to the carrier once it has been imported", async () => {
    renderSheet({ imported: { [seed.dotNumber]: "car_imported" } });

    const link = await screen.findByRole("button", { name: /open in trenova/i });
    expect(link.getAttribute("href")).toBe(
      "/dispatch/carriers?panelType=edit&panelEntityId=car_imported&tab=intelligence",
    );
    expect(screen.queryByRole("button", { name: "Import carrier" })).toBeNull();
  });

  it("marks sections the provider does not cover", async () => {
    const user = userEvent.setup();
    renderSheet();

    await user.click(await screen.findByRole("tab", { name: "Network signals" }));
    expect(screen.getByText("Not provided by CarrierOk")).toBeTruthy();
  });

  it("pulls a preview when opened from a suggestion", async () => {
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
    renderSheet({ target: { dotNumber: "1234567", seed: null } });

    expect(await screen.findByText("Blue Ridge Freight LLC")).toBeTruthy();
    expect(mocks.lookupCarrierIntelProspect).toHaveBeenCalledWith(
      { dotNumber: "1234567", depth: "Lite" },
      expect.anything(),
    );
    expect(screen.getByTestId("decision-headline").textContent).toBe("No blocking issues");
  });
});
