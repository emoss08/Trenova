import { recordPath } from "@/config/record-links";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { edi } from "@/lib/queries/edi";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeApiError } from "@/lib/api-error-message";
import { cn } from "@trenova/shared/lib/utils";
import type { EDIMessage } from "@trenova/shared/types/edi";
import { useMemo } from "react";
import { Link } from "react-router";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardRecordRow,
  CardSection,
  CardSectionError,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { plainReading } from "../../record-card/edi-reading";
import { transactionName } from "../../record-card/edi-transaction-sets";
import { cardMoment, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  diagnosticTally,
  envelopeLayers,
  messageJourney,
  messageLinks,
  messageStanding,
  tapeSegments,
  type MessageLink,
  wayHalt,
} from "./edi-message-card";
import {
  EnvelopeNest,
  MessageStamp,
  MessageWay,
  PlainReadingView,
  WayHaltSlip,
  X12Tape,
} from "./edi-message-parts";
import { EDI_MESSAGE_SECTIONS } from "./edi-message-sections";
import { DeskEdiMessageSkeleton, TapeSkeleton } from "./edi-message-skeleton";

/**
 * What the card is given: the message, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type EdiMessageCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the message was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One EDI message as an EDI coordinator or a dispatcher reads it, opened from
 * a table's row or looked up by the agent: which document it is and who it
 * went to or came from, how far it travelled (generated, delivered,
 * acknowledged; or received, read and tendered), what it says in plain words,
 * the envelope it travels in with its control numbers, the X12 itself with
 * every finding pinned to its segment, and the records it names. It reads;
 * retrying delivery and replaying stay on the EDI messages page, one click
 * away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskEdiMessage({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: EdiMessageCardProps) {
  const t = useT();
  const query = useQuery({
    ...edi.message(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const message = query.data;

  if (!message) {
    return query.isError ? (
      <CardUnreadable
        kind={t("EDI message")}
        title={t("This EDI message could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskEdiMessageSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <EdiMessageCard
      key={message.id}
      message={message}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The message as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskEdiMessageArtifact({ recordId }: RecordArtifactProps) {
  return <DeskEdiMessage recordId={recordId} />;
}

function EdiMessageCard({
  message,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<EdiMessageCardProps, "order" | "recordId"> & {
  message: EDIMessage;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const access = useCardAccess(EDI_MESSAGE_SECTIONS);
  const id = message.id;
  const purged = message.rawPurgedAt !== null && message.rawPurgedAt !== undefined;
  const inspection = useQuery({
    ...edi.messageInspection(id),
    retry: false,
    enabled: !purged,
  });

  const standing = messageStanding(message, t);
  const neighbors = neighborsIn(rowOrder, id);
  const partner = message.partner ?? null;
  const reading = plainReading(message, t);
  const journey = messageJourney(message, t);
  const layers = envelopeLayers(message, inspection.data?.inspection, t);
  const tape = inspection.data ? tapeSegments(inspection.data.inspection) : null;
  const tally = diagnosticTally(
    inspection.data?.inspection.diagnostics ?? message.validationErrors,
  );
  const links = messageLinks(message, t).filter(
    (link) =>
      (link.entity !== "shipment" || access.shipment) &&
      (link.entity !== "invoice" || access.invoice),
  );
  const halt = wayHalt(message, t);

  const backToMessage = t("Back to {0}", transactionName(message.transactionSet, t));
  const shipmentIds = useIds(links, "shipment");
  const messageIds = useIds(links, "edi_message");
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipmentIds,
    backLabel: backToMessage,
  });
  const listedMessages = useListedRecords({
    entity: "edi_message",
    order: messageIds,
    backLabel: backToMessage,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedShipments.opened ?? listedMessages.opened;
  if (opened) {
    return opened;
  }

  const openLink = (link: MessageLink) =>
    link.entity === "shipment"
      ? listedShipments.open && (() => listedShipments.open?.(link.id))
      : link.entity === "edi_message"
        ? listedMessages.open && (() => listedMessages.open?.(link.id))
        : undefined;

  return (
    <RecordCard className={cn("dk-edi", `dk-edi-${message.direction.toLowerCase()}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          message.direction === "Inbound" ? t("Inbound") : t("Outbound"),
          partner?.code ?? "",
          t("X12 {0}", message.x12Version),
        ]}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous message")}
              nextLabel={t("Next message")}
            />
          )
        }
        lead={<MessageStamp set={message.transactionSet} direction={message.direction} />}
        title={transactionName(message.transactionSet, t)}
        subtitle={
          partner
            ? message.direction === "Inbound"
              ? t("From {0}", partner.name)
              : t("To {0}", partner.name)
            : t("Partner not found")
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                message.direction === "Inbound"
                  ? cardMoment(message.generatedAt)
                  : t("Generated {0}", cardMoment(message.generatedAt)),
                message.diagnosticCount > 0
                  ? t("{0, plural, one {# finding} other {# findings}}", message.diagnosticCount)
                  : "",
                purged ? t("Raw X12 purged") : "",
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection title={t("Its way")}>
        <MessageWay stations={journey} />
        {halt && <WayHaltSlip halt={halt} stations={journey} />}
      </CardSection>

      <CardSection title={t("What it says")}>
        {reading ? (
          <PlainReadingView reading={reading} />
        ) : (
          <p className="dk-rec-note">{t("No decoded copy of this message was kept.")}</p>
        )}
      </CardSection>

      <CardSection
        title={t("Envelope")}
        aside={
          message.deliveryRemotePath ? (
            <span className="dk-edi-path">{message.deliveryRemotePath}</span>
          ) : undefined
        }
      >
        <EnvelopeNest layers={layers} />
      </CardSection>

      <CardSection
        title={t("The X12")}
        aside={purged || !tape ? undefined : <TapeAside message={message} tally={tally} />}
      >
        {purged ? (
          <p className="dk-rec-note">
            {reading
              ? t(
                  "The raw X12 was purged {0} under the retention policy; what it said is kept above.",
                  cardMoment(message.rawPurgedAt),
                )
              : t(
                  "The raw X12 was purged {0} under the retention policy.",
                  cardMoment(message.rawPurgedAt),
                )}
          </p>
        ) : inspection.isError ? (
          <CardSectionError>
            {describeApiError(inspection.error, t("The X12 could not be inspected."))}
          </CardSectionError>
        ) : tape ? (
          <X12Tape rows={tape.rows} loose={tape.loose} />
        ) : (
          <TapeSkeleton />
        )}
      </CardSection>

      {links.length > 0 && (
        <CardSection title={t("Records it names")}>
          <div className="dk-rec-list">
            {links.map((link) => (
              <CardRecordRow
                key={link.key}
                to={recordPath(link.entity, link.id)}
                onOpen={openLink(link)}
              >
                <span className="dk-rec-main">
                  <b>{link.title}</b>
                  <span className="dk-rec-sub">{link.line}</span>
                </span>
                <CardIcon name="forward" size={12} />
              </CardRecordRow>
            ))}
          </div>
        </CardSection>
      )}

      <CardBar>
        <CardBarHint>
          {message.direction === "Inbound"
            ? t("Reprocess the file it came in on the EDI inbound files page")
            : t("Retry delivery or replay it on the EDI messages page")}
        </CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("edi_message", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open message")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** The ids of the listed records of one kind, held still while the list is. */
