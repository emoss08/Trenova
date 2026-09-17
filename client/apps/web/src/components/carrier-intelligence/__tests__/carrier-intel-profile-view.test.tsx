import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CarrierIntelProfileView } from "../carrier-intel-profile-view";

function emptyProfile(overrides: Partial<CarrierIntelProfile> = {}): CarrierIntelProfile {
  return {
    coverage: [],
    identity: null,
    authority: null,
    insurance: null,
    safety: null,
    basics: null,
    inspections: null,
    crashes: null,
    fleet: null,
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

function section(name: string) {
  return screen.getByRole("region", { name });
}

describe("CarrierIntelProfileView coverage", () => {
  it("says a section was not provided instead of rendering zeros", () => {
    render(
      <CarrierIntelProfileView
        provider="FMCSAQCMobile"
        cards={["identity", "network", "lanes", "changeHistory"]}
        profile={emptyProfile({
          coverage: ["Identity"],
          identity: {
            dotNumber: "818175",
            docketPrefix: "MC",
            docketNumber: "123456",
            legalName: "Acme Freight LLC",
            dbaName: null,
            ein: null,
            usdotStatus: "Active",
            entityType: "Carrier",
            carrierOperation: "Interstate",
            dotAddedAt: null,
            dotAgeDays: 1200,
            physicalAddress: null,
            mailingAddress: null,
          },
          network: {
            sharedAddresses: 0,
            sharedPhones: 0,
            sharedEmails: 0,
            sharedEins: 0,
            sharedEquipment: 0,
            links: [],
          },
        })}
      />,
    );

    expect(within(section("Identity")).getByText("Acme Freight LLC")).toBeInTheDocument();
    expect(within(section("Identity")).queryByText(/Not provided by/)).toBeNull();

    const network = section("Network signals");
    expect(within(network).getByText("Not provided by FMCSA QCMobile")).toBeInTheDocument();
    expect(within(network).queryByText("Shared EIN")).toBeNull();
    expect(within(network).queryByText("0")).toBeNull();

    expect(
      within(section("Lanes")).getByText("Not provided by FMCSA QCMobile"),
    ).toBeInTheDocument();
    expect(
      within(section("Change history")).getByText("Not provided by FMCSA QCMobile"),
    ).toBeInTheDocument();
  });

  it("distinguishes a covered section the provider returned empty", () => {
    render(
      <CarrierIntelProfileView
        provider="CarrierOK"
        cards={["lanes"]}
        profile={emptyProfile({ coverage: ["Lanes"], lanes: null })}
      />,
    );

    const lanes = section("Lanes");
    expect(within(lanes).queryByText(/Not provided by/)).toBeNull();
    expect(
      within(lanes).getByText("CarrierOk returned no data for this section"),
    ).toBeInTheDocument();
  });

  it("renders covered counters, including real zeros", () => {
    render(
      <CarrierIntelProfileView
        provider="CarrierOK"
        cards={["network"]}
        profile={emptyProfile({
          coverage: ["Network"],
          network: {
            sharedAddresses: 0,
            sharedPhones: 2,
            sharedEmails: 0,
            sharedEins: 1,
            sharedEquipment: 0,
            links: [
              {
                kind: "EIN",
                dotNumber: "999999",
                legalName: "Shadow Carrier Inc",
                value: "12-3456789",
                status: "Inactive",
              },
            ],
          },
        })}
      />,
    );

    const network = section("Network signals");
    expect(within(network).queryByText(/Not provided by/)).toBeNull();
    expect(within(network).getByText("Identity overlaps with other carriers")).toBeInTheDocument();
    expect(within(network).getByText("Shadow Carrier Inc")).toBeInTheDocument();
  });

  it("summarises which sections the provider left out", () => {
    render(
      <CarrierIntelProfileView
        provider="FMCSAQCMobile"
        cards={[]}
        profile={emptyProfile({
          coverage: [
            "Identity",
            "Authority",
            "Insurance",
            "Safety",
            "Basics",
            "Inspections",
            "Crashes",
            "Fleet",
            "Operations",
          ],
        })}
      />,
    );

    const summary = screen.getByTestId("carrier-intel-coverage-summary");
    expect(summary).toHaveTextContent("FMCSA QCMobile provided 9 of 15 sections.");
    expect(summary).toHaveTextContent("Network signals");
    expect(summary).not.toHaveTextContent("Authority");
  });
});
