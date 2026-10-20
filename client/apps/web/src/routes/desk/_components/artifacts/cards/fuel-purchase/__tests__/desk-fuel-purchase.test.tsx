import { recordPath } from "@/config/record-links";
import type { DeskFuelPurchaseCard } from "@/lib/graphql/fuel-desk-cards";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskFuelPurchase } from "../desk-fuel-purchase";
import { importedPurchase, manualPurchase } from "./fixtures";

const fetchers = vi.hoisted(() => ({ purchase: vi.fn() }));

vi.mock("@/lib/graphql/fuel-desk-cards", () => ({
  fetchDeskFuelPurchase: fetchers.purchase,
  fetchDeskFuelCard: vi.fn(),
}));
vi.mock("../../fuel-card/desk-fuel-card", () => ({
  DeskFuelCard: ({ recordId, backLabel }: { recordId: string; backLabel?: string }) => (
    <p>
      Fuel card {recordId} · {backLabel}
    </p>
  ),
  DeskFuelCardArtifact: () => null,
}));

let current: DeskFuelPurchaseCard | Error;

function grant({ cards = true } = {}) {
  seedPermissions({
    [Resource.FuelPurchase]: Operation.Read,
    ...(cards ? { [Resource.FuelCard]: Operation.Read } : {}),
  });
}

function draw(card: DeskFuelPurchaseCard | Error, recordId = "fpur_1019") {
  current = card;
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskFuelPurchase recordId={recordId} onBack={vi.fn()} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  grant();
  fetchers.purchase.mockReset().mockImplementation(async () => {
    if (current instanceof Error) throw current;
    return structuredClone(current);
  });
});

afterEach(() => {
  clearPermissions();
  vi.restoreAllMocks();
});

describe("DeskFuelPurchase", () => {
  it("holds the card's outline while it is read, then draws the receipt beside what it does to IFTA", async () => {
    const { container } = draw(importedPurchase());

    expect(screen.getByRole("status")).toHaveTextContent("Loading the fuel purchase…");
    expect(container.querySelector(".dk-rec-skel.dk-fpu .dk-fpu-hero")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Pilot Travel Center", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-stub")).toHaveTextContent("DSL");
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$461.72118.42 gal at $3.8990",
    );
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Tax paid in TX");

    const receipt = screen.getByRole("figure", { name: "Receipt as recorded" });
    expect(Array.from(receipt.querySelectorAll("dl > div"), (line) => line.textContent)).toEqual([
      "ProductDiesel",
      "Quantity118.42 gal",
      "Price / gal$3.8990",
      "Odometer181,604 mi",
      "CardComdata •••• 4412",
      "RefTX-20261019-004812",
    ]);
    expect(receipt.querySelector(".dk-fpu-total")).toHaveTextContent("Total$461.72");

    const ifta = container.querySelector<HTMLElement>(".dk-fpu-ifta");
    if (!ifta) throw new Error("No IFTA panel");
    expect(ifta.querySelector(".dk-fpu-juris")).toHaveTextContent("TXTexasIFTA, Q4 2026Counts");
    expect(Array.from(ifta.querySelectorAll("dl > div"), (line) => line.textContent)).toEqual([
      "Gallons118.420",
      "Tax-paid gallons118.420",
      "Tax already paid$23.68at $0.2000/gal",
      "T-101 in Texas this quarter612.840 gal · 5 purchases",
    ]);
  });

  it("checks the purchase and shows how the feed filed it", async () => {
    draw(importedPurchase());
    await screen.findByRole("heading", { name: "Pilot Travel Center", level: 2 });

    const checks = section("Checks");
    expect(checks.querySelector(".dk-rec-sh")).toHaveTextContent("ChecksNothing to fix");
    expect(Array.from(checks.querySelectorAll(".dk-rec-ck"), (check) => check.className)).toEqual([
      "dk-rec-ck dk-ok",
      "dk-rec-ck dk-ok",
      "dk-rec-ck dk-ok",
    ]);

    const filed = within(section("How it was filed")).getByRole("list", {
      name: "How the purchase was filed",
    });
    expect(Array.from(filed.querySelectorAll("li b"), (step) => step.textContent)).toEqual([
      "Read from the Comdata transaction file",
      "Card found by its last four",
      "Tractor matched by unit code",
    ]);
  });

  it("opens the card it was bought on in the card's place", async () => {
    draw(importedPurchase());
    await screen.findByRole("heading", { name: "Pilot Travel Center", level: 2 });

    fireEvent.click(within(section("The card")).getByRole("button"));
    expect(
      await screen.findByText("Fuel card fcard_4412 · Back to Pilot Travel Center"),
    ).toBeInTheDocument();
  });

  it("links the card to its page for someone who may not read fuel cards here", async () => {
    grant({ cards: false });
    draw(importedPurchase());
    await screen.findByRole("heading", { name: "Pilot Travel Center", level: 2 });

    expect(within(section("The card")).getByRole("link")).toHaveAttribute(
      "href",
      recordPath("fuel_card", "fcard_4412"),
    );
  });

  it("flags what to check on a hand-entered purchase and sends the reader to its page to fix it", async () => {
    const { container } = draw(manualPurchase(), "fpur_yard");
    await screen.findByRole("heading", { name: "Yard bulk tank", level: 2 });

    expect(container.querySelector(".dk-rec-amt b")).toHaveClass("dk-warn");
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("2 to check");
    expect(section("Checks").querySelector(".dk-rec-sh")).toHaveTextContent("Checks2 to check");
    expect(container.querySelector(".dk-fpu-diff")).toHaveTextContent("Qty × price$558.24");
    expect(container.querySelector(".dk-fpu-miss")).toHaveTextContent("OdometerNot recorded");
    expect(section("How it was filed")).toHaveTextContent("Entered by Dana Park");
    expect(screen.queryByRole("heading", { name: "The card", level: 3 })).toBeNull();

    const bar = container.querySelector<HTMLElement>(".dk-rec-bar");
    if (!bar) throw new Error("No bar");
    expect(bar).toHaveTextContent("Correct what's flagged on the purchase's page");
    expect(within(bar).getByRole("link", { name: "Open purchase" })).toHaveAttribute(
      "href",
      recordPath("fuel_purchase", "fpur_yard"),
    );
  });

  it("says a purchase it cannot read could not be read", async () => {
    draw(new Error("Fuel purchase not found"));

    expect(await screen.findByText("This fuel purchase could not be read")).toBeInTheDocument();
  });
});
