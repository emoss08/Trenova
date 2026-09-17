import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import { SourcingResultList } from "../sourcing-result-list";
import { buildCandidate, buildFinding, buildProfile, TestProviders } from "./fixtures";

function renderList(props: Partial<ComponentProps<typeof SourcingResultList>> = {}) {
  const onOpen = vi.fn();
  const onImport = vi.fn();
  const existing = buildCandidate({
    dotNumber: "265752",
    legalName: "Existing Freight Inc",
    existingCarrierId: "car_existing",
    riskLevel: "Low",
  });
  const prospect = buildCandidate({
    dotNumber: "7654321",
    legalName: "Piedmont Haulers",
    riskLevel: "VeryHigh",
    findings: [
      buildFinding(),
      buildFinding({ code: "insurance.bipd_below_required", category: "Insurance" }),
      buildFinding({ code: "safety.oos_rate", action: "Warn", severity: "Medium" }),
    ],
    profile: buildProfile({
      identity: { ...buildProfile().identity!, dotNumber: "7654321", dbaName: "Piedmont" },
    }),
  });
  render(
    <TestProviders>
      <SourcingResultList
        candidates={[existing, prospect]}
        canImport
        onOpen={onOpen}
        onImport={onImport}
        {...props}
      />
    </TestProviders>,
  );
  const [existingRow, prospectRow] = screen.getAllByTestId("sourcing-result");
  return { onOpen, onImport, existingRow, prospectRow, prospect };
}

describe("SourcingResultRow", () => {
  it("shows In Trenova for a carrier that already exists instead of an import", () => {
    const { existingRow } = renderList();

    const link = within(existingRow).getByTestId("existing-carrier-link");
    expect(link.textContent).toBe("In Trenova");
    expect(link.getAttribute("href")).toBe(
      "/dispatch/carriers?panelType=edit&panelEntityId=car_existing&tab=intelligence",
    );
    expect(within(existingRow).queryByRole("button", { name: "Import" })).toBeNull();
  });

  it("offers Import for a prospect and summarises it in plain text", async () => {
    const user = userEvent.setup();
    const { prospectRow, onImport, onOpen, prospect } = renderList();

    expect(within(prospectRow).queryByTestId("existing-carrier-link")).toBeNull();
    expect(within(prospectRow).getByText("Piedmont Haulers")).toBeTruthy();
    expect(within(prospectRow).getByText("Piedmont")).toBeTruthy();
    expect(within(prospectRow).getByText("USDOT 7654321 · MC 765432 · Asheville, NC")).toBeTruthy();
    expect(within(prospectRow).getByText("101,844 trucks")).toBeTruthy();
    expect(within(prospectRow).getByText("18 yrs")).toBeTruthy();
    expect(within(prospectRow).getByText("Very high")).toBeTruthy();
    expect(within(prospectRow).getByText("2 blockers")).toBeTruthy();

    await user.click(within(prospectRow).getByRole("button", { name: "Import" }));
    expect(onImport).toHaveBeenCalledWith(prospect);
    expect(onOpen).not.toHaveBeenCalled();
  });

  it("hides Import when the user cannot import carriers", () => {
    const { prospectRow } = renderList({ canImport: false });
    expect(within(prospectRow).queryByRole("button", { name: "Import" })).toBeNull();
  });

  it("opens a row on click and moves between rows with the arrow keys", async () => {
    const user = userEvent.setup();
    const { onOpen, existingRow, prospectRow, prospect } = renderList();

    const first = within(existingRow).getByRole("button", { name: "Open Existing Freight Inc" });
    const second = within(prospectRow).getByRole("button", { name: "Open Piedmont Haulers" });
    expect(first.getAttribute("tabindex")).toBe("0");
    expect(second.getAttribute("tabindex")).toBe("-1");

    first.focus();
    await user.keyboard("{ArrowDown}");
    expect(document.activeElement).toBe(second);
    await user.keyboard("{Enter}");
    expect(onOpen).toHaveBeenCalledWith(prospect);

    await user.click(first);
    expect(onOpen).toHaveBeenCalledTimes(2);
  });
});
