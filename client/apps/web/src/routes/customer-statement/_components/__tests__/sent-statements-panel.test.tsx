import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SentStatementsPanel } from "../sent-statements-panel";

const { listSent } = vi.hoisted(() => ({ listSent: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { customerStatementService: { listSent } },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderPanel() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <SentStatementsPanel customerId="cus_1" />
    </QueryClientProvider>,
  );
}

describe("SentStatementsPanel", () => {
  it("lists each statement with what it said was owed and who sent it", async () => {
    listSent.mockResolvedValue([
      {
        id: "cst_2",
        periodStart: 0,
        periodEnd: 1788220799,
        endingBalanceMinor: 410236,
        pastDueMinor: 233874,
        currencyCode: "USD",
        sentTo: "ap@halstead.test",
        monthly: false,
        sentByName: "Dana Ortiz",
        sentAt: 1788300000,
      },
      {
        id: "cst_1",
        periodStart: 1785542400,
        periodEnd: 1788220799,
        endingBalanceMinor: 0,
        pastDueMinor: 0,
        currencyCode: "USD",
        sentTo: "ap@halstead.test",
        monthly: true,
        sentByName: "",
        sentAt: 1788290000,
      },
    ]);

    renderPanel();

    expect(await screen.findByText("Sent by hand")).toBeInTheDocument();
    expect(screen.getByText("Monthly")).toBeInTheDocument();
    expect(screen.getByText("$4,102.36 due")).toBeInTheDocument();
    expect(screen.getByText("$2,338.74 past due")).toBeInTheDocument();
    expect(screen.getByText("To ap@halstead.test, by Dana Ortiz")).toBeInTheDocument();
    expect(screen.getByText("To ap@halstead.test")).toBeInTheDocument();
  });

  it("says when nothing has been sent yet", async () => {
    listSent.mockResolvedValue([]);

    renderPanel();

    expect(
      await screen.findByText(/No statement has been emailed to this customer yet/),
    ).toBeInTheDocument();
  });
});
