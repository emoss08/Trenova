import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderAccountingPage } from "@/test/accounting-page-mocks";
import { AccountingBriefing } from "../_components/accounting-briefing";

const { fetchAccountingBriefing, authState } = vi.hoisted(() => ({
  fetchAccountingBriefing: vi.fn(),
  authState: { user: { name: "Dana Whitfield" } as { name?: string } | null },
}));

vi.mock("@/lib/graphql/accounts-receivable", () => ({ fetchAccountingBriefing }));
vi.mock("motion/react", () => ({ useReducedMotion: () => true }));
vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (select: (state: typeof authState) => unknown) => select(authState),
}));

type Segment = { text: string; link: string | null; customerId: string | null };
type Action = {
  link: string;
  count: number;
  amountMinor: number;
  customerId: string | null;
  customerName: string | null;
};

const SEGMENTS: Segment[] = [
  {
    text: "Acme Freight missed a promise to pay $4,000",
    link: "BrokenPromise",
    customerId: "cus_acme 01",
  },
  { text: ", ", link: null, customerId: null },
  {
    text: "2 invoices worth $4,200 pass 90 days past due this week",
    link: "CrossingNinety",
    customerId: null,
  },
  { text: " and ", link: null, customerId: null },
  { text: "Acme Freight", link: "TopCustomer", customerId: "cus_acme 01" },
  { text: " holds 42% of everything past due.", link: null, customerId: null },
];

const ACTIONS: Action[] = [
  {
    link: "BrokenPromise",
    count: 1,
    amountMinor: 400_000,
    customerId: "cus_acme 01",
    customerName: "Acme Freight",
  },
  { link: "CrossingNinety", count: 2, amountMinor: 420_000, customerId: null, customerName: null },
  { link: "Uncontacted", count: 2, amountMinor: 150_000, customerId: null, customerName: null },
  { link: "FollowUpsDue", count: 3, amountMinor: 0, customerId: null, customerName: null },
  { link: "PromisesDue", count: 1, amountMinor: 200_000, customerId: null, customerName: null },
  { link: "DueSoon", count: 1, amountMinor: 250_000, customerId: null, customerName: null },
  {
    link: "ShortPayRemainders",
    count: 1,
    amountMinor: 3_550,
    customerId: null,
    customerName: null,
  },
  { link: "UnappliedCash", count: 1, amountMinor: 250_000, customerId: null, customerName: null },
];

function briefing(over: { segments?: Segment[]; actions?: Action[]; narrated?: boolean } = {}) {
  return {
    segments: over.segments ?? SEGMENTS,
    actions: over.actions ?? ACTIONS,
    narrated: over.narrated ?? false,
    generatedAt: 1_791_000_000,
    generation: 1,
  };
}

function Location() {
  const location = useLocation();
  return <output aria-label="location">{`${location.pathname}${location.search}`}</output>;
}

function renderBriefing() {
  return renderAccountingPage(
    <Routes>
      <Route
        path="*"
        element={
          <>
            <AccountingBriefing />
            <Location />
          </>
        }
      />
    </Routes>,
    ["/accounting"],
  );
}

function location() {
  return screen.getByLabelText("location").textContent;
}

beforeEach(() => {
  fetchAccountingBriefing.mockResolvedValue(briefing());
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  authState.user = { name: "Dana Whitfield" };
});

