import type { ActivityEntry } from "@/lib/graphql/record-activity";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type {
  RateAgreement,
  RateAgreementLaneSummary,
  RateAgreementRule,
} from "@trenova/shared/types/rate";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskRateAgreement } from "../desk-rate-agreement";

const service = vi.hoisted(() => ({
  getWithoutLanes: vi.fn(),
  listLanes: vi.fn(),
  listVersions: vi.fn(),
  laneSummary: vi.fn(),
}));
const fetchers = vi.hoisted(() => ({ pricing: vi.fn(), options: vi.fn(), activity: vi.fn() }));

vi.mock("@/services/api", () => ({ apiService: { rateAgreementService: service } }));
vi.mock("@/lib/graphql/rate-agreement-card", () => ({
  fetchRateAgreementPricing: fetchers.pricing,
}));
vi.mock("@/lib/graphql/record-activity", () => ({ fetchRecordActivity: fetchers.activity }));
vi.mock("@/lib/graphql/select-options", () => ({
  fetchGraphQLSelectOptions: fetchers.options,
}));
// The cards a listed record opens as are their own chunks; here each stands in
// as what it was given, with its way back.
function standIn(kind: string) {
  return ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>
        {kind} card {recordId}
      </p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  );
}
vi.mock("../../customer/desk-customer", () => ({
  DeskCustomer: standIn("Customer"),
  DeskCustomerArtifact: () => null,
}));
vi.mock("../../formula-template/desk-formula-template", () => ({
  DeskFormulaTemplate: standIn("Formula template"),
  DeskFormulaTemplateArtifact: () => null,
}));

const DAY = 86_400;
const NOW = Math.floor(Date.now() / 1000);

/** The names select-options answers with, by id. */
const NAMES: Record<string, string> = {
  cus_1: "Acme Foods",
  ft_mile: "Per mile",
  acc_det: "Detention",
  fsp_1: "DOE national",
  usr_1: "Avery Lane",
  usr_2: "Robin Hale",
  st_ga: "Georgia",
  st_tx: "Texas",
};

function lane(overrides: Partial<RateAgreementRule> = {}): RateAgreementRule {
  return {
    id: "rule_1",
    label: "",
    status: "Active",
    originScopeType: "CityState",
    originScopeValue: "st_ga",
    originCity: "Atlanta",
    destinationScopeType: "State",
    destinationScopeValue: "st_tx",
    destinationCity: "",
    direction: "Directional",
    formulaTemplateId: "ft_mile",
    rate: 2.1,
    minCharge: 250,
    freightClassSource: "Commodity",
    allowDeficitRating: true,
    daysOfWeek: 0,
    hazmatOnly: false,
    tempControlOnly: false,
    priority: 0,
    effectiveFrom: NOW - 30 * DAY,
    breaks: [],
    ...overrides,
  } as RateAgreementRule;
}

/** An agreement as GET /rate-agreements/{id}/?includeChildren=true returns it, after the client schema. */
function agreement(overrides: Partial<RateAgreement> = {}): RateAgreement {
  return {
    id: "ra_1",
    partyType: "Customer",
    customerId: "cus_1",
    carrierId: null,
    code: "ACME-2026",
    name: "Acme dry van contract",
    description: "Southeast dry van lanes",
    agreementType: "Contract",
    status: "Active",
    contractRef: "MSA-114",
    priority: 0,
    effectiveFrom: NOW - 100 * DAY,
    effectiveTo: NOW + 20 * DAY,
    autoRenew: false,
    renewalNoticeDays: 30,
    currency: "USD",
    defaultMinCharge: 200,
    defaultMaxCharge: null,
    roundingMode: "HalfUp",
    roundingPrecision: 2,
    billToCustomerId: null,
    marginFloorPercent: 12,
    maxPayPercentOfSell: null,
    submittedById: "usr_1",
    submittedAt: NOW - 110 * DAY,
    approvedById: "usr_2",
    approvedAt: NOW - 105 * DAY,
    reviewComment: "",
    currentVersionNumber: 3,
    createdAt: NOW - 120 * DAY,
    rules: [],
    accessorials: [
      {
        accessorialChargeId: "acc_det",
        method: "PerUnit",
        rateUnit: "Hour",
        amount: 45,
        freeUnits: 2,
        waived: false,
        autoApply: true,
        applyCondition: "",
      },
    ],
    fuelBinding: { fuelSurchargeProgramId: "fsp_1", waived: false, capAmount: 0.6 },
    versions: [],
    ...overrides,
  } as RateAgreement;
}

