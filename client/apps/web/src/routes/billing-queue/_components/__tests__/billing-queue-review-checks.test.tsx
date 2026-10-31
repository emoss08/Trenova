import { billingQueue } from "@/lib/queries/billing-queue";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type {
  BillingCheck,
  BillingIssue,
  BillingQueueItem,
} from "@trenova/shared/types/billing-queue";
import { MotionConfig } from "motion/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueReviewChecks } from "../billing-queue-review-checks";

const mocks = vi.hoisted(() => ({
  resolveIssue: vi.fn(),
  undoIssue: vi.fn(),
  selectDocument: vi.fn(),
  canUpdate: true,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canUpdate, isLoading: false }),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    billingQueueService: { resolveIssue: mocks.resolveIssue, undoIssue: mocks.undoIssue },
  },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.canUpdate = true;
});

const passing = (key: BillingCheck["key"], code: string): BillingCheck => ({
  key,
  state: "ok",
  code,
  detail: "",
  issueId: null,
  facts: null,
});

const fuelIssue: BillingIssue = {
  id: "bqis_fuel",
  itemId: "bqi_1",
  checkKey: "charges",
  code: "AccessorialNotOnRateCon",
  subjectKey: "acg_fuel",
  summary: "$1,037.45 fuel surcharge isn't on the rate con",
  reasoning:
    "Miller & Rose Industries's rate con doesn't price a fuel surcharge. It was added to the shipment by hand, so bill it only if the contract passes it through or there's a receipt for it.",
  source: "Deterministic",
  flaggedChargeId: "acg_fuel",
  options: [
    {
      key: "keep",
      label: "Keep it · bill $1,037.45",
      amount: 1037.45,
      done: "Fuel surcharge kept",
      effect: { kind: "keep", chargeId: "acg_fuel" },
    },
    {
      key: "drop",
      label: "Remove it",
      amount: null,
      done: "Fuel surcharge removed",
      effect: { kind: "drop", chargeId: "acg_fuel" },
    },
  ],
  resolutionKey: null,
  resolutionText: null,
  resolvedById: null,
  resolvedAt: null,
  requestedAt: null,
  undoable: false,
  createdAt: 1_800_000_000,
};

const settledFuel: BillingIssue = {
  ...fuelIssue,
  resolutionKey: "keep",
  resolutionText: "Fuel surcharge kept",
  resolvedById: "usr_1",
  resolvedAt: 1_800_000_100,
  undoable: true,
};

function item({
  status = "InReview",
  checks,
  issues,
  blocker = "",
}: {
  status?: BillingQueueItem["status"];
  checks: BillingCheck[];
  issues: BillingIssue[];
  blocker?: string;
}): BillingQueueItem {
  return {
    id: "bqi_1",
    shipmentId: "shp_1",
    status,
    detentionHolds: [],
    ratingReviewHold: null,
    agentReads: [],
    review: {
      checks,
      issues,
      needsCount:
        issues.filter((issue) => !issue.resolutionKey).length +
        checks.filter((check) => check.state !== "ok" && check.code !== "issue").length,
      ready: checks.every((check) => check.state === "ok"),
      blocker,
      documents: [],
      billerName: "Dana Whitfield",
    },
  } as unknown as BillingQueueItem;
}

const flagged = item({
  checks: [
    passing("biller", "assigned"),
    { key: "charges", state: "fail", code: "issue", detail: "", issueId: "bqis_fuel", facts: null },
    passing("pod", "not_required"),
    passing("terms", "terms"),
    passing("duplicate", "unique"),
  ],
  issues: [fuelIssue],
  blocker: "issue",
});

const stopOffIssue: BillingIssue = {
  ...fuelIssue,
  id: "bqis_stop",
  subjectKey: "acg_stop",
  summary: "$100.00 stop-off charge isn't on the rate con",
  reasoning: null,
  flaggedChargeId: "acg_stop",
  options: [
    {
      key: "keep",
      label: "Keep it · bill $100.00",
      amount: 100,
      done: "Stop-off charge kept",
      effect: { kind: "keep", chargeId: "acg_stop" },
    },
    {
      key: "drop",
      label: "Remove it",
      amount: null,
      done: "Stop-off charge removed",
      effect: { kind: "drop", chargeId: "acg_stop" },
    },
  ],
  createdAt: 1_800_000_050,
};

const twoCharges = item({
  checks: [
    passing("biller", "assigned"),
    { key: "charges", state: "warn", code: "issue", detail: "", issueId: "bqis_fuel", facts: null },
    passing("pod", "not_required"),
    passing("terms", "terms"),
    passing("duplicate", "unique"),
  ],
  issues: [fuelIssue, stopOffIssue],
  blocker: "issue",
});

