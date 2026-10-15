import type { RecordEntityType } from "@/config/record-links";
import { useScrollFromTop } from "@/hooks/use-scroll-from-top";
import { useCanRead } from "../record-card/card-access";
import { lazyComponent, preloadable, type LazyComponent } from "@/lib/lazy-component";
import { whenIdle } from "@/lib/when-idle";
import { Resource } from "@trenova/shared/types/permission";
import type { AssistantArtifact } from "@/types/assistant";
import { Suspense, useEffect, useState, type ComponentType, type ReactNode } from "react";
import { BillingItemSkeleton } from "./billing-queue-item/billing-item-skeleton";
import { DeskCustomerSkeleton } from "./customer/customer-skeleton";
import { DeskRateAgreementSkeleton } from "./rate-agreement/rate-agreement-skeleton";
import { DeskFormulaTemplateSkeleton } from "./formula-template/formula-template-skeleton";
import { DeskShipmentSkeleton } from "./shipment/shipment-skeleton";
import { DeskTractorSkeleton } from "./tractor/tractor-skeleton";
import { DeskTrailerSkeleton } from "./trailer/trailer-skeleton";
import { DeskWorkerSkeleton } from "./worker/worker-skeleton";
import { DeskLocationSkeleton } from "./location/location-skeleton";
import { DeskCarrierSkeleton } from "./carrier/carrier-skeleton";
import { DeskOrderSkeleton } from "./order/order-skeleton";
import { DeskServiceFailureSkeleton } from "./service-failure/service-failure-skeleton";
import { DeskEdiMessageSkeleton } from "./edi-message/edi-message-skeleton";
import { DeskEdiInboundFileSkeleton } from "./edi-inbound-file/edi-inbound-file-skeleton";
import { DeskEdiTransferSkeleton } from "./edi-transfer/edi-transfer-skeleton";
import { DeskInvoiceSkeleton } from "./invoice/invoice-skeleton";
import { DeskDriverSettlementSkeleton } from "./driver-settlement/driver-settlement-skeleton";
import { DeskCarrierSettlementSkeleton } from "./carrier-settlement/carrier-settlement-skeleton";
import { DeskDetentionOccurrenceSkeleton } from "./detention-occurrence/detention-occurrence-skeleton";

/*
The Desk's resource cards: a record drawn for what a person reads that kind of
record for. A card is drawn in two places: in place of a table, when one of its
rows is opened, and as an artifact of its own, when the agent looked the record
up. Each card is its own chunk, read the first time it is wanted, or while the
browser is idle once a table of its kind is on screen; its skeleton is drawn in
the card's own layout and stays in the workspace's chunk, so the wait for the
code looks like the wait for the record. A new kind of card is one entry here.
*/

/** What a card opened from a table's row is given. */
export type InPlaceRecordProps = {
  recordId: string;
  /** The table's row ids as shown, filtered and sorted, for stepping through them. */
  order: readonly string[];
  onSelect: (recordId: string) => void;
  /** Back to the table, as it was left, or to the card the record was opened from. */
  onBack: () => void;
  /** What the way back says, when it leads somewhere other than the table. */
  backLabel?: string;
};

/** What a card drawn as an artifact of its own is given. */
export type RecordArtifactProps = {
  artifact: AssistantArtifact;
  recordId: string;
  /** What shows when the card cannot read the record: the server's own view of it, when it sent one. */
  fallback: ReactNode;
};

export type RecordCardEntry = {
  /** The permission resource the card's own record is read under. */
  resource: string;
  /** The card's outline, while its code or its record is read. */
  Skeleton: ComponentType<{ stepper?: boolean }>;
  inPlace: LazyComponent<InPlaceRecordProps>;
  artifact: LazyComponent<RecordArtifactProps>;
};

const loadFormulaTemplate = preloadable(() => import("./formula-template/desk-formula-template"));
const loadBillingItem = preloadable(() => import("./billing-queue-item/desk-billing-record"));
const loadShipment = preloadable(() => import("./shipment/desk-shipment"));
const loadCustomer = preloadable(() => import("./customer/desk-customer"));
const loadRateAgreement = preloadable(() => import("./rate-agreement/desk-rate-agreement"));
const loadTractor = preloadable(() => import("./tractor/desk-tractor"));
const loadTrailer = preloadable(() => import("./trailer/desk-trailer"));
const loadWorker = preloadable(() => import("./worker/desk-worker"));
const loadLocation = preloadable(() => import("./location/desk-location"));
const loadCarrier = preloadable(() => import("./carrier/desk-carrier"));
const loadOrder = preloadable(() => import("./order/desk-order"));
const loadServiceFailure = preloadable(() => import("./service-failure/desk-service-failure"));
const loadEdiMessage = preloadable(() => import("./edi-message/desk-edi-message"));
const loadEdiInboundFile = preloadable(() => import("./edi-inbound-file/desk-edi-inbound-file"));
const loadEdiTransfer = preloadable(() => import("./edi-transfer/desk-edi-transfer"));
const loadInvoice = preloadable(() => import("./invoice/desk-invoice"));
const loadDriverSettlement = preloadable(
  () => import("./driver-settlement/desk-driver-settlement"),
);
const loadCarrierSettlement = preloadable(
  () => import("./carrier-settlement/desk-carrier-settlement"),
);
const loadDetentionOccurrence = preloadable(
  () => import("./detention-occurrence/desk-detention-occurrence"),
);

