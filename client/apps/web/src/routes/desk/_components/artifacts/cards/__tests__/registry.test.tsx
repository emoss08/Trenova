import { lazyComponent } from "@/lib/lazy-component";
import type { AssistantArtifact } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { BillingItemSkeleton } from "../billing-queue-item/billing-item-skeleton";
import { DeskCustomerSkeleton } from "../customer/customer-skeleton";
import { DeskRateAgreementSkeleton } from "../rate-agreement/rate-agreement-skeleton";
import { DeskFormulaTemplateSkeleton } from "../formula-template/formula-template-skeleton";
import { DeskShipmentSkeleton } from "../shipment/shipment-skeleton";
import {
  InPlaceRecord,
  RecordArtifact,
  recordCardFor,
  type InPlaceRecordProps,
  type RecordArtifactProps,
  type RecordCardEntry,
  useListedRecords,
} from "../registry";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";

vi.mock("../formula-template/desk-formula-template", () => ({
  DeskFormulaTemplate: () => <p>Formula card</p>,
  DeskFormulaTemplateArtifact: () => <p>Formula artifact</p>,
}));
vi.mock("../shipment/desk-shipment", () => ({
  DeskShipment: () => <p>Shipment card</p>,
  DeskShipmentArtifact: () => <p>Shipment artifact</p>,
}));
vi.mock("../customer/desk-customer", () => ({
  DeskCustomer: () => <p>Customer card</p>,
  DeskCustomerArtifact: () => <p>Customer artifact</p>,
}));
vi.mock("../rate-agreement/desk-rate-agreement", () => ({
  DeskRateAgreement: () => <p>Rate agreement card</p>,
  DeskRateAgreementArtifact: () => <p>Rate agreement artifact</p>,
}));
vi.mock("../billing-queue-item/desk-billing-record", () => ({
  DeskBillingRecord: () => <p>Billing card</p>,
  DeskBillingArtifact: () => <p>Billing artifact</p>,
}));

type Cards = {
  InPlace: (props: InPlaceRecordProps) => ReactNode;
  Artifact: (props: RecordArtifactProps) => ReactNode;
};

/** An entry whose code arrives only when the test opens the gate. */
function gatedEntry(Skeleton: RecordCardEntry["Skeleton"]) {
  let open: (module: Cards) => void = () => {};
  const loaded = new Promise<Cards>((resolve) => {
    open = resolve;
  });
  const entry: RecordCardEntry = {
    resource: "formula_template",
    Skeleton,
    inPlace: lazyComponent(
      () => loaded,
      (module) => module.InPlace,
    ),
    artifact: lazyComponent(
      () => loaded,
      (module) => module.Artifact,
    ),
  };
  return { entry, loaded, open };
}

function draw(node: ReactNode) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>{node}</MemoryRouter>
    </QueryClientProvider>,
  );
}

const artifact = { id: "art_1", lineageId: "", payload: {} } as AssistantArtifact;

