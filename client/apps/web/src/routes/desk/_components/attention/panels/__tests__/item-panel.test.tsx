import type { WatchtowerItem, WatchtowerPanel } from "@/lib/graphql/watchtower";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { watchtowerItem } from "../../../watchtower/__tests__/item-fixture";
import { ItemPanel } from "../item-panel";

const NOW = 1_800_000_000;
const HOUR = 3600;

const EMPTY: WatchtowerPanel = {
  kind: "Stop",
  stop: null,
  clock: null,
  paper: null,
  message: null,
  expiry: null,
  coverage: null,
  contact: null,
  run: null,
  books: null,
  money: null,
  change: null,
  ask: null,
  finding: null,
  file: null,
  sighting: null,
};

const mocks = vi.hoisted(() => ({
  toast: vi.fn(),
  readiness: vi.fn(),
  upload: vi.fn(),
  resolve: vi.fn(),
}));

vi.mock("@/services/shipment", () => ({
  shipmentService: { getBillingReadiness: mocks.readiness },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: mocks.toast, error: vi.fn() },
}));
vi.mock("@/services/document", () => ({ documentService: { upload: mocks.upload } }));
vi.mock("@/hooks/use-permission", () => ({ usePermission: () => ({ allowed: true }) }));
vi.mock("@/lib/graphql/watchtower", async (original) => ({
  ...(await original<typeof import("@/lib/graphql/watchtower")>()),
  resolveWatchtowerItem: mocks.resolve,
}));

afterEach(() => vi.clearAllMocks());

function show(overrides: Partial<WatchtowerItem>) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <ItemPanel item={watchtowerItem(overrides)} now={NOW} timezone="UTC" />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const POD = {
  documentTypeId: "dt_pod",
  documentTypeCode: "POD",
  documentTypeName: "Proof of delivery",
  satisfied: false,
  documentCount: 0,
  documentIds: [],
  ineligibleDocuments: [],
};
const BOL = {
  ...POD,
  documentTypeId: "dt_bol",
  documentTypeName: "Bill of lading",
  satisfied: true,
  documentCount: 1,
};

