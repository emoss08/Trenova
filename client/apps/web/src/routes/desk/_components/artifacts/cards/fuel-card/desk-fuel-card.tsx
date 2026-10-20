import { recordPath } from "@/config/record-links";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { usePermission } from "@/hooks/use-permission";
import { describeApiError } from "@/lib/api-error-message";
import { FUEL_CARD_LIST_KEY, UNASSIGNED_FUEL_CARD_LIST_KEY } from "@/lib/graphql/fuel-card";
import type { DeskFuelCardCard } from "@/lib/graphql/fuel-desk-cards";
import { FUEL_PURCHASE_LIST_KEY } from "@/lib/graphql/fuel-purchase";
import {
  FUEL_PURCHASE_IMPORT_KEY,
  FUEL_PURCHASE_IMPORT_ROWS_KEY,
  resolveFuelPurchaseImportRows,
} from "@/lib/graphql/fuel-purchase-import";
import { fuelCard } from "@/lib/queries/fuel-card";
import { fuelPurchase } from "@/lib/queries/fuel-purchase";
import { AssignFuelCardDialog } from "@/routes/fuel-card/_components/assign-fuel-card-dialog";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { FUEL_CARD_PROVIDER_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { Link } from "react-router";
import { CardActionButton } from "../../record-card/card-action";
import { FuelCardLead, FuelCardRow } from "../../record-card/card-fuel";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFactGrid,
  CardFacts,
  CardHead,
  CardIcon,
  CardSection,
  CardSectionError,
  CardStats,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
  type CardFact,
} from "../../record-card/record-card";
import { cardDate, cardMonth, money, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  barHint,
  cardFacts,
  cardFigure,
  cardKicker,
  cardStanding,
  cardSubtitle,
  cardTitle,
  feedCheck,
  formatGallons,
  isUnassigned,
  monthTotals,
  postHeldRows,
  recentNote,
  waitingNote,
} from "./fuel-card-card";
import {
  FuelCardFace,
  FuelCardHeldRows,
  FuelCardHolders,
  FuelCardPurchaseRows,
  FuelJurisdictionBars,
} from "./fuel-card-parts";
import { DeskFuelCardSkeleton } from "./fuel-card-skeleton";

/**
 * What the card is given: the fuel card, and when it was opened from a table
 * or from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type FuelCardCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the card was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * A fuel card as the person who runs the fleet's fuel reads it, opened from a
 * table's row or looked up by the agent: the card as it is printed, who
 * carries it, what it spent this month and where it fuelled this quarter for
 * IFTA, its latest purchases, the statement rows waiting on it, and whether
 * the provider's feed is reading.
 *
 * A card the feed found that nobody holds yet is assigned here, with the
 * fuel cards page's own dialog: assigning it activates it, and the rows it
 * held are then worked again so they post, for someone who may work imports.
 * Suspending and cancelling stay on the fuel cards page.
 */