const RECORD_CARDS: Partial<Record<RecordEntityType, RecordCardEntry>> = {
  formula_template: {
    resource: Resource.FormulaTemplate,
    Skeleton: DeskFormulaTemplateSkeleton,
    inPlace: lazyComponent(loadFormulaTemplate, (module) => module.DeskFormulaTemplate),
    artifact: lazyComponent(loadFormulaTemplate, (module) => module.DeskFormulaTemplateArtifact),
  },
  billing_queue_item: {
    resource: Resource.BillingQueue,
    Skeleton: BillingItemSkeleton,
    inPlace: lazyComponent(loadBillingItem, (module) => module.DeskBillingRecord),
    artifact: lazyComponent(loadBillingItem, (module) => module.DeskBillingArtifact),
  },
  shipment: {
    resource: Resource.Shipment,
    Skeleton: DeskShipmentSkeleton,
    inPlace: lazyComponent(loadShipment, (module) => module.DeskShipment),
    artifact: lazyComponent(loadShipment, (module) => module.DeskShipmentArtifact),
  },
  customer: {
    resource: Resource.Customer,
    Skeleton: DeskCustomerSkeleton,
    inPlace: lazyComponent(loadCustomer, (module) => module.DeskCustomer),
    artifact: lazyComponent(loadCustomer, (module) => module.DeskCustomerArtifact),
  },
  rate_agreement: {
    resource: Resource.RateAgreement,
    Skeleton: DeskRateAgreementSkeleton,
    inPlace: lazyComponent(loadRateAgreement, (module) => module.DeskRateAgreement),
    artifact: lazyComponent(loadRateAgreement, (module) => module.DeskRateAgreementArtifact),
  },
  tractor: {
    resource: Resource.Tractor,
    Skeleton: DeskTractorSkeleton,
    inPlace: lazyComponent(loadTractor, (module) => module.DeskTractor),
    artifact: lazyComponent(loadTractor, (module) => module.DeskTractorArtifact),
  },
  trailer: {
    resource: Resource.Trailer,
    Skeleton: DeskTrailerSkeleton,
    inPlace: lazyComponent(loadTrailer, (module) => module.DeskTrailer),
    artifact: lazyComponent(loadTrailer, (module) => module.DeskTrailerArtifact),
  },
  worker: {
    resource: Resource.Worker,
    Skeleton: DeskWorkerSkeleton,
    inPlace: lazyComponent(loadWorker, (module) => module.DeskWorker),
    artifact: lazyComponent(loadWorker, (module) => module.DeskWorkerArtifact),
  },
  location: {
    resource: Resource.Location,
    Skeleton: DeskLocationSkeleton,
    inPlace: lazyComponent(loadLocation, (module) => module.DeskLocation),
    artifact: lazyComponent(loadLocation, (module) => module.DeskLocationArtifact),
  },
  carrier: {
    resource: Resource.Carrier,
    Skeleton: DeskCarrierSkeleton,
    inPlace: lazyComponent(loadCarrier, (module) => module.DeskCarrier),
    artifact: lazyComponent(loadCarrier, (module) => module.DeskCarrierArtifact),
  },
  order: {
    resource: Resource.Order,
    Skeleton: DeskOrderSkeleton,
    inPlace: lazyComponent(loadOrder, (module) => module.DeskOrder),
    artifact: lazyComponent(loadOrder, (module) => module.DeskOrderArtifact),
  },
  service_failure: {
    resource: Resource.ServiceFailure,
    Skeleton: DeskServiceFailureSkeleton,
    inPlace: lazyComponent(loadServiceFailure, (module) => module.DeskServiceFailure),
    artifact: lazyComponent(loadServiceFailure, (module) => module.DeskServiceFailureArtifact),
  },
  edi_message: {
    resource: Resource.EDI,
    Skeleton: DeskEdiMessageSkeleton,
    inPlace: lazyComponent(loadEdiMessage, (module) => module.DeskEdiMessage),
    artifact: lazyComponent(loadEdiMessage, (module) => module.DeskEdiMessageArtifact),
  },
  edi_inbound_file: {
    resource: Resource.EDI,
    Skeleton: DeskEdiInboundFileSkeleton,
    inPlace: lazyComponent(loadEdiInboundFile, (module) => module.DeskEdiInboundFile),
    artifact: lazyComponent(loadEdiInboundFile, (module) => module.DeskEdiInboundFileArtifact),
  },
  edi_transfer: {
    resource: Resource.EDI,
    Skeleton: DeskEdiTransferSkeleton,
    inPlace: lazyComponent(loadEdiTransfer, (module) => module.DeskEdiTransfer),
    artifact: lazyComponent(loadEdiTransfer, (module) => module.DeskEdiTransferArtifact),
  },
  invoice: {
    resource: Resource.Invoice,
    Skeleton: DeskInvoiceSkeleton,
    inPlace: lazyComponent(loadInvoice, (module) => module.DeskInvoice),
    artifact: lazyComponent(loadInvoice, (module) => module.DeskInvoiceArtifact),
  },
  driver_settlement: {
    resource: Resource.DriverSettlement,
    Skeleton: DeskDriverSettlementSkeleton,
    inPlace: lazyComponent(loadDriverSettlement, (module) => module.DeskDriverSettlement),
    artifact: lazyComponent(loadDriverSettlement, (module) => module.DeskDriverSettlementArtifact),
  },
  carrier_settlement: {
    resource: Resource.CarrierSettlement,
    Skeleton: DeskCarrierSettlementSkeleton,
    inPlace: lazyComponent(loadCarrierSettlement, (module) => module.DeskCarrierSettlement),
    artifact: lazyComponent(
      loadCarrierSettlement,
      (module) => module.DeskCarrierSettlementArtifact,
    ),
  },
  detention_occurrence: {
    resource: Resource.Detention,
    Skeleton: DeskDetentionOccurrenceSkeleton,
    inPlace: lazyComponent(loadDetentionOccurrence, (module) => module.DeskDetentionOccurrence),
    artifact: lazyComponent(
      loadDetentionOccurrence,
      (module) => module.DeskDetentionOccurrenceArtifact,
    ),
  },
};

