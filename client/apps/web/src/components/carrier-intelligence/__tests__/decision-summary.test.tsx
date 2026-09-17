import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DecisionSummary } from "../decision-summary";
import { buildFinding, IntelTestProviders } from "./fixtures";

const ruleLabels = {
  "authority.too_new": "Authority too new",
  "insurance.bipd_below_required": "BIPD below required",
};

describe("DecisionSummary", () => {
  it("counts the blocking issues and lists findings by rule label", () => {
    render(
      <IntelTestProviders>
        <DecisionSummary
          findings={[
            buildFinding({
              code: "safety.unrated",
              action: "Warn",
              message: "Carrier is not rated",
            }),
            buildFinding(),
            buildFinding({
              code: "insurance.bipd_below_required",
              message: "BIPD on file is below the $750,000 required",
            }),
          ]}
          ruleLabels={ruleLabels}
          notFound={false}
          provider="CarrierOK"
        />
      </IntelTestProviders>,
    );

    const headline = screen.getByTestId("decision-headline");
    expect(headline.textContent).toContain("2 issues block tendering");
    expect(headline.textContent).toContain("1 advisory");

    expect(screen.getByText("Authority too new")).toBeTruthy();
    expect(screen.getByText("Operating authority is younger than 180 days")).toBeTruthy();
    expect(screen.getByText("BIPD below required")).toBeTruthy();
    expect(screen.getByText("Carrier is not rated")).toBeTruthy();
    expect(screen.queryByText("authority.too_new")).toBeNull();

    const kinds = Array.from(document.querySelectorAll("[data-finding-kind]")).map((node) =>
      node.getAttribute("data-finding-kind"),
    );
    expect(kinds).toEqual(["blocker", "blocker", "advisory"]);
  });

  it("says nothing blocks tendering when only advisories remain", () => {
    render(
      <IntelTestProviders>
        <DecisionSummary
          findings={[buildFinding({ action: "Warn" })]}
          ruleLabels={ruleLabels}
          notFound={false}
          provider="CarrierOK"
        />
      </IntelTestProviders>,
    );

    expect(screen.getByTestId("decision-headline").textContent).toContain("No blocking issues");
  });

  it("confirms every rule passed when there are no findings", () => {
    render(
      <IntelTestProviders>
        <DecisionSummary findings={[]} ruleLabels={{}} notFound={false} provider="CarrierOK" />
      </IntelTestProviders>,
    );

    expect(screen.getByTestId("decision-headline").textContent).toBe("No blocking issues");
    expect(screen.getByText("Every enabled vetting rule passed for this carrier.")).toBeTruthy();
  });

  it("leaves overridden blockers out of the blocking count", () => {
    render(
      <IntelTestProviders>
        <DecisionSummary
          findings={[
            buildFinding({
              overridden: true,
              overrideId: "cio_1",
              overrideExpiresAt: 1_900_000_000,
            }),
          ]}
          ruleLabels={ruleLabels}
          notFound={false}
          provider="CarrierOK"
        />
      </IntelTestProviders>,
    );

    const headline = screen.getByTestId("decision-headline").textContent;
    expect(headline).toContain("No blocking issues");
    expect(headline).toContain("1 overridden");
    expect(
      screen.getByText(/^Operating authority is younger than 180 days · Overridden until /),
    ).toBeTruthy();
  });

  it("explains when the provider has no record", () => {
    render(
      <IntelTestProviders>
        <DecisionSummary findings={[]} ruleLabels={{}} notFound provider="FMCSAQCMobile" />
      </IntelTestProviders>,
    );

    expect(screen.getByTestId("decision-headline").textContent).toBe(
      "FMCSA QCMobile has no record for this carrier",
    );
  });
});
