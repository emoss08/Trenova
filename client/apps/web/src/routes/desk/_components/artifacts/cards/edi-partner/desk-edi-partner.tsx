import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import type {
  DeskEdiPartnerCard,
  EdiPartnerMessage,
  EdiPartnerMessagesPage,
  EdiPartnerTransport,
} from "@/lib/graphql/edi-partner-card";
import { edi, ediPartnerMessagesQuery } from "@/lib/queries/edi";
import { apiService } from "@/services/api";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link01Icon, RefreshCw01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRowTotal } from "@trenova/shared/lib/row-total";
import { cn } from "@trenova/shared/lib/utils";
import type { EDIConnectionTestResult } from "@trenova/shared/types/edi";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import { CardActionButton } from "../../record-card/card-action";
import { CardPagedList } from "../../record-card/card-paged-list";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFactGrid,
  CardFacts,
  CardHead,
  CardIcon,
  CardRecordRow,
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { usePagedRows, type PagedRows } from "../../record-card/paged-rows";
import { cardDate, cardMoment, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  ackCounts,
  ackRuler,
  attentionItems,
  certificateReading,
  connectionExchange,
  connectionFacts,
  credentialChips,
  deliveryFigure,
  headFacts,
  leadTransport,
  mappingChips,
  messageRowLine,
  messageRowStatus,
  messageRowTime,
  partnerAddress,
  partnerExchange,
  partnerKicker,
  partnerLocalTime,
  partnerStanding,
  partnerSubtitle,
  partnerUsage,
  pulseDays,
  pulseFigures,
  readinessChecks,
  transactionSetName,
  transportEnds,
  transportFacts,
  transportLine,
  transportPolls,
  type AttentionItem,
} from "./edi-partner-card";
import {
  AddressLabel,
  CertificateTile,
  CredentialChips,
  MappingChips,
  PartnerContact,
  PartnerExchange,
  PartnerNeed,
  PartnerNeeds,
  PartnerPort,
  PartnerPulse,
  PartnerRef,
  ReadinessList,
  RowState,
  RowTag,
  RowWay,
  TransportPatch,
} from "./edi-partner-parts";
import { DeskEdiPartnerSkeleton } from "./edi-partner-skeleton";

/** The messages the card reads at a time; the next page is read on request. */
const MESSAGE_ROWS = 5;

/**
 * What the card is given: the partner, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type EdiPartnerCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the partner was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One EDI trading partner as the person who keeps it running reads it, opened
 * from a table's row or looked up by the agent: whether it is trading, set up
 * or in trouble; what each side sends the other and what answer it waits for;
 * how the last fortnight went, a column a day, with how quickly acknowledgments
 * came back against what was promised; what wants a look now (pickups failing,
 * dead letters, late acknowledgments, files set aside, a certificate running
 * out, a credential missing), each opening the message or file it is about;
 * the line we reach them over with its certificates and credentials, named and
 * never shown; how interchanges are addressed; whether it is ready to trade;
 * their codes we know; the latest messages; whose EDI it is; and who to call.
 * Testing the line and picking up their files by hand are the person's, shown
 * only to someone who may change EDI settings; setting the partner up stays on
 * its page.
 *
 * The card steps through the rows it was opened from in the order shown.
 * A step draws the next partner's skeleton until it is read, never the one it left.
 */
