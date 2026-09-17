import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { CarrierIntelProfileView } from "../carrier-intel-profile-view";
import { buildProfile, emptyProfile } from "./fixtures";

async function openTab(name: string) {
  const user = userEvent.setup();
  await user.click(screen.getByRole("tab", { name }));
  return screen.getByRole("tabpanel", { name });
}

describe("CarrierIntelProfileView coverage", () => {
  it("says a section was not provided instead of rendering zeros", async () => {
    render(
      <CarrierIntelProfileView
        provider="FMCSAQCMobile"
        sections={["company", "network", "lanes"]}
        profile={emptyProfile({
          coverage: ["Identity"],
          identity: buildProfile().identity,
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

    const company = screen.getByRole("tabpanel", { name: "Company" });
    expect(within(company).getByText("Blue Ridge Freight LLC")).toBeInTheDocument();
    expect(within(company).getByText("6 yrs")).toBeInTheDocument();
    expect(within(company).getAllByText("Not provided by FMCSA QCMobile")).toHaveLength(2);

    const network = await openTab("Network signals");
    expect(within(network).getByText("Not provided by FMCSA QCMobile")).toBeInTheDocument();
    expect(within(network).queryByText("0")).toBeNull();

    const lanes = await openTab("Lanes");
    expect(within(lanes).getByText("Not provided by FMCSA QCMobile")).toBeInTheDocument();
  });

  it("distinguishes a covered section the provider returned empty", () => {
    render(
      <CarrierIntelProfileView
        provider="CarrierOK"
        sections={["lanes"]}
        profile={emptyProfile({ coverage: ["Lanes"], lanes: null })}
      />,
    );

    const lanes = screen.getByRole("tabpanel", { name: "Lanes" });
    expect(within(lanes).queryByText(/Not provided by/)).toBeNull();
    expect(
      within(lanes).getByText("CarrierOk returned no data for this section"),
    ).toBeInTheDocument();
  });

  it("groups network links by carrier and flags shared EINs", () => {
    const { container } = render(
      <CarrierIntelProfileView
        provider="CarrierOK"
        sections={["network"]}
        profile={emptyProfile({
          coverage: ["Network"],
          network: {
            sharedAddresses: 1,
            sharedPhones: 2,
            sharedEmails: 0,
            sharedEins: 1,
            sharedEquipment: 0,
            links: [
              { kind: "Phone", dotNumber: "999999", legalName: null, value: null, status: null },
              { kind: "EIN", dotNumber: "999999", legalName: null, value: null, status: null },
              { kind: "Address", dotNumber: "111111", legalName: null, value: null, status: null },
            ],
          },
        })}
      />,
    );

    const network = screen.getByRole("tabpanel", { name: "Network signals" });
    expect(within(network).getByText("Change history")).toBeInTheDocument();
    expect(within(network).getAllByText(/Not provided by/)).toHaveLength(1);
    const linked = [...container.querySelectorAll("[data-linked-dot]")].map((node) =>
      node.getAttribute("data-linked-dot"),
    );
    expect(linked).toEqual(["999999", "111111"]);
    expect(within(network).getByText("USDOT 999999")).toBeInTheDocument();
    expect(within(network).getByText("Shared EIN and Shared phone")).toBeInTheDocument();
  });

  it("shows CSA BASICs by measure and violations without an empty percentile column", () => {
    render(
      <CarrierIntelProfileView
        provider="CarrierOK"
        sections={["safety"]}
        profile={emptyProfile({
          coverage: ["Basics"],
          basics: [
            {
              basic: "UnsafeDriving",
              measure: 2.5,
              percentile: null,
              threshold: null,
              alert: true,
              roadsideAlert: false,
              acIndicator: false,
              violations: 14,
              oosViolations: 3,
              measuredAt: 1_788_220_800,
            },
          ],
        })}
      />,
    );

    const basics = screen.getByTestId("csa-basics");
    expect(within(basics).getByText("Unsafe Driving")).toBeInTheDocument();
    expect(within(basics).getByText("2.5")).toBeInTheDocument();
    expect(within(basics).getByText("14")).toBeInTheDocument();
    expect(within(basics).queryByText("Percentile")).toBeNull();
    expect(within(basics).getByText(/1 BASIC over threshold · measured/)).toBeInTheDocument();
  });

  it("marks active and historical insurance filings", () => {
    render(
      <CarrierIntelProfileView
        provider="CarrierOK"
        sections={["insurance"]}
        profile={emptyProfile({
          coverage: ["Insurance"],
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
            filings: [
              {
                type: "BIPD",
                status: "H",
                insurerName: "Old Mutual",
                policyNumber: "P-1",
                coverage: "750000",
                effectiveAt: null,
                cancelEffectiveAt: null,
                cancelMethod: null,
              },
              {
                type: "BIPD",
                status: "A",
                insurerName: "Progressive",
                policyNumber: "P-2",
                coverage: "500000",
                effectiveAt: null,
                cancelEffectiveAt: null,
                cancelMethod: null,
              },
            ],
          },
        })}
      />,
    );

    const insurance = screen.getByRole("tabpanel", { name: "Insurance" });
    const rows = within(insurance).getAllByRole("row");
    const filingRows = rows.filter((row) => row.textContent?.includes("P-"));
    expect(filingRows[0]).toHaveTextContent("Active");
    expect(filingRows[0]).toHaveTextContent("Progressive");
    expect(filingRows[1]).toHaveTextContent("Historical");
    expect(insurance.querySelector('[data-coverage="short"]')).not.toBeNull();
  });

  it("summarises which sections the provider left out", () => {
    render(
      <CarrierIntelProfileView
        provider="FMCSAQCMobile"
        sections={["authority"]}
        showCoverageSummary
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