/** Answers GET /rate-agreements/{id}/lanes/ from a list of live lanes, a page at a time. */
function lanePages(live: readonly RateAgreementRule[], retired = 0) {
  return async (_id: string, page: { limit: number; offset: number }) => ({
    results: live.slice(page.offset, page.offset + page.limit),
    count: live.length,
    retired,
    limit: page.limit,
    offset: page.offset,
  });
}

/** Lanes numbered from one, each its own lane. */
function manyLanes(count: number): RateAgreementRule[] {
  return Array.from({ length: count }, (_, index) =>
    lane({ id: `rule_${index + 1}`, label: `Lane ${index + 1}` }),
  );
}

/** The agreement's live lanes by what prices them, as GET /rate-agreements/{id}/lanes/summary/ returns them. */
function summary(overrides: Partial<RateAgreementLaneSummary> = {}): RateAgreementLaneSummary {
  return {
    count: 40,
    groups: [
      {
        formulaTemplateId: "ft_mile",
        formulaTemplateName: "Per mile",
        rateMatrixId: null,
        rateMatrixName: "",
        currency: "USD",
        lanes: 30,
        minRate: "2.1",
        maxRate: "3.45",
      },
      {
        formulaTemplateId: null,
        formulaTemplateName: "",
        rateMatrixId: "rm_1",
        rateMatrixName: "Southeast zones",
        currency: "CAD",
        lanes: 10,
        minRate: null,
        maxRate: null,
      },
    ],
    ...overrides,
  };
}

const entry = {
  id: "ae_1",
  operation: "update",
  timestamp: NOW - 3_600,
  changes: { status: { from: "PendingApproval", to: "Active" } },
  comment: null,
  principalType: "session_user",
  user: { id: "usr_2", name: "Robin Hale" },
} as ActivityEntry;

const EVERYTHING = {
  [Resource.RateAgreement]: Operation.Read,
  [Resource.AuditLog]: Operation.Read,
  [Resource.RateQuote]: Operation.Read,
  [Resource.Customer]: Operation.Read,
  [Resource.FormulaTemplate]: Operation.Read,
  [Resource.ProtectedMargins]: Operation.Read,
};

function draw(props: Partial<Parameters<typeof DeskRateAgreement>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const tree = (more: Partial<typeof props> = {}) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskRateAgreement
          recordId="ra_1"
          order={["ra_0", "ra_1"]}
          {...handlers}
          {...props}
          {...more}
        />
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(tree());
  return {
    ...view,
    ...handlers,
    step: (recordId: string) => view.rerender(tree({ recordId })),
  };
}

async function section(title: string): Promise<HTMLElement> {
  const heading = await screen.findByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  seedPermissions(EVERYTHING);
  for (const fn of [...Object.values(service), ...Object.values(fetchers)]) fn.mockReset();
  service.getWithoutLanes.mockResolvedValue(agreement());
  service.listLanes.mockImplementation(lanePages([lane()], 1));
  service.listVersions.mockResolvedValue([
    {
      id: "rav_2",
      versionNumber: 2,
      changeMessage: "Raised the floor",
      createdById: "usr_1",
      createdAt: NOW - 50 * DAY,
      effectiveFrom: NOW - 50 * DAY,
    },
    {
      id: "rav_3",
      versionNumber: 3,
      changeMessage: "",
      createdById: "usr_2",
      createdAt: NOW - 10 * DAY,
      effectiveFrom: NOW - 10 * DAY,
    },
  ]);
  fetchers.pricing.mockResolvedValue({ loads: 412, lastRatedAt: NOW - 2 * DAY });
  service.laneSummary.mockResolvedValue(summary());
  fetchers.activity.mockResolvedValue({
    rows: [entry],
    total: 6,
    endCursor: "c1",
    hasNextPage: true,
  });
  fetchers.options.mockImplementation(async ({ ids }: { ids: string[] }) => ({
    results: ids.map((id) => ({
      id,
      label: NAMES[id] ?? id,
      description: null,
      meta: id.startsWith("st_") ? { abbreviation: id === "st_ga" ? "GA" : "TX" } : null,
    })),
  }));
});

