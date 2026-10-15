import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { edi } from "@/lib/queries/edi";
import { money } from "../../../record-card/record-card-values";
import { DeskEdiTransfer, DeskEdiTransferArtifact } from "../desk-edi-transfer";
import { wireMappingPreview, wireTransfer } from "./fixtures";

const server = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, get: server.get } };
});
// The shipment and message cards are chunks of their own; here each stands in
// as what it was given, with its way back.
const standIn =
  (name: string) =>
  ({
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
        {name} card {recordId}
      </p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  );
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: standIn("Shipment"),
  DeskShipmentArtifact: () => null,
}));
vi.mock("../../edi-message/desk-edi-message", () => ({
  DeskEdiMessage: standIn("Message"),
  DeskEdiMessageArtifact: () => null,
}));

type Routes = Record<string, () => unknown>;
let routes: Routes;

function route(overrides: Routes = {}) {
  routes = {
    "/edi/transfers/edilt_acme_88241/": () => wireTransfer("mappingRequired"),
    "/edi/transfers/edilt_acme_88241/mapping-preview/": () => wireMappingPreview(),
    "/edi/transfers/edilt_acme_88240/": () => wireTransfer("approved"),
    "/edi/transfers/edilt_acme_88199/": () => wireTransfer("rejected"),
    "/edi/transfers/edilt_s2610_1366/": () => wireTransfer("outbound"),
    ...overrides,
  };
}

function draw(props: Partial<Parameters<typeof DeskEdiTransfer>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const tree = (more: Partial<typeof props> = {}) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskEdiTransfer
          recordId="edilt_acme_88241"
          order={["edilt_acme_88199", "edilt_acme_88241", "edilt_acme_88240"]}
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
    queryClient,
  };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

const asked = (url: string) => server.get.mock.calls.some(([called]) => called === url);

beforeEach(() => {
  seedPermissions({ [Resource.EDI]: Operation.Read, [Resource.Shipment]: Operation.Read });
  useAuthStore.setState({
    user: { currentOrganizationId: "org_1" } as ReturnType<typeof useAuthStore.getState>["user"],
  });
  route();
  server.get.mockReset().mockImplementation(async (url: string) => {
    const answer = routes[url];
    if (!answer) throw new Error(`Not found: ${url}`);
    return answer();
  });
});

afterEach(() => {
  clearPermissions();
  useAuthStore.setState({ user: null });
  vi.restoreAllMocks();
});