function useIds(links: readonly MessageLink[], entity: MessageLink["entity"]): readonly string[] {
  return useMemo(() => {
    const ids = links.filter((link) => link.entity === entity).map((link) => link.id);
    return ids.length > 0 ? ids : NO_ROWS;
  }, [links, entity]);
}

/** What the validator found across the X12, and the way to copy it whole. */
function TapeAside({
  message,
  tally,
}: {
  message: EDIMessage;
  tally: { errors: number; warnings: number };
}) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();
  const clean = tally.errors === 0 && tally.warnings === 0;
  return (
    <span className="dk-edi-aside">
      <span className={cn(clean ? "dk-ok" : tally.errors > 0 ? "dk-bad" : "dk-warn")}>
        {clean
          ? t("Nothing flagged")
          : [
              tally.errors > 0
                ? t("{0, plural, one {# error} other {# errors}}", tally.errors)
                : "",
              tally.warnings > 0
                ? t("{0, plural, one {# warning} other {# warnings}}", tally.warnings)
                : "",
            ]
              .filter(Boolean)
              .join(" · ")}
      </span>
      <Button
        variant="bare"
        size="bare"
        className="dk-rec-lnk"
        onClick={() => void copy(message.rawX12)}
      >
        <CardIcon name={isCopied ? "check" : "copy"} size={11} />
        {isCopied ? t("Copied") : t("Copy X12")}
      </Button>
    </span>
  );
}
