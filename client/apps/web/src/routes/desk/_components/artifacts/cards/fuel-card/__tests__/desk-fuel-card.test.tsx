import { recordPath } from "@/config/record-links";
import type { DeskFuelCardCard } from "@/lib/graphql/fuel-desk-cards";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskFuelCard } from "../desk-fuel-card";
import { activeCard, cancelledCard, discoveredCard } from "./fixtures";

const fetchers = vi.hoisted(() => ({
  card: vi.fn(),
  resolve: vi.fn(),
}));

vi.mock("@/lib/graphql/fuel-desk-cards", () => ({
  fetchDeskFuelCard: fetchers.card,
  fetchDeskFuelPurchase: vi.fn(),
}));
vi.mock("@/lib/graphql/fuel-purchase-import", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/fuel-purchase-import")>()),
  resolveFuelPurchaseImportRows: fetchers.resolve,
}));
vi.mock("@/routes/fuel-card/_components/assign-fuel-card-dialog", () => ({
  AssignFuelCardDialog: ({
    open,
    card,
    onOpenChange,
    onAssigned,
  }: {
    open: boolean;
    card: { id: string; version: number } | null;
    onOpenChange: (open: boolean) => void;
    onAssigned: () => Promise<void> | void;
  }) =>
    open ? (
      <div role="dialog" aria-label="Assign card">
        <span>
          Assigning {card?.id} at version {card?.version}
        </span>
        <button
          type="button"
          onClick={() => {
            onOpenChange(false);
            void onAssigned();
          }}
        >
          Assign
        </button>
      </div>
    ) : null,
}));
vi.mock("../../fuel-purchase/desk-fuel-purchase", () => ({
  DeskFuelPurchase: ({ recordId, backLabel }: { recordId: string; backLabel?: string }) => (
    <p>
      Purchase card {recordId} · {backLabel}
    </p>
  ),
  DeskFuelPurchaseArtifact: () => null,
}));

let current: DeskFuelCardCard;
/** Cards other than the one drawn that the card can open in its place, by id. */
let others: Record<string, DeskFuelCardCard> = {};

/** Signs the test's person in to read cards and purchases and, when asked, to assign cards and work imports. */
function grant({ assign = false, imports = false } = {}) {
  seedPermissions({
    [Resource.FuelCard]: Operation.Read | (assign ? Operation.Update : 0),
    [Resource.FuelPurchase]: Operation.Read,
    [Resource.FuelPurchaseImport]: Operation.Read | (imports ? Operation.Import : 0),
  });
}

