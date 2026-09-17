import type { CarrierIntelFinding } from "@/lib/graphql/carrier-intelligence";
import { groupFindings } from "@/lib/carrier-intelligence";
import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { FindingList } from "../finding-list";

function finding(overrides: Partial<CarrierIntelFinding> = {}): CarrierIntelFinding {
  return {
    code: "authority.inactive",
    category: "Authority",
    action: "Block",
    severity: "High",
    message: "Operating authority is not active",
    unverifiable: false,
    unconfirmed: false,
    overridden: false,
    overrideId: null,
    overrideExpiresAt: null,
    ...overrides,
  };
}

const FINDINGS: CarrierIntelFinding[] = [
  finding({ code: "insurance.low", action: "Warn", severity: "Medium", category: "Insurance" }),
  finding({ code: "authority.inactive", action: "Block", severity: "High" }),
  finding({ code: "safety.oos", action: "Block", severity: "Critical", category: "Safety" }),
  finding({ code: "contacts.changed", action: "Notify", severity: "Info", category: "Contacts" }),
  finding({ code: "lanes.off", action: "Off", severity: "Low", category: "Lanes" }),
];

describe("groupFindings", () => {
  it("splits findings by action, drops rules that are off, and orders by severity", () => {
    const grouped = groupFindings(FINDINGS);

    expect(grouped.blockers.map((item) => item.code)).toEqual(["safety.oos", "authority.inactive"]);
    expect(grouped.advisories.map((item) => item.code)).toEqual(["insurance.low"]);
    expect(grouped.notices.map((item) => item.code)).toEqual(["contacts.changed"]);
  });

  it("does not reorder the caller's array", () => {
    const input = [...FINDINGS];
    groupFindings(input);
    expect(input.map((item) => item.code)).toEqual(FINDINGS.map((item) => item.code));
  });
});

describe("FindingList", () => {
  it("renders blockers, advisories and notices as separate groups in that order", () => {
    const { container } = render(<FindingList findings={FINDINGS} />);

    const groups = [...container.querySelectorAll("[data-finding-group]")].map((node) =>
      node.getAttribute("data-finding-group"),
    );
    expect(groups).toEqual(["blocker", "advisory", "notice"]);

    const blockers = screen.getByRole("region", { name: "Blockers" });
    expect(within(blockers).getByText("safety.oos")).toBeInTheDocument();
    expect(within(blockers).getByText("authority.inactive")).toBeInTheDocument();
    expect(within(blockers).queryByText("insurance.low")).toBeNull();
    expect(screen.queryByText("lanes.off")).toBeNull();
  });

  it("omits a group that has nothing in it", () => {
    render(<FindingList findings={[finding({ action: "Warn" })]} />);

    expect(screen.queryByRole("region", { name: "Blockers" })).toBeNull();
    expect(screen.getByRole("region", { name: "Advisories" })).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Notices" })).toBeNull();
  });

  it("flags unverifiable, unconfirmed and overridden findings", () => {
    render(
      <FindingList
        findings={[
          finding({ code: "a.unverifiable", unverifiable: true }),
          finding({ code: "b.unconfirmed", unconfirmed: true }),
          finding({
            code: "c.overridden",
            overridden: true,
            overrideId: "cio_1",
            overrideExpiresAt: 1_900_000_000,
          }),
        ]}
      />,
    );

    expect(screen.getByText("Unverifiable")).toBeInTheDocument();
    expect(screen.getByText("Unconfirmed")).toBeInTheDocument();
    expect(screen.getByText(/^Overridden until /)).toBeInTheDocument();
  });

  it("prefers the rule label and still shows the code", () => {
    render(
      <FindingList
        findings={[finding()]}
        ruleLabels={{ "authority.inactive": "Authority must be active" }}
      />,
    );

    expect(screen.getByText("Authority must be active")).toBeInTheDocument();
    expect(screen.getByText("authority.inactive")).toBeInTheDocument();
  });

  it("renders per-finding actions from the caller", () => {
    render(
      <FindingList
        findings={FINDINGS}
        renderActions={(item) =>
          item.action === "Block" ? <button type="button">Override {item.code}</button> : null
        }
      />,
    );

    expect(screen.getByRole("button", { name: "Override safety.oos" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Override insurance.low" })).toBeNull();
  });

  it("says when every rule passed", () => {
    render(<FindingList findings={[finding({ action: "Off" })]} />);

    expect(
      screen.getByText("No findings. Every enabled rule passed on the latest vetting."),
    ).toBeInTheDocument();
  });
});