afterEach(() => {
  clearPermissions();
});

describe("DeskRateAgreement", () => {
  it("draws the next agreement's outline, not the one it left, while the next one is read", async () => {
    const { container, step } = draw();
    await screen.findByRole("heading", { name: "Acme dry van contract", level: 2 });

    service.getWithoutLanes.mockReturnValue(new Promise(() => {}));
    step("ra_0");

    expect(screen.queryByRole("heading", { name: "Acme dry van contract", level: 2 })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("holds the card's outline while the agreement is read, then draws what it is", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the rate agreement…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-path")).not.toBeNull();
    expect(container.querySelectorAll(".dk-rec-skel .dk-ra-mix .dk-rec-gauge")).toHaveLength(2);
    expect(container.querySelector(".dk-rec-skel .dk-rec-sk-log")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Acme dry van contract", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("ACME-2026")).toBeInTheDocument();
    expect(screen.getByText("Contract")).toBeInTheDocument();
    expect(screen.getByText("Customer agreement")).toBeInTheDocument();
    expect(screen.getByText("v3")).toBeInTheDocument();
    expect(screen.getByText("Southeast dry van lanes")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-q .dk-ra-warn")).toHaveTextContent("Ends in 20 days");
    expect(screen.getByText("Contract MSA-114")).toBeInTheDocument();
    expect(await screen.findByText(/^Priced 412 loads, last /)).toBeInTheDocument();
    expect(await screen.findByText("lane priced")).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent("1lane priced");
  });

  it("draws how its lanes are priced, each way as its share of the live lanes", async () => {
    draw();

    const structure = await section("Rate structure");
    const mile = await within(structure).findByRole("meter", { name: "Per mile" });
    expect(mile).toHaveAttribute("aria-valuenow", "75");
    expect(mile.closest(".dk-rec-gauge")).toHaveTextContent("30 lanes2.1 – 3.45");
    const table = within(structure).getByRole("meter", {
      name: "Rate table Southeast zones · CAD",
    });
    expect(table).toHaveAttribute("aria-valuenow", "25");
    expect(table.closest(".dk-rec-gauge")).toHaveTextContent(/10 lanes$/u);
    expect(service.laneSummary).toHaveBeenCalledWith("ra_1", expect.anything());
  });

  it("says how its lanes are priced could not be read, and the rest of the card stands", async () => {
    service.laneSummary.mockRejectedValue(new Error("Lane summary is unavailable"));
    draw();

    expect(await screen.findByText("Lane summary is unavailable")).toBeInTheDocument();
    expect(await within(await section("Lanes")).findByText("Per mile")).toBeInTheDocument();
  });

  it("lists what has been done to it, and reads earlier activity on request", async () => {
    draw();

    const activity = await section("Activity");
    expect(await within(activity).findByText("Changed status")).toBeInTheDocument();
    expect(within(activity).getByText(/^Robin Hale · /u)).toBeInTheDocument();
    expect(within(activity).getByText("6 changes")).toBeInTheDocument();
    expect(fetchers.activity).toHaveBeenCalledWith(
      "ra_1",
      { first: 5, after: null },
      expect.anything(),
    );

    fetchers.activity.mockResolvedValue({
      rows: [{ ...entry, id: "ae_2", operation: "create", changes: null }],
      total: null,
      endCursor: null,
      hasNextPage: false,
    });
    fireEvent.click(within(activity).getByRole("button", { name: "Show more" }));
    expect(await within(activity).findByText("Created")).toBeInTheDocument();
    expect(fetchers.activity).toHaveBeenLastCalledWith(
      "ra_1",
      { first: 5, after: "c1" },
      expect.anything(),
    );
  });

  it("stands with nothing around it: no lanes, no charges, no versions, no activity", async () => {
    service.getWithoutLanes.mockResolvedValue(
      agreement({ accessorials: [], fuelBinding: null, currentVersionNumber: 1 }),
    );
    service.listLanes.mockImplementation(lanePages([]));
    service.listVersions.mockResolvedValue([]);
    service.laneSummary.mockResolvedValue({ count: 0, groups: [] });
    fetchers.activity.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    draw();

    expect(
      await screen.findByText("Once a lane prices freight, how each is priced shows here."),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("No lane prices freight on this agreement yet."),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("Nothing has been done to this agreement yet."),
    ).toBeInTheDocument();
    expect(await screen.findByText("lanes priced")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Charges" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Versions" })).toBeNull();
  });

  it("leads with the term on its track, coloured by where it stands", async () => {
    const { container } = draw();

    const term = await screen.findByRole("meter", { name: "Term elapsed" });
    expect(term).toHaveAttribute("aria-valuenow", "83");
    expect(container.querySelector(".dk-ra-now")).toHaveClass("dk-ra-notice");
    // The notice's span on the track has a class of its own: the panel's state
    // classes are the panel's, and one shared with the span lifts the panel
    // out of the card along with it.
    expect(container.querySelectorAll(".dk-ra-notice")).toHaveLength(1);
    expect(term.querySelector(".dk-ra-notice-zone")).not.toBeNull();
    expect(within(term).getByText("Today")).toBeInTheDocument();
    const panel = container.querySelector(".dk-ra-now") as HTMLElement;
    expect(within(panel).getByText("Started")).toBeInTheDocument();
    expect(within(panel).getByText("Ends in 20 days")).toHaveClass("dk-ra-chip", "dk-lead");
    expect(within(panel).getByText("30 days' notice")).toBeInTheDocument();
  });

  it("walks the approval path down a rail, with who took each step and when", async () => {
    draw();

    const path = await screen.findByRole("list", { name: "Approval" });
    const steps = within(path).getAllByRole("listitem");
    expect(steps.map((step) => step.className)).toEqual(["dk-done", "dk-done", "dk-done"]);
    expect(steps.map((step) => step.querySelector("b")?.textContent)).toEqual([
      "Drafted",
      "Submitted for review",
      "Approved",
    ]);
    expect(await within(path).findByText(/Avery Lane$/)).toBeInTheDocument();
    expect(within(path).getByText(/Robin Hale$/)).toBeInTheDocument();
  });

  it("waits on a reviewer while in review, and pauses a suspended agreement", async () => {
    service.getWithoutLanes.mockResolvedValue(agreement({ status: "InReview", approvedAt: null }));
    const { container, unmount } = draw();

    const path = await screen.findByRole("list", { name: "Approval" });
    expect(
      within(path)
        .getAllByRole("listitem")
        .map((step) => step.className),
    ).toEqual(["dk-done", "dk-now", "dk-todo"]);
    expect(within(path).getByText("Waiting on a reviewer")).toBeInTheDocument();
    expect(container.querySelector(".dk-ra-now")).toHaveClass("dk-ra-pending");
    expect(screen.getByText("Prices freight once approved")).toBeInTheDocument();
    unmount();

    service.getWithoutLanes.mockResolvedValue(agreement({ status: "Suspended" }));
    const held = draw();
    const heldPath = await screen.findByRole("list", { name: "Approval" });
    expect(within(heldPath).getAllByRole("listitem").at(-1)).toHaveClass("dk-now", "dk-held");
    expect(held.container.querySelector(".dk-ra-now")).toHaveClass("dk-ra-held");
    expect(screen.getByText("Prices nothing while suspended")).toBeInTheDocument();
  });

  it("lists each lane by its places and price, leaving retired lanes out", async () => {
    draw();

    const lanes = await section("Lanes");
    await waitFor(() =>
      expect(lanes.querySelector(".dk-ra-places b")).toHaveTextContent("Atlanta, GA → TX"),
    );
    expect(within(lanes).getByText("2.1")).toBeInTheDocument();
    expect(within(lanes).getByText("Min $250.00")).toBeInTheDocument();
    expect(within(lanes).getByText("1 retired")).toBeInTheDocument();
    expect(lanes.querySelectorAll(".dk-ra-lane")).toHaveLength(1);
    expect(service.listLanes).toHaveBeenCalledWith(
      "ra_1",
      { limit: 8, offset: 0 },
      expect.anything(),
    );
  });

  // An agreement can price tens of thousands of lanes. The card reads none of
  // them with the agreement: it reads the most specific page first, counts the
  // rest from the server, and reads each next page only when asked.
  it("reads its lanes a page at a time, counting all of them from the server", async () => {
    service.listLanes.mockImplementation(lanePages(manyLanes(20)));
    const { container } = draw();

    const lanes = await section("Lanes");
    expect(await within(lanes).findByText("8 of 20 shown")).toBeInTheDocument();
    expect(lanes.querySelectorAll(".dk-ra-lane")).toHaveLength(8);
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent("20lanes priced");

    fireEvent.click(within(lanes).getByRole("button", { name: "Show 8 more lanes" }));
    await waitFor(() => expect(lanes.querySelectorAll(".dk-ra-lane")).toHaveLength(16));
    expect(service.listLanes).toHaveBeenLastCalledWith(
      "ra_1",
      { limit: 8, offset: 8 },
      expect.anything(),
    );

    fireEvent.click(within(lanes).getByRole("button", { name: "Show 4 more lanes" }));
    await waitFor(() => expect(lanes.querySelectorAll(".dk-ra-lane")).toHaveLength(20));
    expect(within(lanes).queryByRole("button", { name: /^Show / })).toBeNull();
    expect(within(lanes).queryByText(/ shown$/)).toBeNull();
    expect(service.getWithoutLanes).toHaveBeenCalledWith("ra_1", expect.anything());
  });

  it("writes a count in the thousands as a number a person reads", async () => {
    service.listLanes.mockImplementation(async () => ({
      results: manyLanes(8),
      count: 12_500,
      retired: 0,
      limit: 8,
      offset: 0,
    }));
    const { container } = draw();

    await waitFor(() =>
      expect(container.querySelector(".dk-rec-amt")).toHaveTextContent("12,500lanes priced"),
    );
    expect(await screen.findByText("8 of 12,500 shown")).toBeInTheDocument();
  });

  it("says the lanes could not be read in their own place, and the rest of the card stands", async () => {
    service.listLanes.mockRejectedValue(new Error("Lanes are unavailable"));
    const { container } = draw();

    const lanes = await section("Lanes");
    expect(await within(lanes).findByText("Lanes are unavailable")).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent("—");
    expect(await screen.findByRole("list", { name: "Approval" })).toBeInTheDocument();
  });

  it("states its charges, its fuel and its limits", async () => {
    draw();

    const charges = await section("Charges");
    expect(await within(charges).findByText("Detention")).toBeInTheDocument();
    expect(
      within(charges).getByText("$45.00 per hour · 2 free · Added automatically"),
    ).toBeInTheDocument();
    expect(await within(charges).findByText("Fuel · DOE national")).toBeInTheDocument();
    expect(within(charges).getByText("Capped at $0.60")).toBeInTheDocument();

    const limits = await section("Limits");
    expect(within(limits).getByText("$200.00")).toBeInTheDocument();
    expect(within(limits).getByText("12%")).toBeInTheDocument();
  });

  it("keeps the margin floor off the card for a role that withholds margins", async () => {
    const { [Resource.ProtectedMargins]: _margins, ...withoutMargins } = EVERYTHING;
    seedPermissions(withoutMargins);
    draw();

    const limits = await section("Limits");
    expect(within(limits).getByText("$200.00")).toBeInTheDocument();
    expect(within(limits).queryByText("12%")).not.toBeInTheDocument();
  });

  it("lists its versions, oldest first, with who cut each", async () => {
    draw();

    const versions = await screen.findByRole("heading", { name: "Versions", level: 3 });
    const entries = within(versions.closest("section") as HTMLElement).getAllByRole("listitem");
    expect(entries.map((entry) => entry.textContent)).toEqual([
      expect.stringContaining("v2 · Raised the floor"),
      expect.stringContaining("v3 · No note"),
    ]);
  });

  // A record the card names opens as a table's row does: as its own card, in
  // this card's place, and the way back returns to the agreement.
  it("opens its customer in the card's place, and goes back to the agreement", async () => {
    draw();

    fireEvent.click(
      await within(await section("Customer")).findByRole("button", { name: /Acme Foods/ }),
    );

    expect(await screen.findByText("Customer card cus_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to Acme dry van contract" }));
    expect(
      await screen.findByRole("heading", { name: "Acme dry van contract", level: 2 }),
    ).toBeInTheDocument();
  });

  it("opens the formula template a lane is priced by in the card's place", async () => {
    draw();

    fireEvent.click(
      await within(await section("Lanes")).findByRole("button", { name: "Per mile" }),
    );

    expect(await screen.findByText("Formula template card ft_mile")).toBeInTheDocument();
  });

  // Someone who may read the agreement but not quotes or customers sees the
  // card without what those would add, and nothing is asked for them.
  it("opens with only what the person may read", async () => {
    seedPermissions({ [Resource.RateAgreement]: Operation.Read });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Acme dry van contract", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/^Priced /)).toBeNull();
    expect(fetchers.pricing).not.toHaveBeenCalled();
    expect(screen.queryByRole("heading", { name: "Activity" })).toBeNull();
    expect(fetchers.activity).not.toHaveBeenCalled();

    // The customer is a linked record: without customer read it is named, locked, not linked.
    const customers = await section("Customer");
    expect(within(customers).queryByRole("link", { name: /Acme Foods/ })).toBeNull();
    const customer = (await within(customers).findByText("Acme Foods")).closest(
      '[data-slot="locked-reference"]',
    );
    expect(customer?.getAttribute("aria-description")).toBe("You don't have access to customers");
    const lanes = await section("Lanes");
    expect(await within(lanes).findByText("Per mile")).not.toHaveRole("button");
  });

  it("leaves the outline of activity the person may not read out of the skeleton", () => {
    seedPermissions({ [Resource.RateAgreement]: Operation.Read });
    service.getWithoutLanes.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(container.querySelector(".dk-rec-skel .dk-ra-mix")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-rec-sk-log")).toBeNull();
  });

  it("says when an agreement has no end, and when it has never priced a load", async () => {
    service.getWithoutLanes.mockResolvedValue(agreement({ effectiveTo: null }));
    fetchers.pricing.mockResolvedValue({ loads: 0, lastRatedAt: null });
    const { container } = draw();

    expect(await screen.findByText("Runs on")).toBeInTheDocument();
    expect(screen.queryByRole("meter", { name: "Term elapsed" })).toBeNull();
    expect(container.querySelector(".dk-ra-track")).toHaveClass("dk-open");
    expect(container.querySelector(".dk-ra-end.dk-to b")).toHaveTextContent("No end date");
    expect(await screen.findByText("Has not priced a load yet")).toBeInTheDocument();
  });

  it("leaves an open term that has not begun unfilled, with today not yet on it", async () => {
    service.getWithoutLanes.mockResolvedValue(
      agreement({
        status: "Draft",
        approvedAt: null,
        effectiveFrom: NOW + 14 * DAY,
        effectiveTo: null,
      }),
    );
    const { container } = draw();

    expect(await screen.findByText("Starts")).toBeInTheDocument();
    expect(container.querySelector(".dk-ra-track")).toHaveClass("dk-open");
    expect(container.querySelector<HTMLElement>(".dk-ra-done")?.style.width).toBe("0%");
    expect(
      within(container.querySelector(".dk-ra-now") as HTMLElement).queryByText("Today"),
    ).toBeNull();
  });

  it("puts a reviewer's comment on the step the agreement was sent back to", async () => {
    service.getWithoutLanes.mockResolvedValue(
      agreement({ status: "Draft", approvedAt: null, reviewComment: "Fuel cap is too low" }),
    );
    draw();

    const comment = await screen.findByText("Fuel cap is too low");
    expect(comment.closest("li")).toHaveClass("dk-now");
    expect(comment.closest("li")?.querySelector("b")).toHaveTextContent("Drafted");
    expect(screen.getByText("Draft")).toBeInTheDocument();
  });

  it("says when the agreement cannot be read, with the way back", async () => {
    service.getWithoutLanes.mockRejectedValue(new Error("Rate agreement not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This rate agreement could not be read" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });
});