function draw(card: DeskFuelCardCard) {
  current = card;
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskFuelCard recordId={card.card.id} onBack={vi.fn()} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return view;
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

function bar(container: HTMLElement): HTMLElement {
  const found = container.querySelector<HTMLElement>(".dk-rec-bar");
  if (!found) throw new Error("No bar");
  return found;
}

beforeEach(() => {
  grant();
  others = {};
  fetchers.card.mockReset().mockImplementation(async (id: string) => {
    const found = id === current.card.id ? current : others[id];
    if (!found) throw new Error("Fuel card not found");
    return structuredClone(found);
  });
  fetchers.resolve.mockReset().mockResolvedValue({ reviewed: 6, resolved: 6, committed: 6 });
});

afterEach(() => {
  clearPermissions();
  vi.restoreAllMocks();
});

describe("DeskFuelCard", () => {
  it("holds the card's outline while it is read, then draws the card, who holds it and its month", async () => {
    const { container } = draw(activeCard());

    expect(screen.getByRole("status")).toHaveTextContent("Loading the fuel card…");
    expect(container.querySelector(".dk-rec-skel .dk-fcd-face")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Comdata •••• 4412 — Unit T-101", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$4,092.18spent in October · 1,046.3 gal",
    );
    expect(screen.getByRole("img", { name: "Comdata card ending 4412" })).toHaveTextContent(
      "Fleet fuel••••••••••••4412CardComdata •••• 4412 — Unit T-101Expires08/29Rosa Diaz",
    );
    const holders = container.querySelector<HTMLElement>(".dk-fcd-holds");
    if (!holders) throw new Error("No holders");
    expect(
      within(holders)
        .getAllByRole("link")
        .map((link) => [link.textContent, link.getAttribute("href")]),
    ).toEqual([
      ["RDRosa DiazDriver", recordPath("worker", "wrk_rosa")],
      ["T-101T-101Freightliner Cascadia · Tractor", recordPath("tractor", "tr_101")],
    ]);
    expect(container.querySelector(".dk-rec-stats")).toHaveTextContent(
      "Purchases9Gallons1,046.3Avg / gal$3.91Untaxed15.0 gal",
    );
  });

  it("lists the latest purchases with what each is to IFTA, and opens one in the card's place", async () => {
    draw(activeCard());
    await screen.findByRole("heading", { name: "Comdata •••• 4412 — Unit T-101", level: 2 });

    const recent = section("Recent purchases");
    expect(recent).toHaveTextContent("23 in all");
    expect(
      Array.from(
        recent.querySelectorAll(".dk-fcd-buy [data-slot='badge']"),
        (badge) => badge.textContent,
      ),
    ).toEqual(["Tax paid", "Spend", "Untaxed"]);
    expect(recent).toHaveTextContent(
      "3 of 23 shown. Every purchase on the card is on the purchases page.",
    );

    fireEvent.click(within(recent).getAllByRole("button")[1] as HTMLElement);
    expect(
      await screen.findByText("Purchase card fpur_1019b · Back to Comdata •••• 4412 — Unit T-101"),
    ).toBeInTheDocument();
  });

  it("draws the quarter's fuel by jurisdiction against the largest, and the feed's last read", async () => {
    draw(activeCard());
    await screen.findByRole("heading", { name: "Comdata •••• 4412 — Unit T-101", level: 2 });

    const bars = section("Where it fuels");
    expect(bars).toHaveTextContent("Q4 2026 · for IFTA");
    expect(
      Array.from(bars.querySelectorAll<HTMLElement>(".dk-fcd-track i"), (bar) => bar.style.width),
    ).toEqual(["100%", "50%"]);
    expect(section("The feed")).toHaveTextContent("Comdata transaction file");
  });

  it("offers no assigning to someone who may not change cards", async () => {
    const { container } = draw(discoveredCard());
    await screen.findByRole("heading", { name: "WEX •••• 7720", level: 2 });

    expect(within(bar(container)).queryByRole("button", { name: "Assign card" })).toBeNull();
    expect(bar(container)).toHaveTextContent("Assign, suspend or cancel it on the fuel cards page");
  });

  it("assigns a card the feed found and then posts what was waiting in each of its imports", async () => {
    grant({ assign: true, imports: true });
    const { container } = draw(discoveredCard());
    await screen.findByRole("heading", { name: "WEX •••• 7720", level: 2 });

    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$1,612.45waiting to post · 418.6 gal",
    );
    expect(container.querySelector(".dk-fcd-wait")).toHaveTextContent(
      "Assigning it activates the cardThe 12 purchases waiting on it are posted once it's assigned.",
    );
    const waiting = section("Waiting on this card");
    expect(waiting).toHaveTextContent("12 purchases · $1,612.45");
    expect(waiting).toHaveTextContent("2 of 12 shown. The rest are on the fuel imports page.");

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Assign card" }));
    const dialog = screen.getByRole("dialog", { name: "Assign card" });
    expect(dialog).toHaveTextContent("Assigning fcard_7720 at version 1");

    current = {
      ...activeCard(),
      card: { ...activeCard().card, id: "fcard_7720", provider: "WEX", lastFour: "7720" },
    };
    fireEvent.click(within(dialog).getByRole("button", { name: "Assign" }));

    await waitFor(() => expect(fetchers.resolve).toHaveBeenCalledTimes(2));
    expect(fetchers.resolve.mock.calls).toEqual([
      ["fpib_feed", 4],
      ["fpib_upload", 2],
    ]);
    expect(
      await screen.findByRole("heading", { name: "Comdata •••• 4412 — Unit T-101", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Waiting on this card", level: 3 })).toBeNull();
    expect(screen.queryByRole("button", { name: "Assign card" })).toBeNull();
  });

  it("assigns without working the imports for someone who may not, and says who posts them", async () => {
    grant({ assign: true });
    const { container } = draw(discoveredCard());
    await screen.findByRole("heading", { name: "WEX •••• 7720", level: 2 });

    expect(container.querySelector(".dk-fcd-wait")).toHaveTextContent(
      "Someone who can import fuel statements then posts what's waiting on it.",
    );
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Assign card" }));
    fireEvent.click(screen.getByRole("button", { name: "Assign" }));

    await waitFor(() => expect(fetchers.card).toHaveBeenCalledTimes(2));
    expect(fetchers.resolve).not.toHaveBeenCalled();
  });

  it("says which import refused to post its rows, and still works the others", async () => {
    grant({ assign: true, imports: true });
    fetchers.resolve
      .mockReset()
      .mockRejectedValueOnce(new Error("The import was changed by someone else"))
      .mockResolvedValueOnce({ reviewed: 1, resolved: 1, committed: 0 });
    const { container } = draw(discoveredCard());
    await screen.findByRole("heading", { name: "WEX •••• 7720", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Assign card" }));
    fireEvent.click(screen.getByRole("button", { name: "Assign" }));

    expect(
      await screen.findByText(
        "The card was assigned, but an import holding its rows refused: The import was changed by someone else",
      ),
    ).toBeInTheDocument();
    expect(fetchers.resolve).toHaveBeenCalledTimes(2);
  });

  it("says why a cancelled card was cancelled and opens its replacement in place", async () => {
    const replacement = activeCard();
    others = {
      fcard_5306: {
        ...replacement,
        card: { ...replacement.card, id: "fcard_5306", label: "Comdata •••• 5306 — Unit T-087" },
      },
    };
    const { container } = draw(cancelledCard());
    await screen.findByRole("heading", { name: "Comdata •••• 2290 — Unit T-087", level: 2 });

    expect(container.querySelector(".dk-fcd-face")).toHaveClass("dk-cancelled");
    expect(container.querySelector(".dk-fcd-void")).toHaveTextContent(
      "Why it was cancelledReported lost by the driver at Love's, Oklahoma City.",
    );
    expect(container.querySelector(".dk-rec-amt")).toBeNull();
    expect(section("Last purchases")).toHaveTextContent("Before it was cancelled");
    expect(bar(container)).toHaveTextContent(
      "A cancelled card can't be reopened; add its replacement on the fuel cards page",
    );

    fireEvent.click(screen.getByRole("button", { name: /Comdata •••• 5306/u }));
    expect(
      await screen.findByRole("heading", { name: "Comdata •••• 5306 — Unit T-087", level: 2 }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Back to Comdata •••• 2290 — Unit T-087" }),
    ).toBeInTheDocument();
  });
});
