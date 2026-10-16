import type { DeskEdiPartnerCard, EdiPartnerMessagesPage } from "@/lib/graphql/edi-partner-card";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskEdiPartner, DeskEdiPartnerArtifact } from "../desk-edi-partner";
import acmeFixture from "./acme-trouble.json";
import krogerFixture from "./kroger-setup.json";
import lonestarFixture from "./lonestar-internal.json";
import tysonMessages from "./tyson-messages.json";
import tysonFixture from "./tyson-trading.json";

const fetchers = vi.hoisted(() => ({
  partner: vi.fn(),
  messages: vi.fn(),
  test: vi.fn(),
  poll: vi.fn(),
}));

vi.mock("@/lib/graphql/edi-partner-card", () => ({
  fetchDeskEdiPartner: fetchers.partner,
  fetchEdiPartnerMessages: fetchers.messages,
}));
vi.mock("@/services/api", () => ({
  apiService: {
    ediService: { testProfileConnection: fetchers.test, pollProfile: fetchers.poll },
  },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));
// The EDI message card is its own chunk; here it stands in as what it was given,
// with its way back.
vi.mock("../../edi-message/desk-edi-message", () => ({
  DeskEdiMessage: ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>EDI message card {recordId}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskEdiMessageArtifact: () => null,
}));

const tyson = tysonFixture.card as unknown as DeskEdiPartnerCard;
const acme = acmeFixture.card as unknown as DeskEdiPartnerCard;
const kroger = krogerFixture.card as unknown as DeskEdiPartnerCard;
const lonestar = lonestarFixture.card as unknown as DeskEdiPartnerCard;

function page(rows: unknown[], total: number | null): EdiPartnerMessagesPage {
  return { rows, total, endCursor: null, hasNextPage: false } as EdiPartnerMessagesPage;
}

function draw(props: Partial<Parameters<typeof DeskEdiPartner>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const tree = (more: Partial<typeof props> = {}) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskEdiPartner
          recordId="edip_tyson"
          order={["edip_acme", "edip_tyson"]}
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
    queryClient,
    step: (recordId: string) => view.rerender(tree({ recordId })),
  };
}

function section(name: string): HTMLElement {
  const heading = screen.getByRole("heading", { name, level: 3 });
  return heading.closest("section") as HTMLElement;
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(tysonFixture.now * 1000);
  seedPermissions({ [Resource.EDI]: Operation.Read });
  fetchers.partner.mockReset().mockImplementation(async (id: string) => {
    const cards: Record<string, DeskEdiPartnerCard> = {
      edip_tyson: tyson,
      edip_acme: acme,
      edip_kroger: kroger,
      edip_lonestar_link: lonestar,
    };
    if (!cards[id]) throw new Error("EDIPartner not found");
    return structuredClone(cards[id]);
  });
  fetchers.messages
    .mockReset()
    .mockImplementation(async (id: string) =>
      id === "edip_tyson" ? page(tysonMessages.messages, tysonMessages.total) : page([], 0),
    );
  fetchers.test.mockReset();
  fetchers.poll.mockReset();
});

afterEach(() => {
  clearPermissions();
  vi.useRealTimers();
});