/** The card a kind of record is drawn as, or null when it has none. */
export function recordCardFor(entity: string): RecordCardEntry | null {
  return (entity !== "" && RECORD_CARDS[entity as RecordEntityType]) || null;
}

/** Reads a card's code while the browser is idle, so the first row opened draws at once. */
export function usePreloadRecordCard(entry: RecordCardEntry | null): void {
  useEffect(() => {
    if (!entry) return undefined;
    return whenIdle(() => void entry.inPlace.preload().catch(() => undefined));
  }, [entry]);
}

/**
 * A record opened in place of a table or of the card that listed it, its skeleton
 * holding the pane until its code is read. It opens at the top of the pane, and so
 * does each record stepped to; going back finds the pane where it was left.
 */
export function InPlaceRecord({
  entry,
  ...props
}: InPlaceRecordProps & { entry: RecordCardEntry }) {
  const { Component } = entry.inPlace;
  const anchor = useScrollFromTop<HTMLDivElement>(props.recordId);
  return (
    <div ref={anchor} className="contents">
      <Suspense fallback={<entry.Skeleton stepper={props.order.length > 1} />}>
        <Component {...props} />
      </Suspense>
    </div>
  );
}

/** A record drawn as an artifact of its own, its skeleton holding the pane until its code is read. */
export function RecordArtifact({
  entry,
  ...props
}: RecordArtifactProps & { entry: RecordCardEntry }) {
  const { Component } = entry.artifact;
  return (
    <Suspense fallback={<entry.Skeleton />}>
      <Component {...props} />
    </Suspense>
  );
}

/** How a card opens the records of another kind it lists. */
export type ListedRecords = {
  /**
   * Opens a listed record in the card's place. Absent when the kind has no
   * card here, and the card links each row to the record's page instead.
   */
  open: ((recordId: string) => void) | undefined;
  /** The listed record open now, drawn in the card's place; null while the card itself shows. */
  opened: ReactNode;
};

/**
 * The records of another kind a card lists, such as a customer's shipments,
 * opened the way a table opens its rows: in the card's place as the record's
 * own card, stepping through the list in the order shown, with the way back
 * to the card that listed them. A kind with no card here, or one the person may
 * not read, opens on its page.
 */
export function useListedRecords({
  entity,
  order,
  backLabel,
}: {
  entity: RecordEntityType;
  /** The listed records' ids, in the order the card shows them. */
  order: readonly string[];
  /** What the way back to the listing card says. */
  backLabel: string;
}): ListedRecords {
  const card = recordCardFor(entity);
  const entry = useCanRead(card?.resource ?? null) ? card : null;
  const [recordId, setRecordId] = useState<string | null>(null);
  usePreloadRecordCard(order.length > 0 ? entry : null);

  return {
    open: entry ? setRecordId : undefined,
    opened:
      entry && recordId ? (
        <InPlaceRecord
          entry={entry}
          recordId={recordId}
          order={order}
          onSelect={setRecordId}
          onBack={() => setRecordId(null)}
          backLabel={backLabel}
        />
      ) : null,
  };
}
