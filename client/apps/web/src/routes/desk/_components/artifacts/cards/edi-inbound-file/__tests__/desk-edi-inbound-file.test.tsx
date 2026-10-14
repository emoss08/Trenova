import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskEdiInboundFile, DeskEdiInboundFileArtifact } from "../desk-edi-inbound-file";
import { wireFileInspection, wireInboundFile } from "./fixtures";

const server = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, get: server.get } };
});
// The message card is its own chunk; here it stands in as what it was given, with its way back.
vi.mock("../../edi-message/desk-edi-message", () => ({
  DeskEdiMessage: ({
    recordId,
    order,
    onBack,
    backLabel,
  }: {
    recordId: string;
    order: readonly string[];
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>Message card {recordId}</p>
      <p>Among {order.join(", ")}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskEdiMessageArtifact: () => null,
}));

type Routes = Record<string, () => unknown>;
let routes: Routes;

function route(overrides: Routes = {}) {
  routes = {
    "/edi/inbound-files/ediinf_sim/": () => wireInboundFile("simulator"),
    "/edi/inbound-files/ediinf_sim/inspect/": () => wireFileInspection("simulator"),
    "/edi/inbound-files/ediinf_acme/": () => wireInboundFile("batch"),
    "/edi/inbound-files/ediinf_acme/inspect/": () => wireFileInspection("batch"),
    "/edi/inbound-files/ediinf_bad/": () => wireInboundFile("quarantined"),
    "/edi/inbound-files/ediinf_bad/inspect/": () => wireFileInspection("quarantined"),
    "/edi/inbound-files/ediinf_dup/": () => wireInboundFile("duplicate"),
    "/edi/inbound-files/ediinf_dup/inspect/": () => wireFileInspection("simulator"),
    ...overrides,
  };
}

function draw(props: Partial<Parameters<typeof DeskEdiInboundFile>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskEdiInboundFile
          recordId="ediinf_sim"
          order={["ediinf_bad", "ediinf_sim", "ediinf_acme"]}
          {...handlers}
          {...props}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, ...handlers };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

const asked = (url: string) => server.get.mock.calls.some(([called]) => called === url);

beforeEach(() => {
  seedPermissions({ [Resource.EDI]: Operation.Read });
  route();
  server.get.mockReset().mockImplementation(async (url: string) => {
    const answer = routes[url];
    if (!answer) throw new Error(`Not found: ${url}`);
    return answer();
  });
});

afterEach(() => {
  clearPermissions();
  vi.restoreAllMocks();
});

describe("DeskEdiInboundFile", () => {
  it("holds the card's outline while the file is read, then draws the simulator's file", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the EDI inbound file…");
    expect(container.querySelector(".dk-rec-skel .dk-eif-intake")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-eif-label")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-eif-glyph")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "sim-204-1791716400000000000.edi", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(document.querySelector(".dk-rec-kick")).toHaveTextContent(
      /Inbound file.*SIMPARTNER.*AS2/u,
    );
    expect(screen.getByText("From EDI Partner Simulator")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Processed");
    expect(document.querySelector(".dk-eif-glyph")).toHaveTextContent(".edi");

    const stages = within(section("Intake")).getAllByRole("listitem");
    expect(stages.map((stage) => stage.textContent)).toEqual([
      expect.stringContaining("Over AS2"),
      expect.stringContaining("Interchange 000000042"),
      expect.stringContaining("1 transaction set handled"),
    ]);
    expect(within(section("Intake")).queryByRole("list", { name: "Processing notes" })).toBeNull();
  });

  it("addresses the file from its ISA envelope, naming the AS2 message it came as", async () => {
    draw();
    await screen.findByRole("heading", { level: 2 });

    const label = section("Routing label");
    expect(within(label).getByText("SIMPARTNER")).toBeInTheDocument();
    expect(within(label).getByText("TRENOVA")).toBeInTheDocument();
    expect(within(label).getAllByTitle("Mutually defined")).toHaveLength(2);
    expect(within(label).getByText("000000042")).toBeInTheDocument();
    expect(within(label).getByText("Simulator AS2")).toBeInTheDocument();
    expect(
      within(label).getByText("<sim-1791716400000000000@edi-partner-sim>"),
    ).toBeInTheDocument();
    expect(within(label).getByText("9f2c 1a7b 4e8d 3f60")).toBeInTheDocument();
  });

  it("lists what the file carried and checks its envelope once it is inspected", async () => {
    draw();
    await screen.findByRole("heading", { level: 2 });

    const carried = section("What it carried");
    expect(await within(carried).findByText("SM")).toBeInTheDocument();
    expect(within(carried).getByText("Load tenders")).toBeInTheDocument();
    expect(within(carried).getByText("GS 42")).toBeInTheDocument();
    expect(within(carried).getByText("1 set, as declared")).toBeInTheDocument();
    const row = within(carried).getByRole("button", { name: /Load tender.*0001/u });
    expect(row).toHaveTextContent("Simulator Shipper, Chicago, IL to Simulator Consignee");
    expect(row).toHaveTextContent("16 segments");
    expect(row).toHaveTextContent("Handled");
    expect(within(carried).getByRole("img", { name: "20 segments" })).toBeInTheDocument();

    const envelope = section("The interchange");
    expect(within(envelope).getByText("ISA 000000042 closed by its IEA")).toBeInTheDocument();
    expect(within(envelope).getByText("1 group, as declared")).toBeInTheDocument();
    expect(within(envelope).getByRole("button", { name: "Copy file" })).toBeInTheDocument();
  });

  it("pins each note and finding on the transaction set it names, and keeps the rest in the intake", async () => {
    draw({ recordId: "ediinf_acme" });
    await screen.findByRole("heading", { name: "ACME_20261013_0912.x12", level: 2 });
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Partly processed");

    const intake = section("Intake");
    expect(within(intake).getByText("3 notes to review")).toBeInTheDocument();
    const ledger = within(intake).getByRole("list", { name: "Processing notes" });
    expect(
      within(ledger)
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual([
      "Acknowledgment for control number 0001 could not be matched to an outbound message",
    ]);

    const carried = section("What it carried");
    await within(carried).findByText("Acknowledgments");
    expect(within(carried).getByText("GS 318")).toBeInTheDocument();
    const changed = within(carried).getByRole("list", { name: "Notes on 0002" });
    expect(
      within(changed)
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual([
      "Load tender change 204/0002 did not match a pending tender for LT-88241; review manually",
      "Load tender 204/0002 requires mapping before approval (1 unresolved entities)",
    ]);
    expect(within(carried).getByText("SE01 says 7; 9 counted")).toBeInTheDocument();
    expect(
      within(carried).getByText("SE01 reports 7 segments but 9 were found from ST through SE."),
    ).toBeInTheDocument();
    expect(within(carried).getByRole("button", { name: /Load tender.*0002/u })).toHaveTextContent(
      "Handled with notes",
    );
  });

  it("opens a transaction set as its message in the card's place, and comes back", async () => {
    draw({ recordId: "ediinf_acme" });
    await screen.findByRole("heading", { name: "ACME_20261013_0912.x12", level: 2 });

    fireEvent.click(
      await screen.findByRole("button", { name: /Functional acknowledgment.*0001/u }),
    );

    expect(await screen.findByText("Message card edimsg_acme_997")).toBeInTheDocument();
    expect(
      screen.getByText("Among edimsg_acme_204a, edimsg_acme_204b, edimsg_acme_997"),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to ACME_20261013_0912.x12" }));
    expect(
      await screen.findByRole("heading", { name: "ACME_20261013_0912.x12", level: 2 }),
    ).toBeInTheDocument();
  });

  it("shows how a file that is not X12 begins, and why it was held", async () => {
    draw({ recordId: "ediinf_bad" });
    await screen.findByRole("heading", { name: "tenders-week-42.txt", level: 2 });
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Quarantined");

    expect(
      within(section("Intake")).getByText(
        "Inbound file does not start with an ISA interchange header",
      ),
    ).toBeInTheDocument();
    expect(within(section("Routing label")).getAllByText("Not read")).toHaveLength(3);

    const envelope = section("The interchange");
    expect(
      await within(envelope).findByText("This file is not an X12 interchange. It begins:"),
    ).toBeInTheDocument();
    expect(within(envelope).getByText("Hello from the Acme EDI team,")).toBeInTheDocument();
    expect(within(envelope).getByText("Regards, Dana")).toBeInTheDocument();
    expect(
      within(envelope).getByRole("button", { name: "Show 6 more findings" }),
    ).toBeInTheDocument();
    expect(
      within(section("What it carried")).getByText(
        "Nothing in it could be read as a transaction set.",
      ),
    ).toBeInTheDocument();
  });

  it("reads a purged file from the messages kept from it, without asking for its interchange", async () => {
    route({
      "/edi/inbound-files/ediinf_acme/": () =>
        wireInboundFile("batch", { rawContent: "", rawPurgedAt: 1_791_700_000 }),
    });
    draw({ recordId: "ediinf_acme" });
    await screen.findByRole("heading", { name: "ACME_20261013_0912.x12", level: 2 });

    expect(screen.getByText("Raw file purged")).toBeInTheDocument();
    const carried = section("What it carried");
    expect(
      within(carried)
        .getAllByRole("button")
        .map((row) => row.textContent),
    ).toEqual([
      expect.stringContaining("0001"),
      expect.stringContaining("0002"),
      expect.stringContaining("0001"),
    ]);
    expect(within(carried).getByText("GS 318")).toBeInTheDocument();
    expect(within(carried).queryByRole("img")).toBeNull();
    expect(
      within(section("The interchange")).getByText(
        /The raw file was purged .* under the retention policy; the messages kept from it are listed above\./u,
      ),
    ).toBeInTheDocument();
    expect(asked("/edi/inbound-files/ediinf_acme/inspect/")).toBe(false);
  });

  it("keeps the kept messages and says so in place when the interchange cannot be inspected", async () => {
    route({
      "/edi/inbound-files/ediinf_acme/inspect/": () => {
        throw new Error("inspector down");
      },
    });
    draw({ recordId: "ediinf_acme" });
    await screen.findByRole("heading", { name: "ACME_20261013_0912.x12", level: 2 });

    expect(
      await within(section("The interchange")).findByText(/inspector down|could not be inspected/u),
    ).toBeInTheDocument();
    expect(within(section("What it carried")).getAllByRole("button")).toHaveLength(3);
  });

  it("says a duplicate was turned away and that nothing in it was processed again", async () => {
    draw({ recordId: "ediinf_dup" });
    await screen.findByRole("heading", { name: "sim-204-1791716400000000001.edi", level: 2 });

    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Duplicate interchange");
    expect(within(section("Intake")).getByText("Already processed")).toBeInTheDocument();
    expect(
      within(section("Intake")).getByText(
        "Interchange control number 000000042 was already processed for this partner",
      ),
    ).toBeInTheDocument();
    expect(
      await within(section("What it carried")).findByText("Not processed"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("The first copy of this interchange was processed already"),
    ).toBeInTheDocument();
  });

  it("says the file could not be read, and keeps the way back", async () => {
    route({
      "/edi/inbound-files/ediinf_sim/": () => {
        throw new Error("gone");
      },
    });
    const { onBack } = draw();

    expect(await screen.findByText("This EDI inbound file could not be read")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });

  it("steps to the files beside it in the table's order", async () => {
    const { onSelect } = draw();
    await screen.findByRole("heading", { level: 2 });

    fireEvent.click(screen.getByRole("button", { name: "Next file" }));
    fireEvent.click(screen.getByRole("button", { name: "Previous file" }));
    expect(onSelect.mock.calls).toEqual([["ediinf_acme"], ["ediinf_bad"]]);
  });

  it("links to the file's own page, and as an artifact has nowhere to go back to", async () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <DeskEdiInboundFileArtifact
            recordId="ediinf_sim"
            artifact={{} as never}
            fallback={null}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await screen.findByRole("heading", { level: 2 });

    expect(screen.getByRole("link", { name: "Open file" })).toHaveAttribute(
      "href",
      "/edi/inbound-files?panelType=edit&panelEntityId=ediinf_sim",
    );
    expect(screen.queryByRole("button", { name: "Back to the table" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next file" })).toBeNull();
  });
});
