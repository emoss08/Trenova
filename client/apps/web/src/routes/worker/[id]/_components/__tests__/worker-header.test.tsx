import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WorkerHeader, type WorkerHeaderProps } from "../worker-header";
import { mariaRecord } from "./worker-fixtures";

vi.mock("../add-to-record-menu", () => ({ WorkerRecordMenus: () => null }));
vi.mock("@/components/accounting-sync/sync-state-line", () => ({
  AccountingSyncStateLine: () => null,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: false, isLoading: false }),
}));

afterEach(cleanup);

function overview(overrides: Partial<WorkerOverview> = {}): WorkerOverview {
  return {
    asOf: 1_795_305_600,
    standing: "Watch",
    concernCount: 2,
    concerns: [
      {
        severity: "Critical",
        code: "EmploymentVerificationOutstanding",
        subject: "Ridgeline Freight",
        count: 1,
        occurredAt: null,
        dueAt: null,
        daysUntil: null,
        stopsAssignment: false,
        destination: { tab: "Qualification", view: "DqFile", record: null, recordId: null },
      },
    ],
    nextTimeOff: null,
    documentCount: 14,
    agentMemoryCount: null,
    nextReviewAt: null,
    clearance: null,
    credentials: null,
    qualificationFile: null,
    drugAlcohol: null,
    training: null,
    safety: null,
    employment: null,
    checklist: null,
    pto: null,
    leave: null,
    openReview: null,
    lastReview: null,
    schedule: null,
    hos: null,
    pay: null,
    benefits: null,
    portal: null,
    worker: {} as WorkerOverview["worker"],
    ...overrides,
  };
}

function renderHeader(props: Partial<WorkerHeaderProps> = {}) {
  const onTab = vi.fn();
  render(
    <WorkerHeader
      worker={mariaRecord()}
      overview={overview()}
      overviewUpdatedAt={0}
      overviewFetching={false}
      onRefresh={() => undefined}
      tabs={["overview", "profile", "qualification", "files"]}
      tab="overview"
      onTab={onTab}
      onTabIntent={() => undefined}
      profileUnsaved={false}
      onOpen={() => undefined}
      {...props}
    />,
  );
  return { onTab };
}

describe("the worker's record header", () => {
  it("names the worker with their status and standing", () => {
    renderHeader();

    expect(screen.getByRole("heading", { name: "Maria Alvarez" })).toBeTruthy();
    expect(screen.getByText("Active")).toBeTruthy();
    expect(screen.getByText("Worth watching")).toBeTruthy();
  });

  it("marks the header with the worker's face, their initials while they have no photo", () => {
    renderHeader({ worker: mariaRecord({ photo: null }) });

    expect(screen.getByText("MA")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Add a photo" })).toBeNull();
  });

  it("puts the facts a dispatcher reaches for on the meta line", () => {
    renderHeader();

    expect(screen.getByText("MW-OTR")).toBeTruthy();
    expect(screen.getByText("Company driver")).toBeTruthy();
    expect(screen.getByText("Joliet, IL")).toBeTruthy();
    expect(screen.getByText("(815) 555-0142")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Copy phone number" })).toBeTruthy();
  });

  it("draws only the drawable tabs, with a red dot where something needs action and the document count", () => {
    renderHeader();

    const tabs = screen.getAllByRole("tab");
    expect(tabs.map((tab) => tab.textContent)).toEqual([
      "Overview",
      "Profile",
      "Qualification",
      "Files14",
    ]);
    const qualification = screen.getByRole("tab", { name: /Qualification/ });
    expect(within(qualification).getByRole("img", { name: "1 thing needs action" })).toBeTruthy();
  });

  it("marks the Profile tab while it holds unsaved edits", () => {
    renderHeader({ profileUnsaved: true });

    const profile = screen.getByRole("tab", { name: /Profile/ });
    expect(within(profile).getByRole("img", { name: "Unsaved changes" })).toBeTruthy();
  });

  it("leaves the standing out until the overview answers", () => {
    renderHeader({ overview: undefined });

    expect(screen.queryByText("Worth watching")).toBeNull();
    expect(screen.getByRole("tab", { name: "Files" })).toBeTruthy();
  });

  it("asks for the tab a person picks", () => {
    const { onTab } = renderHeader();

    fireEvent.click(screen.getByRole("tab", { name: /Qualification/ }));

    expect(onTab).toHaveBeenCalledWith("qualification");
  });
});