// A check settled by its last issue names that issue, as the server sends it.
const cleared = item({
  checks: [
    passing("biller", "assigned"),
    { ...passing("charges", "resolved"), detail: "Fuel surcharge kept", issueId: "bqis_fuel" },
    passing("pod", "not_required"),
    passing("terms", "terms"),
    passing("duplicate", "unique"),
  ],
  issues: [settledFuel],
});

function renderChecks(value: BillingQueueItem, attention = 0, reducedMotion = false) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const view = (next: BillingQueueItem, calls: number) => (
    <MotionConfig reducedMotion={reducedMotion ? "always" : "never"}>
      <QueryClientProvider client={client}>
        <BillingQueueReviewChecks
          item={next}
          attention={calls}
          onDocumentSelect={mocks.selectDocument}
        />
      </QueryClientProvider>
    </MotionConfig>
  );
  const { rerender } = render(view(value, attention));
  return {
    client,
    invalidate,
    rerender: (next: BillingQueueItem, calls = attention) => rerender(view(next, calls)),
  };
}

describe("BillingQueueReviewChecks", () => {
  it("shows nothing for an item that is not in review, or was read without its review", () => {
    renderChecks({ ...flagged, status: "ReadyForReview" });
    expect(screen.queryByTestId("billing-queue-review-checks")).not.toBeInTheDocument();

    renderChecks({ ...flagged, review: null } as BillingQueueItem);
    expect(screen.queryByTestId("billing-queue-review-checks")).not.toBeInTheDocument();
  });

  it("names each flagged check with what was found, why, and the choices that settle it", () => {
    renderChecks(flagged);

    expect(screen.getByText("1 needs you")).toBeInTheDocument();
    const check = screen.getByTestId("billing-queue-check-charges");
    expect(check).toHaveTextContent("Charges$1,037.45 fuel surcharge isn't on the rate con");
    expect(check).toHaveTextContent("rate con doesn't price a fuel surcharge");
    expect(within(check).getByRole("button", { name: "Keep it · bill $1,037.45" })).toBeEnabled();
    expect(within(check).getByRole("button", { name: "Remove it" })).toBeEnabled();
    expect(screen.queryByTestId("billing-queue-check-pod")).not.toBeInTheDocument();
  });

  it("shows one flagged charge at a time, turning through them with the chevrons", async () => {
    renderChecks(twoCharges);

    expect(screen.getByText("2 need you")).toBeInTheDocument();
    const check = screen.getByTestId("billing-queue-check-charges");
    expect(check).toHaveTextContent("Charges$1,037.45 fuel surcharge isn't on the rate con1 of 2");
    expect(check).not.toHaveTextContent("stop-off");
    expect(within(check).getByRole("button", { name: "Keep it · bill $1,037.45" })).toBeEnabled();
    expect(within(check).getAllByRole("button", { name: "Remove it" })).toHaveLength(1);
    expect(within(check).getByRole("button", { name: "Previous issue" })).toBeDisabled();

    await userEvent.click(within(check).getByRole("button", { name: "Next issue" }));

    expect(check).toHaveTextContent("Charges$100.00 stop-off charge isn't on the rate con2 of 2");
    expect(check).not.toHaveTextContent("fuel surcharge");
    expect(within(check).getByRole("button", { name: "Keep it · bill $100.00" })).toBeEnabled();
    expect(within(check).getByRole("button", { name: "Next issue" })).toBeDisabled();

    await userEvent.click(within(check).getByRole("button", { name: "Previous issue" }));
    expect(check).toHaveTextContent("$1,037.45 fuel surcharge isn't on the rate con1 of 2");
  });

  it("settles the charge shown", async () => {
    mocks.resolveIssue.mockResolvedValue(flagged);
    renderChecks(twoCharges);

    const check = screen.getByTestId("billing-queue-check-charges");
    await userEvent.click(within(check).getByRole("button", { name: "Next issue" }));
    await userEvent.click(within(check).getByRole("button", { name: "Remove it" }));

    await waitFor(() =>
      expect(mocks.resolveIssue).toHaveBeenCalledWith("bqi_1", "bqis_stop", "drop"),
    );
  });

  it("shows the one left in place of a charge that settled, without the chevrons", async () => {
    const { rerender } = renderChecks(twoCharges);
    const check = screen.getByTestId("billing-queue-check-charges");
    await userEvent.click(within(check).getByRole("button", { name: "Next issue" }));

    rerender(
      item({
        checks: twoCharges.review!.checks,
        issues: [fuelIssue, { ...stopOffIssue, resolutionKey: "drop", resolvedAt: 1_800_000_200 }],
        blocker: "issue",
      }),
    );

    expect(check).toHaveTextContent("Charges$1,037.45 fuel surcharge isn't on the rate con");
    expect(within(check).queryByRole("button", { name: "Next issue" })).not.toBeInTheDocument();
    expect(within(check).getByRole("button", { name: "Keep it · bill $1,037.45" })).toBeEnabled();
  });

  it("offers no choices on a check its last issue settled, only what was done", () => {
    renderChecks(cleared);

    const check = screen.getByTestId("billing-queue-check-charges");
    expect(check).toHaveTextContent("ChargesFuel surcharge kept");
    expect(within(check).queryByRole("button")).not.toBeInTheDocument();
  });

  it("settles the check with the chosen option and puts the answered item in the cache", async () => {
    mocks.resolveIssue.mockResolvedValue(cleared);
    const { client, invalidate } = renderChecks(flagged);

    await userEvent.click(screen.getByRole("button", { name: "Keep it · bill $1,037.45" }));

    await waitFor(() =>
      expect(client.getQueryData(billingQueue.get("bqi_1").queryKey)).toEqual(cleared),
    );
    expect(mocks.resolveIssue).toHaveBeenCalledWith("bqi_1", "bqis_fuel", "keep");
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billingQueue"] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billing-queue-list"] });
  });

  it("says every check is clear, and offers to undo what was settled", async () => {
    mocks.undoIssue.mockResolvedValue(flagged);
    renderChecks(cleared);

    expect(screen.getByText("All five clear")).toBeInTheDocument();
    expect(screen.getByText("Settled · Fuel surcharge kept")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Undo" }));

    await waitFor(() => expect(mocks.undoIssue).toHaveBeenCalledWith("bqi_1", "bqis_fuel"));
  });

  it("shows the checks without the choices to someone who can't update the queue", () => {
    mocks.canUpdate = false;
    renderChecks(flagged);

    expect(screen.getByTestId("billing-queue-check-charges")).toHaveTextContent(
      "$1,037.45 fuel surcharge isn't on the rate con",
    );
    expect(screen.queryByRole("button", { name: "Remove it" })).not.toBeInTheDocument();
  });

  it("shows what Nova made of the item under the checks, a concern in the same ink as agreement", () => {
    renderChecks({
      ...cleared,
      agentVerdict: "Concern",
      agentSummary: "Fuel is 18% above this lane's usual",
      agentReviewedAt: Math.floor(Date.now() / 1000) - 360,
    } as BillingQueueItem);

    const note = screen.getByTestId("billing-queue-nova-note");
    expect(note).toHaveTextContent("Nova · Fuel is 18% above this lane's usual");
    expect(note).toHaveTextContent(/Reviewed .+/u);
    expect(within(note).getByText("Nova").tagName).toBe("B");
    const summary = within(note).getByText("Nova").parentElement;
    expect(summary).toHaveClass("text-foreground");
    expect(summary).not.toHaveClass("text-warning-foreground");
    expect(
      within(note).queryByRole("button", { name: "See what it read" }),
    ).not.toBeInTheDocument();
  });

  it("opens what Nova read as rows like the checks, and a document from it beside the item", async () => {
    const user = userEvent.setup();
    renderChecks({
      ...cleared,
      billToCustomer: { name: "Acme Foods" },
      agentVerdict: "Concern",
      agentSummary: "I also found a $185 lumper receipt that is not on the invoice.",
      agentReviewedAt: Math.floor(Date.now() / 1000) - 360,
      agentReads: [
        { kind: "Invoice", count: 3 },
        { kind: "Stops", count: 2 },
        { kind: "Terms" },
        { kind: "BillingNotes" },
        { kind: "Comments", count: 1 },
        { kind: "PastInvoices", count: 12 },
        { kind: "ChargeHistory", code: "DET", count: 4, of: 40 },
        { kind: "ChargeHistory", code: "FREIGHT", count: 40, of: 40 },
        {
          kind: "Document",
          documentId: "doc_lumper",
          name: "Lumper receipt",
          fileName: "lumper.pdf",
          part: 1,
          parts: 1,
        },
        {
          kind: "Document",
          documentId: "doc_ratecon",
          name: "Rate confirmation",
          fileName: "rc-88412.pdf",
          part: 2,
          parts: 3,
        },
        {
          kind: "Document",
          documentId: "doc_scan",
          name: "Proof of delivery",
          fileName: "pod.jpg",
        },
      ],
    } as unknown as BillingQueueItem);

    expect(screen.queryByTestId("billing-queue-nova-reads")).not.toBeInTheDocument();
    const toggle = screen.getByRole("button", { name: "See what it read" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    await user.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    const panel = await screen.findByTestId("billing-queue-nova-reads");
    const row = (label: string) => {
      const term = within(panel).getByText(label);
      return `${term.textContent}: ${term.nextElementSibling?.textContent}`;
    };
    expect(row("Paperwork")).toBe(
      "Paperwork: Lumper receipt · Rate confirmation, 2 of 3 parts · Proof of delivery, no text",
    );
    expect(row("Shipment")).toBe("Shipment: 3 charges · 2 stops with times · 1 comment");
    expect(row("Payer")).toBe("Payer: Terms · Billing notes");
    expect(row("History")).toBe("History: Last 12 invoices · DET on 4 of 40 · Freight on 40 of 40");

    await user.click(within(panel).getByRole("button", { name: "Rate confirmation" }));
    expect(mocks.selectDocument).toHaveBeenCalledWith("doc_ratecon", "rc-88412.pdf");

    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    await waitFor(() =>
      expect(screen.queryByTestId("billing-queue-nova-reads")).not.toBeInTheDocument(),
    );
  });

  it("leaves out a row of what Nova read when it read nothing there", async () => {
    const user = userEvent.setup();
    renderChecks({
      ...cleared,
      agentVerdict: "Agrees",
      agentSummary: "The charges match the rate con.",
      agentReads: [{ kind: "Invoice", count: 1 }],
    } as unknown as BillingQueueItem);

    await user.click(screen.getByRole("button", { name: "See what it read" }));

    const panel = await screen.findByTestId("billing-queue-nova-reads");
    expect(panel.querySelectorAll("dt")).toHaveLength(1);
    expect(panel).toHaveTextContent("Shipment1 charge");
    expect(panel.querySelector("dd")?.textContent).toBe("1 charge");
  });

  it("still shows Nova's note on an item past review, without the checks", () => {
    renderChecks({
      ...flagged,
      status: "Approved",
      agentVerdict: "Agrees",
      agentSummary: "Matches the contract and the lane's history",
      agentReviewedAt: null,
    } as BillingQueueItem);

    expect(screen.queryByTestId("billing-queue-review-checks")).not.toBeInTheDocument();
    expect(screen.getByTestId("billing-queue-nova-note")).toHaveTextContent(
      "Matches the contract and the lane's history",
    );
  });

  it("shows no Nova note when Nova found nothing beyond what the checks found", () => {
    renderChecks({ ...flagged, agentVerdict: "Concern", agentSummary: "" } as BillingQueueItem);
    expect(screen.queryByTestId("billing-queue-nova-note")).not.toBeInTheDocument();
  });

  it("shows no Nova note when Nova never reviewed the item", () => {
    renderChecks(cleared);
    expect(screen.queryByTestId("billing-queue-nova-note")).not.toBeInTheDocument();
  });

  it("comes into view and pulses the checks that wait on the biller when they are sent here", () => {
    const scrollIntoView = vi.fn();
    window.HTMLElement.prototype.scrollIntoView = scrollIntoView;
    const { rerender } = renderChecks(twoCharges, 3);

    expect(scrollIntoView).not.toHaveBeenCalled();
    expect(screen.getByTestId("billing-queue-check-charges").className).not.toMatch(
      /animate-attention/u,
    );

    rerender(twoCharges, 4);

    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "smooth", block: "start" });
    expect(screen.getByRole("region", { name: "Review" })).toHaveFocus();
    expect(screen.getByTestId("billing-queue-check-charges")).toHaveClass("animate-attention-a");
    expect(screen.getByTestId("billing-queue-check-biller").className).not.toMatch(
      /animate-attention/u,
    );

    rerender(twoCharges, 5);
    expect(screen.getByTestId("billing-queue-check-charges")).toHaveClass("animate-attention-b");
  });

  it("brings the review into view without a pulse or a smooth scroll for someone who reduced motion", () => {
    const scrollIntoView = vi.fn();
    window.HTMLElement.prototype.scrollIntoView = scrollIntoView;
    const { rerender } = renderChecks(twoCharges, 0, true);

    rerender(twoCharges, 1);

    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "auto", block: "start" });
    expect(screen.getByRole("region", { name: "Review" })).toHaveFocus();
    expect(screen.getByTestId("billing-queue-check-charges").className).not.toMatch(
      /animate-attention/u,
    );
  });
});