describe("DeskEdiTransfer", () => {
  it("draws the next tender's outline, not the one it left, while the next one is read", async () => {
    const { container, step } = draw();
    await screen.findByRole("heading", { name: "LT-88241", level: 2 });

    server.get.mockReturnValue(new Promise(() => {}));
    step("edilt_acme_88240");

    expect(screen.queryByRole("heading", { name: "LT-88241", level: 2 })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("holds the card's outline while the tender is read, then draws who tendered it and what it holds", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the EDI transfer…");
    expect(container.querySelector(".dk-rec-skel .dk-tdr-gates")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-tdr-route")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-tdr-ticket")).not.toBeNull();

    expect(await screen.findByRole("heading", { name: "LT-88241", level: 2 })).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(document.querySelector(".dk-rec-kick")).toHaveTextContent(
      /Inbound tender.*ACME.*Change/u,
    );
    expect(screen.getByText("From Acme Foods")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Needs mapping");
    expect(document.querySelector(".dk-rec-amt")).toHaveTextContent(money("2880"));

    const gates = within(section("Decision")).getAllByRole("listitem");
    expect(gates.map((gate) => gate.textContent)).toEqual([
      expect.stringContaining("Read from an EDI 204"),
      expect.stringContaining("3 of 5 mapped"),
      "Decided",
      "Shipment",
    ]);
  });

  it("reads the tender's mappings live while it waits on a decision", async () => {
    draw();
    await screen.findByRole("heading", { level: 2 });

    const mappings = section("Mappings");
    expect(within(mappings).getByText("3 of 5 mapped")).toBeInTheDocument();
    expect(within(mappings).getByRole("meter", { name: "3 of 5 mapped" })).toHaveAttribute(
      "aria-valuenow",
      "3",
    );
    expect(within(mappings).getAllByText("Not mapped")).toHaveLength(2);
    expect(within(section("Route")).getByText("Not mapped")).toBeInTheDocument();
    expect(within(section("Route")).getByText("Ours: Acme Plant 2 (Dallas)")).toBeInTheDocument();
    expect(asked("/edi/transfers/edilt_acme_88241/mapping-preview/")).toBe(true);
    expect(
      screen.getByText("Map its codes and decide on the inbound transfers page"),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open tender" })).toHaveAttribute(
      "href",
      "/edi/transfers/inbound?panelType=edit&panelEntityId=edilt_acme_88241",
    );
  });

  it("writes the freight and its rate down to the total", async () => {
    draw({ recordId: "edilt_acme_88240" });
    await screen.findByRole("heading", { name: "LT-88240", level: 2 });

    const freight = section("Freight and rate");
    expect(within(freight).getByText("Frozen poultry")).toBeInTheDocument();
    expect(within(freight).getByText("34–38 °F")).toBeInTheDocument();
    expect(within(freight).getByText("Fuel surcharge")).toBeInTheDocument();
    expect(within(freight).getByText("Total").nextSibling).toHaveTextContent(money("2880"));
    expect(asked("/edi/transfers/edilt_acme_88240/mapping-preview/")).toBe(false);
    expect(
      screen.getByText("Its history and the shipment link are on the transfers page"),
    ).toBeInTheDocument();
  });

  it("opens the shipment an accepted tender built in the card's place, and comes back", async () => {
    draw({ recordId: "edilt_acme_88240" });
    await screen.findByRole("heading", { name: "LT-88240", level: 2 });

    const records = section("Records it names");
    expect(within(records).getByRole("button", { name: /Load tender 204/u })).toBeInTheDocument();
    fireEvent.click(within(records).getByRole("button", { name: /Shipment it built/u }));

    expect(await screen.findByText("Shipment card shp_acme_88240")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to LT-88240" }));
    expect(await screen.findByRole("heading", { name: "LT-88240", level: 2 })).toBeInTheDocument();
  });

  it("leaves out the shipment for someone who may not read shipments", async () => {
    seedPermissions({ [Resource.EDI]: Operation.Read });
    draw({ recordId: "edilt_acme_88240" });
    await screen.findByRole("heading", { name: "LT-88240", level: 2 });

    const records = section("Records it names");
    expect(within(records).queryByText("Shipment it built")).toBeNull();
    expect(within(records).getByText("Load tender 204")).toBeInTheDocument();
  });

  it("keeps why a tender was declined, and says it is closed", async () => {
    draw({ recordId: "edilt_acme_88199" });
    await screen.findByRole("heading", { name: "LT-88199", level: 2 });

    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Declined");
    expect(
      within(section("Decision")).getByRole("note", { name: "Why it was declined" }),
    ).toHaveTextContent("No reefer capacity on the Dallas to Houston lane that day");
    expect(screen.getByText("Its history is on the transfers page")).toBeInTheDocument();
  });

  it("reads a tender we sent from our side, naming them as we know them", async () => {
    draw({ recordId: "edilt_s2610_1366" });
    await screen.findByRole("heading", { name: "BOL-55120", level: 2 });

    expect(
      [...document.querySelectorAll(".dk-rec-kick > span:not(.dk-rec-nav)")].map(
        (part) => part.textContent,
      ),
    ).toEqual(["Outbound tender", "LONESTAR"]);
    expect(screen.getByText("To Lone Star Logistics")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Awaiting their decision");
    expect(screen.getByText("Lone Star Logistics decides on their side")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open tender" })).toHaveAttribute(
      "href",
      "/edi/transfers/inbound?panelType=edit&panelEntityId=edilt_s2610_1366",
    );
    expect(
      within(section("Records it names")).getByRole("button", { name: /Shipment tendered/u }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Mappings", level: 3 })).toBeNull();
    expect(asked("/edi/transfers/edilt_s2610_1366/mapping-preview/")).toBe(false);
  });

  it("holds its outline until it knows which organization is reading", async () => {
    useAuthStore.setState({ user: null });
    const { container, queryClient } = draw({ recordId: "edilt_s2610_1366" });

    await vi.waitFor(() =>
      expect(queryClient.getQueryState(edi.transfer("edilt_s2610_1366").queryKey)?.status).toBe(
        "success",
      ),
    );
    await act(async () => {});
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
    expect(screen.queryByText("Outbound tender")).toBeNull();
    expect(screen.queryByText("Inbound tender")).toBeNull();
  });

  it("says the tender could not be read, and keeps the way back", async () => {
    route({
      "/edi/transfers/edilt_acme_88241/": () => {
        throw new Error("gone");
      },
    });
    const { onBack } = draw();

    expect(await screen.findByText("This EDI transfer could not be read")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });

  it("steps to the tenders beside it in the table's order", async () => {
    const { onSelect } = draw();
    await screen.findByRole("heading", { level: 2 });

    fireEvent.click(screen.getByRole("button", { name: "Next tender" }));
    fireEvent.click(screen.getByRole("button", { name: "Previous tender" }));
    expect(onSelect.mock.calls).toEqual([["edilt_acme_88240"], ["edilt_acme_88199"]]);
  });

  it("has nowhere to go back to as an artifact", async () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <DeskEdiTransferArtifact
            recordId="edilt_acme_88240"
            artifact={{} as never}
            fallback={null}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await screen.findByRole("heading", { level: 2 });

    expect(screen.queryByRole("button", { name: "Back to the table" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next tender" })).toBeNull();
  });
});