/** Each kind of item is drawn by the view made for it, from what its source handed over. */
describe("an item's panel", () => {
  it("draws a missed stop on its line of time, with how late it is", () => {
    show({
      panel: {
        ...EMPTY,
        kind: "Stop",
        stop: {
          role: "Delivery",
          verdict: "Missed",
          place: "Los Angeles Terminal in Los Angeles, CA",
          windowEnd: NOW - 2 * HOUR,
          estimatedAt: 0,
          arrivedAt: 0,
          lastHeardAt: 0,
          milesRemaining: 0,
          stale: false,
        },
      },
    });

    expect(screen.getByText("Delivery")).toBeVisible();
    expect(
      screen.getByRole("heading", { name: "Los Angeles Terminal in Los Angeles, CA" }),
    ).toBeVisible();
    expect(screen.getByText("Missed")).toBeVisible();
    expect(screen.getByText("Late by")).toBeVisible();
    expect(screen.queryByText("Projected arrival")).toBeNull();
  });

  it("draws a detention clock that is billing, with what has accrued", () => {
    show({
      panel: {
        ...EMPTY,
        kind: "Clock",
        clock: {
          place: "Phoenix Cold Storage",
          customer: "FreshHaul Foods",
          arrivedAt: NOW - 5 * HOUR,
          freeEndsAt: NOW - 4 * HOUR,
          departedAt: 0,
          freeMinutes: 60,
          billableMinutes: 220,
          accrued: "$275.00",
        },
      },
    });

    expect(screen.getByText("Detention · FreshHaul Foods")).toBeVisible();
    expect(screen.getByText("Billing")).toBeVisible();
    expect(screen.getByText("$275.00")).toBeVisible();
    expect(screen.getByText("Free time ended")).toBeVisible();
  });

  it("lists the shipment's own paperwork, and files a missing paper from its row without a case", async () => {
    mocks.readiness
      .mockResolvedValueOnce({ requirements: [BOL, POD], missingRequirements: [POD] })
      .mockResolvedValue({
        requirements: [BOL, { ...POD, satisfied: true, documentCount: 1 }],
        missingRequirements: [],
      });
    mocks.upload.mockResolvedValue({});
    mocks.resolve.mockResolvedValue({});
    const user = userEvent.setup();
    show({
      id: "wt_9",
      version: 4,
      subjectType: "Shipment",
      subjectId: "shp_1",
      record: { kind: "shipment", noun: "Shipment", label: "SEED-DET-007" },
      panel: {
        ...EMPTY,
        kind: "Paper",
        paper: {
          customer: "Acme Manufacturing",
          deliveredAt: NOW - 30 * HOUR,
          papers: [{ name: "Proof of delivery", onFile: false }],
          holds: "",
        },
      },
    });

    const pod = (await screen.findByText("Bill of lading")).closest("ul")!.lastElementChild!;
    expect(pod).toHaveTextContent("Proof of delivery");
    expect(pod).toHaveTextContent("Not on file");
    expect(screen.getByText("1 missing")).toBeVisible();

    const file = new File(["pod"], "pod.pdf", { type: "application/pdf" });
    await user.upload(screen.getByLabelText("Choose a file for Proof of delivery"), file);

    await waitFor(() => expect(mocks.resolve).toHaveBeenCalledOnce());
    expect(mocks.upload).toHaveBeenCalledExactlyOnceWith({
      file,
      resourceId: "shp_1",
      resourceType: "shipment",
      documentTypeId: "dt_pod",
      description: undefined,
    });
    expect(mocks.resolve).toHaveBeenCalledWith({
      id: "wt_9",
      version: 4,
      code: "DocumentReceived",
    });
    expect(await screen.findByText("All on file")).toBeVisible();
    // The item leaves the list as it closes, so a toast says what became of it.
    expect(mocks.toast).toHaveBeenCalledExactlyOnceWith(
      "Proof of delivery filed on SEED-DET-007. Nothing else is missing, so it is cleared from Needs you.",
    );
  });

  it("offers the way to where a thing is fixed, and none for a message, which Triage answers", () => {
    show({
      path: "/dispatch/console",
      panel: {
        ...EMPTY,
        kind: "Coverage",
        coverage: {
          startsAt: NOW + 6 * HOUR,
          reason: "No driver has the hours.",
          exhausted: false,
          endedAt: 0,
        },
      },
    });

    expect(screen.getByRole("link", { name: "Cover it in dispatch" })).toHaveAttribute(
      "href",
      "/dispatch/console",
    );
    expect(screen.getByText("No driver has the hours.")).toBeVisible();
  });

  it("shows a change as the value it had beside the value it has", () => {
    show({
      panel: {
        ...EMPTY,
        kind: "Change",
        change: {
          who: "Billing desk",
          what: "Evaluation score",
          from: "82%",
          to: "61%",
          ref: "",
          at: NOW - HOUR,
        },
      },
    });

    expect(screen.getByText("Was").parentElement).toHaveTextContent("82%");
    expect(screen.getByText("Now").parentElement).toHaveTextContent("61%");
  });

  it("lists credentials soonest first, and says which have run out", () => {
    show({
      panel: {
        ...EMPTY,
        kind: "Expiry",
        expiry: {
          who: "Robert Brown",
          papers: [
            { name: "Hazmat endorsement", expiresAt: NOW + 30 * 24 * HOUR, daysLeft: 30 },
            { name: "Medical card", expiresAt: NOW - 2 * 24 * HOUR, daysLeft: -2 },
          ],
        },
      },
    });

    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("Medical card");
    expect(rows[0]).toHaveTextContent("Expired 2 days ago");
    expect(rows[1]).toHaveTextContent("30 days left");
    expect(screen.getByText("1 expired")).toBeVisible();
  });

  it("reads an item with no panel from its facts, in the same frame", () => {
    show({
      panel: null,
      kindLabel: "Service failure",
      impact: ["FreshHaul Foods"],
      facts: [{ label: "Late by", value: "42 minutes", at: null }],
      trail: [],
    });

    expect(screen.getByText("Service failure")).toBeVisible();
    expect(screen.getByRole("heading", { name: "FreshHaul Foods" })).toBeVisible();
    expect(screen.getByText("42 minutes")).toBeVisible();
  });

  it("draws nothing for an item that reported nothing", () => {
    const { container } = show({ panel: null, impact: [], facts: [], trail: [] });

    expect(container).toBeEmptyDOMElement();
  });
});