export function DeskEdiPartner({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: EdiPartnerCardProps) {
  const t = useT();
  const query = useQuery({ ...edi.partnerCard(recordId), retry: false });
  const card = query.data;

  if (!card) {
    return query.isError ? (
      <CardUnreadable
        kind={t("EDI partner")}
        title={t("This EDI partner could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskEdiPartnerSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <PartnerCard
      key={card.partner.id}
      card={card}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The partner as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskEdiPartnerArtifact({ recordId }: RecordArtifactProps) {
  return <DeskEdiPartner recordId={recordId} />;
}

function messageIds(messages: readonly EdiPartnerMessage[] | undefined): readonly string[] {
  return messages && messages.length > 0 ? messages.map((message) => message.id) : NO_ROWS;
}

function PartnerCard({
  card,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<EdiPartnerCardProps, "order">> &
  Omit<EdiPartnerCardProps, "order" | "recordId"> & { card: DeskEdiPartnerCard }) {
  const t = useT();
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const { partner } = card;
  const id = partner.id;
  const internal = partner.kind === "Internal";

  const messages = usePagedRows<EdiPartnerMessage, EdiPartnerMessagesPage>(
    useInfiniteQuery({ ...ediPartnerMessagesQuery(id, MESSAGE_ROWS), retry: false }),
  );
  const items = useMemo(() => attentionItems(card, now, t), [card, now, t]);
  const standing = partnerStanding(card, items.length, t);
  const figure = deliveryFigure(card, t);
  const neighbors = neighborsIn(order, id);
  const usage = partnerUsage(card.documentProfiles);
  const address = internal ? null : partnerAddress(card.documentProfiles);
  const exchange = internal ? connectionExchange(card, t) : partnerExchange(card, t);
  const traded = card.activity.sets.length > 0;
  const checks = readinessChecks(card, t);
  const chips = mappingChips(card, t);

  const backToPartner = t("Back to {0}", partner.name);
  const latestIds = useMemo(() => messageIds(messages.rows), [messages.rows]);
  const attentionMessageIds = useMemo(
    () => messageIds([...card.attention.deadLettered, ...card.attention.overdueAcks]),
    [card.attention.deadLettered, card.attention.overdueAcks],
  );
  const fileIds = useMemo(
    () =>
      card.attention.quarantinedFiles.length > 0
        ? card.attention.quarantinedFiles.map((file) => file.id)
        : NO_ROWS,
    [card.attention.quarantinedFiles],
  );
  const customerIds = useMemo(
    () => (card.customer ? [card.customer.id] : NO_ROWS),
    [card.customer],
  );
  const carrierIds = useMemo(
    () =>
      card.carriers && card.carriers.length > 0
        ? card.carriers.map((entry) => entry.carrierId)
        : NO_ROWS,
    [card.carriers],
  );
  const listedLatest = useListedRecords({
    entity: "edi_message",
    order: latestIds,
    backLabel: backToPartner,
  });
  const listedAttention = useListedRecords({
    entity: "edi_message",
    order: attentionMessageIds,
    backLabel: backToPartner,
  });
  const listedFiles = useListedRecords({
    entity: "edi_inbound_file",
    order: fileIds,
    backLabel: backToPartner,
  });
  const listedCustomer = useListedRecords({
    entity: "customer",
    order: customerIds,
    backLabel: backToPartner,
  });
  const listedCarriers = useListedRecords({
    entity: "carrier",
    order: carrierIds,
    backLabel: backToPartner,
  });

  // A record opened from the card takes the card's place, and its way back
  // returns here.
  const opened =
    listedLatest.opened ??
    listedAttention.opened ??
    listedFiles.opened ??
    listedCustomer.opened ??
    listedCarriers.opened;
  if (opened) {
    return opened;
  }

  const ready = (
    <CardSection
      title={t("Ready to trade")}
      aside={
        card.readiness.ready ? undefined : (
          <span className="dk-warn">
            {t("{0} of {1}", card.readiness.completedCount, card.readiness.totalCount)}
          </span>
        )
      }
    >
      <ReadinessList checks={checks} />
    </CardSection>
  );

  return (
    <RecordCard className="dk-edp">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={partnerKicker(card, t)}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous partner")}
              nextLabel={t("Next partner")}
            />
          )
        }
        lead={
          <PartnerPort inbound={partner.enabledForInbound} outbound={partner.enabledForOutbound} />
        }
        title={partner.name}
        subtitle={partnerSubtitle(card, t)}
        figure={
          figure && (
            <>
              <b className={cn(figure.warn && "dk-warn")}>{figure.value}</b>
              <span>{figure.caption}</span>
            </>
          )
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>{headFacts(card, now, t)}</CardFacts>
          </>
        }
      />

      {items.length > 0 && (
        <CardSection
          title={t("Needs a look")}
          aside={
            <span className={items.some((item) => item.state === "bad") ? "dk-bad" : "dk-warn"}>
              {t("{0, plural, one {# thing} other {# things}}", items.length)}
            </span>
          }
        >
          <PartnerNeeds>
            {items.map((item) => (
              <PartnerNeed key={item.key} item={item}>
                <NeedRecords
                  item={item}
                  onOpenMessage={listedAttention.open}
                  onOpenFile={listedFiles.open}
                />
              </PartnerNeed>
            ))}
          </PartnerNeeds>
        </CardSection>
      )}

      {!internal && !card.readiness.ready && ready}

      <CardSection
        title={traded || internal ? t("What we trade") : t("What we'll trade")}
        aside={traded ? <span>{t("Last 7 days")}</span> : undefined}
      >
        <PartnerExchange
          exchange={exchange}
          note={
            internal
              ? t(
                  "Nothing is enveloped or sent over a line: each side's Trenova hands the other the record.",
                )
              : usage === "test"
                ? t("Every set goes out marked T for test until it is switched to production.")
                : undefined
          }
        />
      </CardSection>

      {!internal && (
        <CardSection
          title={t("How it's going")}
          aside={traded ? <span>{t("Last 14 days")}</span> : undefined}
        >
          {traded || card.activity.series.length > 0 ? (
            <PartnerPulse
              figures={pulseFigures(card, t)}
              days={pulseDays(card.activity.series, now)}
              ruler={ackRuler(card, t)}
              acks={ackCounts(card, t)}
            />
          ) : (
            <p className="dk-rec-note">
              {t("Nothing has been sent to {0} or received from them in 14 days.", partner.name)}
            </p>
          )}
        </CardSection>
      )}

      {internal ? (
        <CardSection title={t("The connection")}>
          <CardFactGrid facts={connectionFacts(card, t)} />
        </CardSection>
      ) : (
        <Transports card={card} now={now} />
      )}

      {address && (
        <CardSection title={t("How it's addressed")}>
          <AddressLabel address={address} usage={usage} />
        </CardSection>
      )}

      {!internal && card.readiness.ready && ready}

      {!internal && (
        <CardSection
          title={t("Their codes we know")}
          aside={
            card.mappings.updatedAt ? (
              <span>{t("Changed {0}", cardDate(card.mappings.updatedAt))}</span>
            ) : undefined
          }
        >
          {chips.length > 0 ? (
            <MappingChips chips={chips} />
          ) : (
            <p className="dk-rec-note">
              {t("None of their codes are matched to our records yet.")}
            </p>
          )}
        </CardSection>
      )}

      {!internal && (
        <CardSection
          title={t("Latest messages")}
          aside={
            messages.total ? (
              <span>
                {messages.totalExact
                  ? t("{0, plural, one {# in all} other {# in all}}", messages.total)
                  : t("About {0} in all", formatRowTotal(messages.total, false))}
              </span>
            ) : undefined
          }
        >
          <LatestMessages messages={messages} onOpen={listedLatest.open} />
        </CardSection>
      )}

      {(card.customer || (card.carriers && card.carriers.length > 0)) && (
        <CardSection title={t("Who it's for")}>
          <div className="dk-edp-refs">
            {card.customer && (
              <PartnerRef
                kind={t("Customer")}
                name={card.customer.name}
                line={card.customer.code}
                to={recordPath("customer", card.customer.id)}
                onOpen={
                  listedCustomer.open && (() => listedCustomer.open?.(card.customer?.id ?? ""))
                }
              />
            )}
            {card.carriers?.map((entry) => (
              <PartnerRef
                key={entry.carrierId}
                kind={entry.isDefault ? t("Carrier, by default") : t("Carrier")}
                name={entry.carrierName}
                line={[entry.carrierCode, entry.scac].filter(Boolean).join(" · ")}
                to={recordPath("carrier", entry.carrierId)}
                onOpen={listedCarriers.open && (() => listedCarriers.open?.(entry.carrierId))}
              />
            ))}
          </div>
        </CardSection>
      )}

      {(partner.contactName || partner.contactEmail || partner.contactPhone) && (
        <CardSection title={t("Who to call")}>
          <PartnerContact
            name={partner.contactName?.trim() || partner.name}
            role={partner.contactName?.trim() ? t("At {0}", partner.name) : ""}
            email={partner.contactEmail?.trim() ?? ""}
            phone={partner.contactPhone?.trim() ?? ""}
            local={partnerLocalTime(partner.timezone, now, t)}
          />
        </CardSection>
      )}

      <PartnerBar card={card} />
    </RecordCard>
  );
}

/** The messages or files a need is about, each opening as its own card. */
function NeedRecords({
  item,
  onOpenMessage,
  onOpenFile,
}: {
  item: AttentionItem;
  onOpenMessage: ((id: string) => void) | undefined;
  onOpenFile: ((id: string) => void) | undefined;
}) {
  const t = useT();
  if (item.messages && item.messages.length > 0) {
    return (
      <div className="dk-rec-list">
        {item.messages.map((message) => {
          const status = messageRowStatus(message, t);
          return (
            <CardRecordRow
              key={message.id}
              to={recordPath("edi_message", message.id)}
              onOpen={onOpenMessage && (() => onOpenMessage(message.id))}
            >
              <RowTag>{message.transactionSet}</RowTag>
              <span className="dk-rec-main">
                <b>{transactionSetName(message.transactionSet, t)}</b>
                <span className="dk-rec-sub">{messageRowLine(message, t)}</span>
              </span>
              <span className="dk-rec-side">
                <RowState state={status.state}>{status.text}</RowState>
                <span className="dk-rec-sub">{cardMoment(message.generatedAt)}</span>
              </span>
            </CardRecordRow>
          );
        })}
      </div>
    );
  }
  if (item.files && item.files.length > 0) {
    return (
      <div className="dk-rec-list">
        {item.files.map((file) => (
          <CardRecordRow
            key={file.id}
            to={recordPath("edi_inbound_file", file.id)}
            onOpen={onOpenFile && (() => onOpenFile(file.id))}
          >
            <RowTag>{t("File")}</RowTag>
            <span className="dk-rec-main">
              <b>{file.fileName}</b>
              <span className="dk-rec-sub">
                {file.failureReason?.trim() || t("From {0}", file.remotePath)}
              </span>
            </span>
            <span className="dk-rec-side">
              <RowState state="warn">{t("Set aside")}</RowState>
              <span className="dk-rec-sub">{cardMoment(file.receivedAt)}</span>
            </span>
          </CardRecordRow>
        ))}
      </div>
    );
  }
  return null;
}

/** The lines we reach the partner over, each as a patch, or a note when none is set up. */
function Transports({ card, now }: { card: DeskEdiPartnerCard; now: number }) {
  const t = useT();
  if (card.transports.length === 0) {
    return (
      <CardSection title={t("How we reach them")}>
        <p className="dk-rec-note">
          {t("No AS2, SFTP or VAN line is set up to reach them. Add one on the partner's page.")}
        </p>
      </CardSection>
    );
  }
  return (
    <CardSection
      title={t("How we reach them")}
      aside={
        card.transports.length > 1 ? (
          <span>{t("{0, plural, one {# line} other {# lines}}", card.transports.length)}</span>
        ) : undefined
      }
    >
      <div className="dk-edp-patches">
        {card.transports.map((transport) => (
          <TransportPatch
            key={transport.profile.id}
            method={transport.profile.method}
            name={transport.profile.name}
            line={transportLine(transport, now, t)}
            ends={transportEnds(transport, t)}
          >
            <TransportBody transport={transport} now={now} />
          </TransportPatch>
        ))}
      </div>
    </CardSection>
  );
}

function TransportBody({ transport, now }: { transport: EdiPartnerTransport; now: number }) {
  const t = useT();
  const facts = transportFacts(transport, now, t);
  return (
    <>
      {facts.length > 0 && <CardFactGrid facts={facts} />}
      {transport.certificates.map((certificate) => (
        <CertificateTile key={certificate.role} reading={certificateReading(certificate, now, t)} />
      ))}
      <CredentialChips chips={credentialChips(transport, t)} />
    </>
  );
}

/** The partner's latest messages both ways, newest first, a page at a time. */
function LatestMessages({
  messages,
  onOpen,
}: {
  messages: PagedRows<EdiPartnerMessage>;
  onOpen: ((messageId: string) => void) | undefined;
}) {
  const t = useT();
  return (
    <CardPagedList
      list={messages}
      waitRows={3}
      emptyText={t("No message has been sent to them or received from them yet.")}
      errorText={t("The partner's messages could not be read.")}
      nextErrorText={t("The next messages could not be read.")}
      loadingLabel={t("Reading more messages…")}
    >
      {(rows) => (
        <div className="dk-rec-list dk-edp-msgs">
          {rows.map((message) => {
            const status = messageRowStatus(message, t);
            return (
              <CardRecordRow
                key={message.id}
                to={recordPath("edi_message", message.id)}
                onOpen={onOpen && (() => onOpen(message.id))}
              >
                <RowWay direction={message.direction} />
                <RowTag>{message.transactionSet}</RowTag>
                <span className="dk-rec-main">
                  <b>{transactionSetName(message.transactionSet, t)}</b>
                  <span className="dk-rec-sub">{messageRowLine(message, t)}</span>
                </span>
                <span className="dk-rec-side">
                  <RowState state={status.state}>{status.text}</RowState>
                  <span className="dk-rec-sub">{messageRowTime(message)}</span>
                </span>
              </CardRecordRow>
            );
          })}
        </div>
      )}
    </CardPagedList>
  );
}

/**
 * The card's own steps: testing the line, and picking up the partner's files
 * now rather than at the next scheduled pickup. Both are the person's, shown
 * only to someone who may change EDI settings, and both say why when they
 * cannot be taken.
 */
function PartnerBar({ card }: { card: DeskEdiPartnerCard }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.EDI, Operation.Update);
  const [result, setResult] = useState<EDIConnectionTestResult | null>(null);
  const transport = leadTransport(card);
  const internal = card.partner.kind === "Internal";
  const id = card.partner.id;

  const test = useApiMutation({
    resourceName: "EDI connection test",
    mutationFn: (profileId: string) => apiService.ediService.testProfileConnection(profileId),
    onSuccess: (outcome) => setResult(outcome),
  });
  const pickup = useApiMutation({
    resourceName: "EDI pickup",
    mutationFn: (profileId: string) => apiService.ediService.pollProfile(profileId),
    onSuccess: (outcome) => {
      void queryClient.invalidateQueries({ queryKey: edi.partnerCard(id).queryKey });
      void queryClient.invalidateQueries({ queryKey: edi.partnerMessages(id).queryKey });
      toast.success(
        outcome.stagedFileIds.length > 0
          ? t(
              "{0, plural, one {Picked up # file} other {Picked up # files}} from {1}",
              outcome.stagedFileIds.length,
              card.partner.name,
            )
          : t("Nothing new to pick up from {0}", card.partner.name),
      );
    },
  });
  const busy = test.isPending || pickup.isPending;

  const testBlocker = !transport
    ? t("No line is set up to test")
    : transport.profile.status !== "Active"
      ? t("The line is turned off")
      : null;
  const pickupBlocker = !transport
    ? t("No line is set up to pick up from")
    : !transportPolls(transport)
      ? t("This line has no folder to pick up their files from")
      : null;
  const showPickup =
    transport !== null &&
    (transport.profile.method === "SFTP" || transport.profile.method === "VAN");

  return (
    <>
      {result && (
        <CardSection
          title={t("Connection test")}
          aside={
            <span className={result.success ? "dk-ok" : "dk-bad"}>
              {result.success ? t("Passed") : t("Failed")}
            </span>
          }
        >
          <CardChecks>
            {result.checks.map((check) => (
              <CardCheck
                key={check.name}
                state={
                  check.status === "passed" ? "ok" : check.status === "warning" ? "warn" : "bad"
                }
                title={check.name}
                detail={check.message ?? ""}
              />
            ))}
          </CardChecks>
        </CardSection>
      )}
      <CardBar>
        {canUpdate && !internal && showPickup && (
          <CardActionButton
            variant="quiet"
            blocker={pickupBlocker}
            busy={busy}
            onClick={() => transport && pickup.mutate(transport.profile.id)}
          >
            <RefreshCw01Icon className="size-3.5" aria-hidden />
            {pickup.isPending ? t("Picking up…") : t("Pick up now")}
          </CardActionButton>
        )}
        {canUpdate && !internal && (
          <CardActionButton
            variant="quiet"
            blocker={testBlocker}
            busy={busy}
            onClick={() => transport && test.mutate(transport.profile.id)}
          >
            <Link01Icon className="size-3.5" aria-hidden />
            {test.isPending ? t("Testing…") : t("Test connection")}
          </CardActionButton>
        )}
        <CardBarHint>
          {internal
            ? t("Suspend or revoke the connection on the partner's page")
            : card.readiness.ready
              ? t("Settings, documents and mappings are on the partner's page")
              : t("Finish setting them up on the partner's page")}
        </CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("edi_partner", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open partner")}
        </Link>
      </CardBar>
    </>
  );
}