export function DeskFuelCard({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: FuelCardCardProps) {
  const t = useT();
  const query = useQuery({ ...fuelCard.card(recordId), retry: false });
  const card = query.data;

  if (!card) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Fuel card")}
        title={t("This fuel card could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskFuelCardSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <FuelCardCard
      key={card.card.id}
      card={card}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The fuel card as an artifact of its own, when the agent looked it up: the
 * same card, with nothing to step through and nowhere to go back to.
 */
export function DeskFuelCardArtifact({ recordId }: RecordArtifactProps) {
  return <DeskFuelCard recordId={recordId} />;
}

function FuelCardCard({
  card,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<FuelCardCardProps, "order" | "recordId"> & {
  card: DeskFuelCardCard;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: mayUpdate } = usePermission(Resource.FuelCard, Operation.Update);
  const { allowed: mayResolve } = usePermission(Resource.FuelPurchaseImport, Operation.Import);
  const [assigning, setAssigning] = useState(false);
  const [posting, setPosting] = useState(false);
  const [refusals, setRefusals] = useState<string[]>([]);
  const now = useNowSeconds();
  const record = card.card;
  const id = record.id;
  const title = cardTitle(record, t);
  const backToCard = t("Back to {0}", title);

  const listedPurchases = useListedRecords({
    entity: "fuel_purchase",
    order: card.recent.map((row) => row.id),
    backLabel: backToCard,
  });
  const listedWorker = useListedRecords({
    entity: "worker",
    order: record.assignedWorkerId ? [record.assignedWorkerId] : NO_ROWS,
    backLabel: backToCard,
  });
  const listedTractor = useListedRecords({
    entity: "tractor",
    order: record.assignedTractorId ? [record.assignedTractorId] : NO_ROWS,
    backLabel: backToCard,
  });
  const listedSuccessor = useListedRecords({
    entity: "fuel_card",
    order: card.successors.map((row) => row.id),
    backLabel: backToCard,
  });
  const opened =
    listedPurchases.opened ?? listedWorker.opened ?? listedTractor.opened ?? listedSuccessor.opened;
  if (opened) {
    return opened;
  }

  const cancelled = record.status === "Cancelled";
  const mayAssign = mayUpdate && !cancelled;
  const standing = cardStanding(record, t);
  const figure = cardFigure(card, t);
  const totals = monthTotals(card.month);
  const feed = feedCheck(card, t);
  const waiting = waitingNote(card, mayResolve, t);
  const note = recentNote(card, t);
  const unassigned = isUnassigned(record);
  const leadCurrency = card.month[0]?.currencyCode ?? "USD";

  const invalidate = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: fuelCard._def }),
      queryClient.invalidateQueries({ queryKey: fuelPurchase._def }),
      queryClient.invalidateQueries({ queryKey: [FUEL_CARD_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: [UNASSIGNED_FUEL_CARD_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: [FUEL_PURCHASE_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: [FUEL_PURCHASE_IMPORT_KEY] }),
      queryClient.invalidateQueries({ queryKey: [FUEL_PURCHASE_IMPORT_ROWS_KEY] }),
    ]);
  };
  const onAssigned = async () => {
    setRefusals([]);
    if (mayResolve && card.heldBatches.length > 0) {
      setPosting(true);
      try {
        setRefusals(
          await postHeldRows(card.heldBatches, resolveFuelPurchaseImportRows, (error) =>
            describeApiError(error, t("The import could not be worked again")),
          ),
        );
      } finally {
        setPosting(false);
      }
    }
    await invalidate();
  };

  const details: CardFact[] = [
    { key: "provider", label: t("Provider"), value: FUEL_CARD_PROVIDER_LABELS[record.provider] },
    {
      key: "number",
      label: t("Card number"),
      value: (
        <>
          {t("•••• {0}", record.lastFour)}
          <small className="dk-fu-dim">{t("the full number is never stored")}</small>
        </>
      ),
    },
    ...(record.externalCardId?.trim()
      ? [
          {
            key: "external",
            label: t("Provider's card ID"),
            value: record.externalCardId.trim(),
          },
        ]
      : []),
    { key: "expires", label: t("Expires"), value: cardMonth(record.expiresAt) || "—" },
    {
      key: "added",
      label: record.discoveredAt ? t("Found") : t("Added"),
      value: cardDate(record.discoveredAt ?? record.createdAt),
    },
    ...(record.notes?.trim()
      ? [{ key: "notes", label: t("Notes"), value: record.notes.trim() }]
      : []),
  ];

  return (
    <RecordCard className="dk-fcd">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={cardKicker(record, t)}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighborsIn(rowOrder, id)}
              onStep={onSelect}
              previousLabel={t("Previous card")}
              nextLabel={t("Next card")}
            />
          )
        }
        lead={<FuelCardLead provider={record.provider} cancelled={cancelled} />}
        title={title}
        subtitle={cardSubtitle(record, now, t)}
        figure={
          figure && (
            <>
              <b className={figure.warn ? "dk-warn" : undefined}>{figure.amount}</b>
              <span>{figure.caption}</span>
            </>
          )
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>{cardFacts(card, t)}</CardFacts>
          </>
        }
      />

      <section className="dk-rec-sec dk-fcd-hero">
        <FuelCardFace card={record} />
        <div className="dk-fcd-side">
          {cancelled ? (
            <div className="dk-fcd-void">
              <span className="dk-fcd-cap">{t("Why it was cancelled")}</span>
              <blockquote>{record.cancelReason?.trim() || t("No reason was given.")}</blockquote>
              {record.cancelledAt && (
                <span className="dk-fcd-by">
                  {t("Cancelled {0}", cardDate(record.cancelledAt))}
                </span>
              )}
            </div>
          ) : (
            <FuelCardHolders
              card={record}
              onOpenWorker={listedWorker.open}
              onOpenTractor={listedTractor.open}
            />
          )}
          {waiting && (
            <div className="dk-fcd-wait">
              <b>{waiting.title}</b>
              <span>{waiting.detail}</span>
            </div>
          )}
          {!cancelled && !unassigned && card.month.length > 0 && (
            <>
              <CardStats
                stats={[
                  { key: "purchases", label: t("Purchases"), value: totals.purchases },
                  {
                    key: "gallons",
                    label: t("Gallons"),
                    value: totals.gallons.toLocaleString(undefined, {
                      maximumFractionDigits: 1,
                    }),
                  },
                  {
                    key: "average",
                    label: t("Avg / gal"),
                    value:
                      totals.averagePerGallon === null
                        ? "—"
                        : money(totals.averagePerGallon, leadCurrency),
                  },
                  {
                    key: "untaxed",
                    label: t("Untaxed"),
                    value: formatGallons(String(totals.untaxedGallons), t),
                    state: totals.untaxedGallons > 0 ? "info" : undefined,
                  },
                ]}
              />
              <span className="dk-fcd-scope">
                {t("{0} so far · Q{1} {2}", cardMonth(card.monthStart), card.quarter, card.year)}
              </span>
            </>
          )}
          {cancelled &&
            card.successors.map((successor) => (
              <FuelCardRow
                key={successor.id}
                card={successor}
                onOpen={listedSuccessor.open && (() => listedSuccessor.open?.(successor.id))}
              />
            ))}
        </div>
      </section>

      {(card.heldCount > 0 || refusals.length > 0) && (
        <CardSection
          title={t("Waiting on this card")}
          aside={
            <span className="dk-warn-t">
              {card.heldSpend.length > 0
                ? t(
                    "{0, plural, one {# purchase} other {# purchases}} · {1}",
                    card.heldCount,
                    money(card.heldSpend[0]?.totalAmount, card.heldSpend[0]?.currencyCode),
                  )
                : t("{0, plural, one {# purchase} other {# purchases}}", card.heldCount)}
            </span>
          }
        >
          {refusals.map((refusal) => (
            <CardSectionError key={refusal}>
              {t("The card was assigned, but an import holding its rows refused: {0}", refusal)}
            </CardSectionError>
          ))}
          <FuelCardHeldRows rows={card.held} />
          {card.heldCount > card.held.length && (
            <p className="dk-fcd-bnote">
              {t(
                "{0} of {1} shown. The rest are on the fuel imports page.",
                card.held.length,
                card.heldCount,
              )}
            </p>
          )}
        </CardSection>
      )}

      {card.recent.length > 0 && (
        <CardSection
          title={cancelled ? t("Last purchases") : t("Recent purchases")}
          aside={
            <span>
              {cancelled
                ? t("Before it was cancelled")
                : t("{0, plural, one {# in all} other {# in all}}", card.purchaseCount)}
            </span>
          }
        >
          <FuelCardPurchaseRows rows={card.recent} onOpen={listedPurchases.open} />
          {note && <p className="dk-fcd-bnote">{note}</p>}
        </CardSection>
      )}

      {card.quarterByJurisdiction.length > 0 && (
        <CardSection
          title={t("Where it fuels")}
          aside={<span>{t("Q{0} {1} · for IFTA", card.quarter, card.year)}</span>}
        >
          <FuelJurisdictionBars rows={card.quarterByJurisdiction} />
          <p className="dk-fcd-bnote">
            {t("IFTA fuel only. DEF, reefer fuel and other spend are left out.")}
          </p>
        </CardSection>
      )}

      {feed && (
        <CardSection title={t("The feed")}>
          <CardChecks>
            <CardCheck state={feed.state} title={feed.title} detail={feed.detail}>
              {feed.aside && <span className="dk-fcd-by">{feed.aside}</span>}
            </CardCheck>
          </CardChecks>
        </CardSection>
      )}

      <CardSection title={t("Details")}>
        <CardFactGrid facts={details} />
      </CardSection>

      <CardBar>
        <CardBarHint>{barHint(card, mayAssign, t)}</CardBarHint>
        <Link
          className={mayAssign && unassigned ? "dk-ax-btn" : "dk-ax-btn dk-ink"}
          to={recordPath("fuel_card", id)}
        >
          <CardIcon name="ext" size={13} />
          {t("Open card")}
        </Link>
        {mayAssign && unassigned && (
          <CardActionButton blocker={null} busy={posting} onClick={() => setAssigning(true)}>
            {posting ? t("Posting what's waiting…") : t("Assign card")}
          </CardActionButton>
        )}
      </CardBar>

      {mayAssign && (
        <AssignFuelCardDialog
          open={assigning}
          onOpenChange={setAssigning}
          card={record}
          onAssigned={onAssigned}
        />
      )}
    </RecordCard>
  );
}
