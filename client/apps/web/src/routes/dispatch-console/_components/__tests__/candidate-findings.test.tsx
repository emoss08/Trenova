import type { DispatchCandidate, DispatchFinding } from "@/lib/graphql/dispatch-console";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import { sharedFindings } from "../candidate-list";
import { FindingSummary } from "../finding-list";

afterEach(cleanup);

// DispatchFinding as dispatch_console.graphqls defines it.
function finding(overrides: Partial<DispatchFinding> = {}): DispatchFinding {
  return {
    code: "WINDOW_CLOSED",
    severity: "Warn",
    field: "originWindow",
    message:
      "The pickup window has already closed, so no driver can make it and on-time is not judged.",
    regulation: null,
    ...overrides,
  } as DispatchFinding;
}

function candidate(findings: DispatchFinding[]): DispatchCandidate {
  return { workerId: `wrk_${findings.length}`, findings } as unknown as DispatchCandidate;
}

describe("sharedFindings", () => {
  it("finds what every ranked driver shares, which is about the move", () => {
    const closed = finding();
    const hours = finding({ code: "HOS_TIGHT", field: "drive", message: "2h of drive left" });

    expect(sharedFindings([candidate([closed, hours]), candidate([closed])])).toEqual([closed]);
  });

  it("shares nothing when a driver lacks the finding", () => {
    expect(sharedFindings([candidate([finding()]), candidate([])])).toEqual([]);
  });

  it("leaves a lone driver's findings on its own row", () => {
    expect(sharedFindings([candidate([finding()])])).toEqual([]);
  });
});

describe("FindingSummary", () => {
  it("shows the most severe finding in one line and says how many more there are", () => {
    render(
      <FindingSummary
        findings={[
          finding(),
          finding({
            code: "MED_CARD",
            severity: "Block",
            field: "medical",
            message: "Medical card expired",
          }),
        ]}
      />,
    );

    const line = screen.getByText("Medical card expired").closest("[tabindex]") as HTMLElement;
    expect(line).toHaveTextContent("Blocking:");
    expect(line).toHaveTextContent("+1");
    expect(line).not.toHaveTextContent("pickup window");
  });

  it("holds every finding in full in its tooltip", async () => {
    render(<FindingSummary findings={[finding({ regulation: "49 CFR 395.3" })]} />);

    await userEvent.hover(
      screen.getByText(/pickup window has already closed/).closest("[tabindex]")!,
    );

    expect(
      await screen.findByText(
        "Warning: The pickup window has already closed, so no driver can make it and on-time is not judged. (49 CFR 395.3)",
      ),
    ).toBeInTheDocument();
  });
});
