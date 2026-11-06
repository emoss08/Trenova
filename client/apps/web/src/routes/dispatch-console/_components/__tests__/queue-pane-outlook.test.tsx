import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { QueueRead } from "../move-queue";
import { QueuePane } from "../queue-pane";
import type { QueueTab } from "../url-state";

vi.mock("../outlook-list", () => ({
  OutlookList: () => <div data-testid="outlook-list" />,
}));

const pending: QueueRead = { isPending: true, isError: false, error: null, refetch: vi.fn() };

function renderPane({ tab, showsDrivers }: { tab: QueueTab; showsDrivers: boolean }) {
  const onTab = vi.fn();
  render(
    <NuqsTestingAdapter searchParams="">
      <QueuePane
        tab={tab}
        onTab={onTab}
        moves={[]}
        moveCount={null}
        read={pending}
        narrowed={false}
        includeCovered={false}
        onClearSearch={vi.fn()}
        selectedMoveId={null}
        departingMoveIds={new Set()}
        onSelectMove={vi.fn()}
        outcome={null}
        canUndo={false}
        onUndo={vi.fn()}
        exceptions={[]}
        exceptionsRead={pending}
        exceptionsMore={false}
        drivers={[]}
        driversTruncated={false}
        selectedWorkerId={null}
        onSelectDriver={vi.fn()}
        showsDrivers={showsDrivers}
        listRef={null}
      />
    </NuqsTestingAdapter>,
  );
  return onTab;
}

describe("QueuePane outlook tab", () => {
  afterEach(cleanup);

  it("offers the outlook as a fourth tab that controls its own panel", async () => {
    const user = userEvent.setup();
    const onTab = renderPane({ tab: "moves", showsDrivers: true });

    const tabs = screen.getAllByRole("tab");
    expect(tabs.map((tab) => tab.textContent)).toEqual([
      "Needs a driver",
      "Exceptions",
      "Drivers",
      "Outlook",
    ]);
    const outlook = screen.getByRole("tab", { name: "Outlook" });
    expect(outlook).toHaveAttribute("id", "dispatch-tab-outlook");
    expect(outlook).toHaveAttribute("aria-controls", "dispatch-tabpanel-outlook");
    expect(outlook).toHaveAttribute("aria-selected", "false");
    expect(outlook).toHaveAttribute("tabindex", "-1");

    await user.click(outlook);

    expect(onTab).toHaveBeenCalledWith("outlook");
  });

  it("draws the outlook in a panel labelled by its tab when it is the open tab", () => {
    renderPane({ tab: "outlook", showsDrivers: true });

    const outlook = screen.getByRole("tab", { name: "Outlook" });
    expect(outlook).toHaveAttribute("aria-selected", "true");
    expect(outlook).toHaveAttribute("tabindex", "0");
    const panel = screen.getByRole("tabpanel");
    expect(panel).toHaveAttribute("id", "dispatch-tabpanel-outlook");
    expect(panel).toHaveAttribute("aria-labelledby", "dispatch-tab-outlook");
    expect(screen.getByTestId("outlook-list")).toBeInTheDocument();
    expect(screen.getByText("Next 48 hours, most short of drivers first")).toBeInTheDocument();
  });

  it("leaves the outlook out for an organization without drivers and opens the moves instead", () => {
    renderPane({ tab: "outlook", showsDrivers: false });

    expect(screen.queryByRole("tab", { name: "Outlook" })).toBeNull();
    expect(screen.queryByTestId("outlook-list")).toBeNull();
    expect(screen.getByRole("tabpanel")).toHaveAttribute("id", "dispatch-tabpanel-moves");
  });
});