/*
A resource card is its own chunk, drawn in place of a table or as an artifact
of its own. Until its code arrives the pane holds the card's own outline, drawn
on the card's layout, and the outline gives way to the card once it lands.
*/
describe("the Desk's resource cards", () => {
  it("draws a formula template, a billing item, a shipment and a customer each with its own outline", () => {
    expect(recordCardFor("formula_template")?.Skeleton).toBe(DeskFormulaTemplateSkeleton);
    expect(recordCardFor("billing_queue_item")?.Skeleton).toBe(BillingItemSkeleton);
    expect(recordCardFor("shipment")?.Skeleton).toBe(DeskShipmentSkeleton);
    expect(recordCardFor("customer")?.Skeleton).toBe(DeskCustomerSkeleton);
    expect(recordCardFor("rate_agreement")?.Skeleton).toBe(DeskRateAgreementSkeleton);
  });

  it("leaves a record without a card, or no record at all, to open its page", () => {
    expect(recordCardFor("worker")).toBeNull();
    expect(recordCardFor("")).toBeNull();
    expect(recordCardFor("not_a_record")).toBeNull();
  });

  it("reads each card's surfaces from the card's own module", async () => {
    const formula = recordCardFor("formula_template");
    const billing = recordCardFor("billing_queue_item");
    const shipment = recordCardFor("shipment");
    const customer = recordCardFor("customer");
    const agreement = recordCardFor("rate_agreement");
    const props = { recordId: "x_1", order: [], onSelect: vi.fn(), onBack: vi.fn() };

    draw(
      <>
        {formula && <InPlaceRecord entry={formula} {...props} />}
        {formula && (
          <RecordArtifact entry={formula} artifact={artifact} recordId="x_1" fallback={null} />
        )}
        {billing && <InPlaceRecord entry={billing} {...props} />}
        {billing && (
          <RecordArtifact entry={billing} artifact={artifact} recordId="x_1" fallback={null} />
        )}
        {shipment && <InPlaceRecord entry={shipment} {...props} />}
        {shipment && (
          <RecordArtifact entry={shipment} artifact={artifact} recordId="x_1" fallback={null} />
        )}
        {customer && <InPlaceRecord entry={customer} {...props} />}
        {customer && (
          <RecordArtifact entry={customer} artifact={artifact} recordId="x_1" fallback={null} />
        )}
        {agreement && <InPlaceRecord entry={agreement} {...props} />}
        {agreement && (
          <RecordArtifact entry={agreement} artifact={artifact} recordId="x_1" fallback={null} />
        )}
      </>,
    );

    expect(await screen.findByText("Formula card")).toBeInTheDocument();
    expect(await screen.findByText("Formula artifact")).toBeInTheDocument();
    expect(await screen.findByText("Billing card")).toBeInTheDocument();
    expect(await screen.findByText("Billing artifact")).toBeInTheDocument();
    expect(await screen.findByText("Shipment card")).toBeInTheDocument();
    expect(await screen.findByText("Shipment artifact")).toBeInTheDocument();
    expect(await screen.findByText("Customer card")).toBeInTheDocument();
    expect(await screen.findByText("Customer artifact")).toBeInTheDocument();
    expect(await screen.findByText("Rate agreement card")).toBeInTheDocument();
    expect(await screen.findByText("Rate agreement artifact")).toBeInTheDocument();
  });

  // A record opened in place takes over the pane its table or listing card was
  // scrolled in: it starts at the top, so does each record stepped to, and the
  // way back finds the pane where it was left.
  it("opens a record at the top of its pane, and goes back to where the pane was left", async () => {
    const pane = document.createElement("div");
    pane.style.overflowY = "auto";
    Object.defineProperty(pane, "scrollTop", { value: 480, writable: true, configurable: true });
    document.body.append(pane);
    const formula = recordCardFor("formula_template");
    if (!formula) throw new Error("No formula template card");
    const opened = (recordId: string) => (
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <InPlaceRecord
            entry={formula}
            recordId={recordId}
            order={["ft_1", "ft_2"]}
            onSelect={vi.fn()}
            onBack={vi.fn()}
          />
        </MemoryRouter>
      </QueryClientProvider>
    );

    const view = render(opened("ft_1"), { container: pane });
    expect(await screen.findByText("Formula card")).toBeInTheDocument();
    expect(pane.scrollTop).toBe(0);

    pane.scrollTop = 220;
    view.rerender(opened("ft_2"));
    expect(pane.scrollTop).toBe(0);

    await act(async () => {
      view.rerender(<p>The table</p>);
      await Promise.resolve();
    });
    expect(pane.scrollTop).toBe(480);

    view.unmount();
    pane.remove();
  });

  // A card that lists records of another kind opens them in its own place only
  // when the kind has a card and the person may read it; otherwise each links
  // to the record's page, which says for itself what the person may see.
  it("opens a listed record in place only for a kind with a card the person may read", () => {
    function Listed({ entity }: { entity: "shipment" | "customer" | "worker" }) {
      const listed = useListedRecords({ entity, order: ["x_1"], backLabel: "Back" });
      return <p>{`${entity}: ${listed.open ? "opens in place" : "links to its page"}`}</p>;
    }
    seedPermissions({ [Resource.Shipment]: Operation.Read });

    draw(
      <>
        <Listed entity="shipment" />
        <Listed entity="customer" />
        <Listed entity="worker" />
      </>,
    );

    expect(screen.getByText("shipment: opens in place")).toBeInTheDocument();
    expect(screen.getByText("customer: links to its page")).toBeInTheDocument();
    expect(screen.getByText("worker: links to its page")).toBeInTheDocument();
    clearPermissions();
  });

  it("holds a table's pane with the card's outline until the card's code is read", async () => {
    const gate = gatedEntry(DeskFormulaTemplateSkeleton);

    const { container } = draw(
      <InPlaceRecord
        entry={gate.entry}
        recordId="ft_1"
        order={["ft_1", "ft_2"]}
        onSelect={vi.fn()}
        onBack={vi.fn()}
      />,
    );

    expect(screen.getByRole("status")).toHaveTextContent("Loading the formula template…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-head")).not.toBeNull();
    expect(container.querySelector(".dk-rec-sk-step")).not.toBeNull();

    await act(async () => {
      gate.open({
        InPlace: ({ recordId }) => <p>Card for {recordId}</p>,
        Artifact: () => null,
      });
      await gate.loaded;
    });

    expect(await screen.findByText("Card for ft_1")).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
  });

  it("draws the outline without a stepper when the table has one row", () => {
    const gate = gatedEntry(DeskFormulaTemplateSkeleton);

    const { container } = draw(
      <InPlaceRecord
        entry={gate.entry}
        recordId="ft_1"
        order={["ft_1"]}
        onSelect={vi.fn()}
        onBack={vi.fn()}
      />,
    );

    expect(container.querySelector(".dk-rec-skel .dk-rec-head")).not.toBeNull();
    expect(container.querySelector(".dk-rec-sk-step")).toBeNull();
  });

  it("holds an artifact's pane with the card's outline until the card's code is read", async () => {
    const gate = gatedEntry(BillingItemSkeleton);

    const { container } = draw(
      <RecordArtifact
        entry={gate.entry}
        artifact={artifact}
        recordId="bqi_1"
        fallback={<p>Server view</p>}
      />,
    );

    expect(screen.getByRole("status")).toHaveTextContent("Loading the billing item…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-checks")).not.toBeNull();
    expect(container.querySelector(".dk-rec-sk-step")).toBeNull();

    await act(async () => {
      gate.open({
        InPlace: () => null,
        Artifact: ({ recordId, fallback }) => (
          <div>
            <p>Artifact for {recordId}</p>
            {fallback}
          </div>
        ),
      });
      await gate.loaded;
    });

    expect(await screen.findByText("Artifact for bqi_1")).toBeInTheDocument();
    expect(screen.getByText("Server view")).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
  });
});