describe("AccountingBriefing", () => {
  it("greets the reader by first name ahead of the day's findings", async () => {
    renderBriefing();

    const sentence = (await screen.findByText(/Hello Dana\./)).closest("p");
    expect(sentence?.textContent).toBe(
      "Hello Dana. Acme Freight missed a promise to pay $4,000, 2 invoices worth $4,200 pass 90 " +
        "days past due this week and Acme Freight holds 42% of everything past due.",
    );
    expect(screen.getByText(/^Brief as of /)).toBeTruthy();
    expect(screen.queryByText(/Written by Nova/)).toBeNull();
  });

  it("opens the page each linked phrase names", async () => {
    renderBriefing();

    fireEvent.click(
      await screen.findByRole("button", { name: "Open the missed promise in collections" }),
    );
    expect(location()).toBe("/accounting/ar/collections?customerId=cus_acme%2001");

    fireEvent.click(screen.getByRole("button", { name: "Open the aging report" }));
    expect(location()).toBe("/accounting/ar/aging");

    fireEvent.click(screen.getByRole("button", { name: "Open this customer's ledger" }));
    expect(location()).toBe("/accounting/ar/customer-ledger?customerId=cus_acme%2001");
  });

  it("lists the next steps, most pressing first, each opening where it gets done", async () => {
    renderBriefing();

    const steps = within(await screen.findByRole("list", { name: "Next steps" }));
    const links = steps.getAllByRole("link");
    expect(links.map((link) => [link.textContent, link.getAttribute("href")])).toEqual([
      [
        "Call Acme Freight about the missed $4,000 promise",
        "/accounting/ar/collections?customerId=cus_acme%2001",
      ],
      ["Call about 2 invoices before 90 days · $4,200", "/accounting/ar/aging"],
      [
        "Contact 2 customers not reached in 14 days · $1,500",
        "/accounting/ar/collections?due=true",
      ],
      ["Work 3 follow-ups that are due", "/accounting/ar/collections?due=true"],
      ["Confirm 1 promise due in 2 days · $2,000", "/accounting/ar/collections"],
      ["Remind on 1 invoice due this week · $2,500", "/accounting/ar/open-items"],
      ["Clear 1 short-pay balance · $35.50", "/accounting/ar/open-items"],
      ["Apply $2,500 of unapplied cash", "/accounting/ar/payments"],
    ]);
  });

  it("chases several missed promises from the due list when no one customer is named", async () => {
    fetchAccountingBriefing.mockResolvedValue(
      briefing({
        actions: [
          {
            link: "BrokenPromise",
            count: 3,
            amountMinor: 900_000,
            customerId: null,
            customerName: null,
          },
        ],
      }),
    );
    renderBriefing();

    const step = await screen.findByRole("link", { name: "Chase 3 missed promises · $9,000" });
    expect(step.getAttribute("href")).toBe("/accounting/ar/collections?due=true");
  });

  it("names what a new book needs before anything is late", async () => {
    fetchAccountingBriefing.mockResolvedValue(
      briefing({
        actions: [
          { link: "Drafts", count: 7, amountMinor: 912_300, customerId: null, customerName: null },
          {
            link: "NoCollections",
            count: 0,
            amountMinor: 4_606_216,
            customerId: null,
            customerName: null,
          },
          {
            link: "DueThisMonth",
            count: 35,
            amountMinor: 4_606_216,
            customerId: null,
            customerName: null,
          },
        ],
      }),
    );
    renderBriefing();

    const steps = within(await screen.findByRole("list", { name: "Next steps" }));
    expect(
      steps.getAllByRole("link").map((link) => [link.textContent, link.getAttribute("href")]),
    ).toEqual([
      ["Post 7 draft invoices · $9,123", "/billing/invoices?status=Draft"],
      [
        "Check payments are being recorded · $46,062.16 billed, none collected",
        "/accounting/ar/payments",
      ],
      [
        "Send statements ahead of 35 invoices due this month · $46,062.16",
        "/accounting/ar/open-items",
      ],
    ]);
  });

  it("draws no row of next steps when there is nothing to do", async () => {
    fetchAccountingBriefing.mockResolvedValue(
      briefing({
        segments: [
          {
            text: "Nothing is past due, so nobody needs a call today.",
            link: null,
            customerId: null,
          },
        ],
        actions: [],
      }),
    );
    renderBriefing();

    await waitFor(() =>
      expect(screen.getByRole("region", { name: "Accounting brief" })).toHaveTextContent(
        /nobody needs a call today/,
      ),
    );
    expect(screen.queryByRole("list", { name: "Next steps" })).toBeNull();
  });

  it("leaves a customer phrase as text when it carries no customer id", async () => {
    fetchAccountingBriefing.mockResolvedValue(
      briefing({
        segments: [
          { text: "Acme Freight", link: "TopCustomer", customerId: null },
          { text: " holds 42% of everything past due.", link: null, customerId: null },
        ],
      }),
    );
    renderBriefing();

    await waitFor(() =>
      expect(screen.getByRole("region", { name: "Accounting brief" })).toHaveTextContent(
        /Acme Freight holds 42%/,
      ),
    );
    expect(screen.queryByRole("button", { name: "Open this customer's ledger" })).toBeNull();
  });

  it("says when a model wrote the wording", async () => {
    fetchAccountingBriefing.mockResolvedValue(briefing({ narrated: true }));
    renderBriefing();

    expect(await screen.findByText(/^Written by Nova · as of /)).toBeTruthy();
    expect(screen.queryByText(/^Brief as of /)).toBeNull();
  });

  it("greets without a name when the session does not know one", async () => {
    authState.user = null;
    renderBriefing();

    expect((await screen.findByText(/Hello\./)).closest("p")?.textContent).toMatch(
      /^Hello\. Acme Freight missed a promise/,
    );
  });

  it("draws nothing when the brief cannot be read, leaving the dashboard below it", async () => {
    fetchAccountingBriefing.mockRejectedValue(new Error("unavailable"));
    renderBriefing();

    await vi.waitFor(() => expect(screen.queryByLabelText("Loading the brief")).toBeNull());
    expect(screen.queryByRole("region", { name: "Accounting brief" })).toBeNull();
    expect(screen.queryByText(/Hello/)).toBeNull();
  });
});