describe("DeskEdiPartner", () => {
  it("draws the next partner's outline, not the one it left, while the next one is read", async () => {
    const { container, step } = draw();
    await screen.findByRole("heading", { name: "Tyson Foods", level: 2 });

    fetchers.partner.mockReturnValue(new Promise(() => {}));
    step("edip_acme");

    expect(screen.queryByRole("heading", { name: "Tyson Foods", level: 2 })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("reads a healthy partner: trading, its share delivered, what each side sends and how it is addressed", async () => {
    draw();

    expect(
      await screen.findByRole("heading", { name: "Tyson Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Trading")).toBeInTheDocument();
    expect(screen.getByText("99.8%")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Needs a look" })).toBeNull();

    const sends = within(section("What we trade")).getByRole("list", { name: "We send" });
    expect(
      within(sends)
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual([
      "210Freight invoiceWaits on a 997 within 1d96",
      "214Shipment status3 failed to send1,284",
      "990Response to a tenderNo acknowledgment asked42",
    ]);
    expect(within(section("How it's addressed")).getByText("Production")).toBeInTheDocument();
    expect(within(section("Ready to trade")).getByText("All 6 checks pass")).toBeInTheDocument();
    expect(within(section("Their codes we know")).getByText("41 locations")).toBeInTheDocument();
  });

  it("names a line's credentials as on file and never shows a secret", async () => {
    draw();

    const reach = await screen.findByRole("heading", { name: "How we reach them" });
    const patch = reach.closest("section") as HTMLElement;
    expect(within(patch).getByRole("list", { name: "Credentials" }).textContent).toBe(
      "Our private keyOn fileBasic auth passwordOn file",
    );
    expect(patch.textContent).not.toContain("BEGIN CERTIFICATE");
    expect(within(patch).getByText("Their encryption certificate")).toBeInTheDocument();
  });

  it("lists the partner's latest messages, with how many there are in all", async () => {
    draw();

    const latest = await screen.findByText("Interchange 000008812");
    expect(fetchers.messages).toHaveBeenCalledWith(
      "edip_tyson",
      { first: 5, after: null },
      expect.anything(),
    );
    expect(within(section("Latest messages")).getByText("12,480 in all")).toBeInTheDocument();
    expect(latest.closest("a,button")?.textContent).toContain("Received");
  });

  it("puts what needs a look first, worst first, and opens a message in the card's place", async () => {
    draw({ recordId: "edip_acme" });

    const needs = (await screen.findByRole("heading", { name: "Needs a look" })).closest(
      "section",
    ) as HTMLElement;
    expect(within(needs).getByText("4 things")).toBeInTheDocument();
    expect(
      Array.from(needs.querySelectorAll(".dk-edp-need-h .dk-what > b")).map(
        (title) => title.textContent,
      ),
    ).toEqual([
      "We can't pick up their files",
      "A message was dead-lettered",
      "3 messages are waiting on an acknowledgment past 4h",
      "A file was set aside",
    ]);
    expect(within(needs).getByText(/no supported methods remain/)).toBeInTheDocument();

    fireEvent.click(within(needs).getByText(/550 Requested action not taken/));
    expect(await screen.findByText("EDI message card edimsg_acme_210_dl")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back to Acme Foods" }));
    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
  });

  it("offers testing the line and picking up only to someone who may change EDI settings", async () => {
    draw({ recordId: "edip_acme" });

    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });
    expect(screen.queryByRole("button", { name: /Test connection/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /Pick up now/ })).toBeNull();
  });

  it("tests the line and shows each check in place", async () => {
    seedPermissions({ [Resource.EDI]: Operation.Read | Operation.Update });
    fetchers.test.mockResolvedValue({
      success: false,
      checks: [
        { name: "Host reachable", status: "passed", message: "sftp.acmefoods.example:22 answered" },
        { name: "Sign in", status: "failed", message: "ssh: unable to authenticate" },
      ],
    });
    draw({ recordId: "edip_acme" });

    fireEvent.click(await screen.findByRole("button", { name: /Test connection/ }));

    const result = (await screen.findByRole("heading", { name: "Connection test" })).closest(
      "section",
    ) as HTMLElement;
    expect(fetchers.test).toHaveBeenCalledWith("edicp_acme_sftp");
    expect(within(result).getByText("Failed")).toBeInTheDocument();
    expect(within(result).getByText("ssh: unable to authenticate")).toBeInTheDocument();
  });

  it("picks up the partner's files by hand and reads the partner again", async () => {
    seedPermissions({ [Resource.EDI]: Operation.Read | Operation.Update });
    fetchers.poll.mockResolvedValue({
      profileId: "edicp_acme_sftp",
      stagedFileIds: ["ediinf_new"],
      skippedFiles: 0,
    });
    draw({ recordId: "edip_acme" });

    fireEvent.click(await screen.findByRole("button", { name: /Pick up now/ }));

    await vi.waitFor(() => expect(fetchers.poll).toHaveBeenCalledWith("edicp_acme_sftp"));
    await vi.waitFor(() => expect(fetchers.partner).toHaveBeenCalledTimes(2));
  });

  it("leads with what is missing for a partner still being set up, and cannot test a line it lacks", async () => {
    seedPermissions({ [Resource.EDI]: Operation.Read | Operation.Update });
    draw({ recordId: "edip_kroger" });

    await screen.findByRole("heading", { name: "Kroger", level: 2 });
    const headings = screen
      .getAllByRole("heading", { level: 3 })
      .map((heading) => heading.textContent);
    expect(headings.slice(0, 2)).toEqual(["Ready to trade", "What we'll trade"]);
    expect(within(section("Ready to trade")).getByText("3 of 6")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Test connection/ })).toBeDisabled();
    expect(screen.queryByRole("button", { name: /Pick up now/ })).toBeNull();
    expect(within(section("How it's addressed")).getByText("Test")).toBeInTheDocument();
  });

  it("reads a Trenova-to-Trenova partner by its connection, with no line to test", async () => {
    seedPermissions({ [Resource.EDI]: Operation.Read | Operation.Update });
    draw({ recordId: "edip_lonestar_link" });

    await screen.findByRole("heading", { name: "Lone Star Logistics", level: 2 });
    expect(screen.getByText("Connected")).toBeInTheDocument();
    expect(within(section("The connection")).getByText("Accepted")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "How we reach them" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Latest messages" })).toBeNull();
    expect(screen.queryByRole("button", { name: /Test connection/ })).toBeNull();
    expect(fetchers.messages).toHaveBeenCalled();
  });

  it("links the customer to its page for someone who may not read customers", async () => {
    draw();

    const forWhom = section(
      (await screen.findByRole("heading", { name: "Who it's for" })).textContent ?? "",
    );
    expect(within(forWhom).getByRole("link", { name: /Tyson Foods/ })).toHaveAttribute(
      "href",
      "/billing/configuration-files/customers?panelType=edit&panelEntityId=cus_tyson",
    );
  });

  it("says a partner it cannot read could not be read", async () => {
    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <MemoryRouter>
          <DeskEdiPartnerArtifact recordId="edip_globex" artifact={{} as never} fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "This EDI partner could not be read", level: 2 }),
    ).toBeInTheDocument();
  });
});
