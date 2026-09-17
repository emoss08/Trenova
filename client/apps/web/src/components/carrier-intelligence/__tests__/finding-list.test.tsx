import type { CarrierIntelFinding } from "@/lib/graphql/carrier-intelligence";
import { groupFindings } from "@/lib/carrier-intelligence";
import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { FindingList } from "../finding-list";
import { buildFinding } from "./fixtures";

function finding(overrides: Partial<CarrierIntelFinding> = {}): CarrierIntelFinding {
  return buildFinding({
    code: "authority.inactive",
    message: "Operating authority is not active",
    ...overrides,
  });
}

const FINDINGS: CarrierIntelFinding[] = [
  finding({
    code: "insurance.low",
    action: "Warn",
    severity: "Medium",
    category: "Insurance",
    message: "Insurance is low",
  }),
  finding({ code: "authority.inactive", action: "Block", severity: "High" }),
  finding({
    code: "safety.oos",
    action: "Block",
    severity: "Critical",
    category: "Safety",
    message: "Carrier is out of service",
  }),
  finding({
    code: "contacts.changed",
    action: "Notify",
    severity: "Info",
    category: "Contacts",
    message: "Contact details changed",
  }),
  finding({
    code: "lanes.off",
    action: "Off",
    severity: "Low",
    category: "Lanes",
    message: "Lane rule is off",
  }),
];

function kinds(container: HTMLElement): (string | null)[] {
  return [...container.querySelectorAll("[data-finding-kind]")].map((node) =>
    node.getAttribute("data-finding-kind"),
  );
}

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
  it("lists blockers, advisories and notices in that order and leaves out rules that are off", () => {
    const { container } = render(<FindingList findings={FINDINGS} />);

    expect(kinds(container)).toEqual(["blocker", "blocker", "advisory", "notice"]);
    expect(screen.getByText("Carrier is out of service")).toBeInTheDocument();
    expect(screen.queryByText("Lane rule is off")).toBeNull();
    expect(screen.queryByText("safety.oos")).toBeNull();
  });

  it("groups under quiet headings with counts and omits empty groups", () => {
    const { container } = render(<FindingList findings={FINDINGS} grouped />);

    const groups = [...container.querySelectorAll("[data-finding-group]")].map((node) =>
      node.getAttribute("data-finding-group"),
    );
    expect(groups).toEqual(["blocker", "advisory", "notice"]);

    const blocking = screen.getByRole("region", { name: "Blocking" });
    expect(within(blocking).getByText("2")).toBeInTheDocument();
    expect(within(blocking).getByText("Carrier is out of service")).toBeInTheDocument();
    expect(within(blocking).queryByText("Insurance is low")).toBeNull();

    render(<FindingList findings={[finding({ action: "Warn" })]} grouped />);
    expect(screen.getAllByRole("region", { name: "Blocking" })).toHaveLength(1);
  });

  it("uses the rule label as the title and the message as muted detail", () => {
    render(
      <FindingList
        findings={[finding()]}
        ruleLabels={{ "authority.inactive": "Authority must be active" }}
      />,
    );

    expect(screen.getByText("Authority must be active")).toBeInTheDocument();
    expect(screen.getByText("Operating authority is not active")).toBeInTheDocument();
    expect(screen.queryByText("authority.inactive")).toBeNull();
  });

  it("notes unverifiable, unconfirmed and overridden findings in plain text", () => {
    render(
      <FindingList
        findings={[
          finding({ code: "a.unverifiable", message: "A", unverifiable: true }),
          finding({ code: "b.unconfirmed", message: "B", unconfirmed: true }),
          finding({
            code: "c.overridden",
            message: "C",
            overridden: true,
            overrideId: "cio_1",
            overrideExpiresAt: 1_900_000_000,
          }),
        ]}
      />,
    );

    expect(screen.getByText("Couldn't be verified")).toBeInTheDocument();
    expect(screen.getByText("Waiting for a confirming refresh")).toBeInTheDocument();
    expect(screen.getByText(/^Overridden until /)).toBeInTheDocument();
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

  it("shows a compact, limited list of the chosen kinds", () => {
    const { container } = render(
      <FindingList findings={FINDINGS} kinds={["blocker", "advisory"]} compact limit={2} />,
    );

    expect(kinds(container)).toEqual(["blocker", "blocker"]);
    expect(screen.getByText("1 more finding")).toBeInTheDocument();
    expect(screen.queryByText("Contact details changed")).toBeNull();
  });

  it("says when every rule passed, or renders nothing when asked", () => {
    const { container, rerender } = render(<FindingList findings={[finding({ action: "Off" })]} />);
    expect(screen.getByText("Every enabled vetting rule passed.")).toBeInTheDocument();

    rerender(<FindingList findings={[]} emptyMessage={null} />);
    expect(container).toBeEmptyDOMElement();
  });
});
