import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { recordPath } from "@/config/record-links";
import type { DeskFuelPurchaseCard } from "@/lib/graphql/fuel-desk-cards";
import { fuelPurchase } from "@/lib/queries/fuel-purchase";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link } from "react-router";
import { CardDocuments } from "../../record-card/card-documents";
import { FuelCardRow } from "../../record-card/card-fuel";
import { CardPath, type PathStep } from "../../record-card/card-path";
import { SettlementStub } from "../../record-card/card-settlement";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardIcon,
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  needsLook,
  RecordCard,
} from "../../record-card/record-card";
import { cardMoment, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  figureCaption,
  filedSteps,
  fuelMark,
  purchaseChecks,
  purchaseFacts,
  purchaseKicker,
  purchaseStanding,
  purchaseSubtitle,
  purchaseTitle,
  totalDifference,
  totalText,
} from "./fuel-purchase-card";
import { FuelIftaPanel, FuelReceipt } from "./fuel-purchase-parts";
import { DeskFuelPurchaseSkeleton } from "./fuel-purchase-skeleton";

/**
 * What the card is given: the purchase, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type FuelPurchaseCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the purchase was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * A fuel purchase as the person who keeps the fuel tax return reads it, opened
 * from a table's row or looked up by the agent: the receipt as it was
 * recorded, what it does to the quarter's IFTA return in its jurisdiction and
 * the credit its tax-paid gallons earn, whether its total, odometer and card
 * hold up against the rest of the record, how it came to be on file, and the
 * card it was bought on, which opens in place.
 *
 * It changes nothing: a purchase is corrected or deleted on its own page.
 */
export function DeskFuelPurchase({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: FuelPurchaseCardProps) {
  const t = useT();
  const query = useQuery({ ...fuelPurchase.card(recordId), retry: false });
  const card = query.data;

  if (!card) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Fuel purchase")}
        title={t("This fuel purchase could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskFuelPurchaseSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <FuelPurchaseCard
      key={card.purchase.id}
      card={card}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The purchase as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskFuelPurchaseArtifact({ recordId }: RecordArtifactProps) {
  return <DeskFuelPurchase recordId={recordId} />;
}

function FuelPurchaseCard({
  card,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<FuelPurchaseCardProps, "order" | "recordId"> & {
  card: DeskFuelPurchaseCard;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const { purchase } = card;
  const id = purchase.id;
  const title = purchaseTitle(purchase, t);
  const canReadFuelCards = useCanReadKind("fuel_card");
  const listedCard = useListedRecords({
    entity: "fuel_card",
    order: purchase.fuelCard ? [purchase.fuelCard.id] : NO_ROWS,
    backLabel: t("Back to {0}", title),
  });
  if (listedCard.opened) {
    return listedCard.opened;
  }

  const checks = purchaseChecks(card, t, canReadFuelCards);
  const standing = purchaseStanding(card, checks, t);
  const off = checks.filter((check) => needsLook(check.state)).length;
  const difference = totalDifference(purchase);
  const steps: PathStep[] = filedSteps(card, t).map((step) => ({
    key: step.key,
    label: step.title,
    line: step.detail,
    state: "done",
  }));
  const filedAt = card.provenance?.batch.createdAt ?? purchase.createdAt;
  const fuelCard = purchase.fuelCard;

  return (
    <RecordCard className="dk-fpu">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={purchaseKicker(card, t)}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighborsIn(rowOrder, id)}
              onStep={onSelect}
              previousLabel={t("Previous purchase")}
              nextLabel={t("Next purchase")}
            />
          )
        }
        lead={<SettlementStub mark={fuelMark(purchase.fuelType)} />}
        title={title}
        subtitle={purchaseSubtitle(purchase, t)}
        figure={
          <>
            <b className={difference && difference.differenceMinor !== 0 ? "dk-warn" : undefined}>
              {totalText(purchase)}
            </b>
            <span>{figureCaption(purchase, t)}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>{purchaseFacts(card, t)}</CardFacts>
          </>
        }
      />

      <section className="dk-rec-sec dk-fpu-hero">
        <FuelReceipt card={card} />
        <FuelIftaPanel card={card} />
      </section>

      <CardSection
        title={t("Checks")}
        aside={
          off > 0 ? (
            <span className="dk-warn-t">
              {t("{0, plural, one {# to check} other {# to check}}", off)}
            </span>
          ) : (
            <span className="dk-ok">{t("Nothing to fix")}</span>
          )
        }
      >
        <CardChecks>
          {checks.map((check) => (
            <CardCheck
              key={check.key}
              state={check.state}
              title={check.title}
              detail={check.detail}
            />
          ))}
        </CardChecks>
      </CardSection>

      {steps.length > 0 && (
        <CardSection title={t("How it was filed")} aside={<span>{cardMoment(filedAt)}</span>}>
          <CardPath steps={steps} label={t("How the purchase was filed")} />
        </CardSection>
      )}

      {fuelCard && (
        <CardSection title={t("The card")}>
          <FuelCardRow
            card={fuelCard}
            onOpen={listedCard.open && (() => listedCard.open?.(fuelCard.id))}
          />
        </CardSection>
      )}

      <CardDocuments
        resourceType="fuel_purchase"
        resourceId={id}
        manageTo={recordPath("fuel_purchase", id)}
      />

      <CardBar>
        <CardBarHint>
          {off > 0
            ? t("Correct what's flagged on the purchase's page")
            : t("Correct or delete it on the purchase's page")}
        </CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("fuel_purchase", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open purchase")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}
