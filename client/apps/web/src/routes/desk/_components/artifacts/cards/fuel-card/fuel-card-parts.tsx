import {
  FuelCardNetworkLogo,
  fuelCardNetworkFill,
  fuelCardNetworkInk,
} from "@/components/fuel-card-mark";
import { recordPath } from "@/config/record-links";
import type {
  DeskFuelCardPurchase,
  DeskFuelCardRecord,
  DeskFuelHeldRow,
  DeskFuelJurisdictionGallons,
} from "@/lib/graphql/fuel-desk-cards";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  FUEL_CARD_PROVIDER_LABELS,
  IFTA_FUEL_TYPE_LABELS,
} from "@trenova/shared/types/fuel-ifta-enums";
import type { CSSProperties } from "react";
import { CardAvatar, CardIcon, CardRecordRow } from "../../record-card/record-card";
import { cardDay, cardTime, money } from "../../record-card/record-card-values";
import {
  expiryFace,
  formatGallons,
  heldPlace,
  heldReason,
  isUnassigned,
  jurisdictionShare,
  purchaseTax,
} from "./fuel-card-card";

/**
 * The card as it is printed: the network's colour and mark, the chip, the
 * number with all but its last four hidden, its label and when it expires,
 * and the driver it is issued to. A card nobody holds yet is drawn dashed
 * round, and a cancelled one greyed and stamped. A provider that wears no
 * brand is drawn in the Desk's own ink.
 */
export function FuelCardFace({ card }: { card: DeskFuelCardRecord }) {
  const t = useT();
  const fill = fuelCardNetworkFill(card.provider);
  const ink = fuelCardNetworkInk(card.provider);
  const provider = FUEL_CARD_PROVIDER_LABELS[card.provider];
  const cancelled = card.status === "Cancelled";
  const holder = card.assignedWorker?.wholeName?.trim() ?? "";
  const style = fill ? ({ "--fc-fill": fill } as CSSProperties) : undefined;

  return (
    <div
      role="img"
      aria-label={t("{0} card ending {1}", provider, card.lastFour)}
      className={cn(
        "dk-fcd-face",
        fill ? (ink === "dark" ? "dk-ink-dark" : "dk-ink-light") : "dk-plain",
        isUnassigned(card) && !cancelled && "dk-pending",
        cancelled && "dk-cancelled",
      )}
      style={style}
    >
      <div className="dk-fcd-top">
        <span className="dk-fcd-brand">
          {fill ? <FuelCardNetworkLogo provider={card.provider} /> : <b>{provider}</b>}
        </span>
        <span className="dk-fcd-kind">{t("Fleet fuel")}</span>
      </div>
      <span className="dk-fcd-chip" aria-hidden>
        <i />
        <i />
        <i />
      </span>
      <span className="dk-fcd-num">
        <em>••••</em>
        <em>••••</em>
        <em>••••</em>
        <b>{card.lastFour}</b>
      </span>
      <div className="dk-fcd-foot">
        <span>
          <small>{t("Card")}</small>
          {card.label.trim() || t("Not labelled")}
        </span>
        <span>
          <small>{t("Expires")}</small>
          {expiryFace(card.expiresAt)}
        </span>
      </div>
      {holder && !cancelled && <span className="dk-fcd-holder">{holder}</span>}
      {cancelled && <span className="dk-fcd-stamp">{t("Cancelled")}</span>}
    </div>
  );
}

/** The driver and the tractor that carry the card, each opening in place. */
export function FuelCardHolders({
  card,
  onOpenWorker,
  onOpenTractor,
}: {
  card: DeskFuelCardRecord;
  onOpenWorker?: (id: string) => void;
  onOpenTractor?: (id: string) => void;
}) {
  const t = useT();
  const worker = card.assignedWorker;
  const tractor = card.assignedTractor;

  if (!worker && !tractor) {
    return (
      <div className="dk-fcd-holds">
        <span className="dk-fcd-cap">{t("Who holds it")}</span>
        <div className="dk-fcd-nobody">
          <span className="dk-fcd-ni" aria-hidden>
            <CardIcon name="info" size={11} stroke={2.4} />
          </span>
          <span>
            <b>{t("No one yet")}</b>
            <span>
              {card.discoveredAt
                ? t(
                    "A card the feed hasn't seen before is held as suspended until someone assigns it to a driver or a tractor.",
                  )
                : t("Purchases on it are matched by the unit on the receipt until it is assigned.")}
            </span>
          </span>
        </div>
      </div>
    );
  }

  const model = tractor ? [tractor.make, tractor.model].filter(Boolean).join(" ") : "";
  return (
    <div className="dk-fcd-holds">
      <span className="dk-fcd-cap">{t("Who holds it")}</span>
      {worker && (
        <CardRecordRow
          className="dk-fu-seat"
          kind="worker" to={recordPath("worker", worker.id)}
          onOpen={onOpenWorker && (() => onOpenWorker(worker.id))}
        >
          <CardAvatar name={worker.wholeName} />
          <span className="dk-rec-main">
            <b>{worker.wholeName}</b>
            <span className="dk-rec-sub">{t("Driver")}</span>
          </span>
          <CardIcon name="forward" size={12} />
        </CardRecordRow>
      )}
      {tractor && (
        <CardRecordRow
          className="dk-fu-seat"
          kind="tractor" to={recordPath("tractor", tractor.id)}
          onOpen={onOpenTractor && (() => onOpenTractor(tractor.id))}
        >
          <span className="dk-fu-unit" aria-hidden>
            {tractor.code}
          </span>
          <span className="dk-rec-main">
            <b>{tractor.code}</b>
            <span className="dk-rec-sub">{model ? t("{0} · Tractor", model) : t("Tractor")}</span>
          </span>
          <CardIcon name="forward" size={12} />
        </CardRecordRow>
      )}
    </div>
  );
}

function PurchaseWhen({ at }: { at: number | null }) {
  return (
    <span className="dk-rec-board-at">
      <b>{cardTime(at)}</b>
      <span>{cardDay(at)}</span>
    </span>
  );
}

/** The card's latest purchases as rows, newest first, each opening in place. */
export function FuelCardPurchaseRows({
  rows,
  onOpen,
}: {
  rows: readonly DeskFuelCardPurchase[];
  onOpen?: (id: string) => void;
}) {
  const t = useT();
  return (
    <div className="dk-rec-board">
      {rows.map((row) => {
        const tax = purchaseTax(row, t);
        const city = row.vendorCity?.trim() ?? "";
        const fuel = IFTA_FUEL_TYPE_LABELS[row.fuelType];
        return (
          <CardRecordRow
            key={row.id}
            className="dk-fcd-buy"
            to={recordPath("fuel_purchase", row.id)}
            onOpen={onOpen && (() => onOpen(row.id))}
          >
            <PurchaseWhen at={row.purchasedAt} />
            <span className="dk-rec-main">
              <b>{row.vendor?.trim() || t("Fuel purchase")}</b>
              <span className="dk-rec-sub">{city ? t("{0} · {1}", city, fuel) : fuel}</span>
            </span>
            <span className="dk-fcd-juris">{row.jurisdiction.code}</span>
            <span className="dk-fcd-gal">{formatGallons(row.gallons, t)}</span>
            <span className="dk-fcd-amt">{money(row.totalAmount, row.currencyCode)}</span>
            <Badge variant={tax.tone}>{tax.label}</Badge>
          </CardRecordRow>
        );
      })}
    </div>
  );
}

/**
 * The statement rows waiting on the card, drawn dashed because they are not
 * purchases yet: what each says it bought and where, and why it is waiting.
 */
export function FuelCardHeldRows({ rows }: { rows: readonly DeskFuelHeldRow[] }) {
  const t = useT();
  return (
    <div className="dk-rec-board">
      {rows.map((row) => {
        const parsed = row.parsed;
        const place = heldPlace(row, t);
        const reason = heldReason(row, t);
        return (
          <div key={row.id} className="dk-rec-row dk-fcd-buy dk-held">
            <PurchaseWhen at={parsed?.purchasedAt ?? null} />
            <span className="dk-rec-main">
              <b>{parsed?.vendor?.trim() || t("Row {0}", row.rowNumber)}</b>
              <span className="dk-rec-sub">{place ? t("{0} · {1}", place, reason) : reason}</span>
            </span>
            <span className="dk-fcd-juris">{parsed?.jurisdictionCode ?? ""}</span>
            <span className="dk-fcd-gal">
              {parsed?.gallons ? formatGallons(parsed.gallons, t) : ""}
            </span>
            <span className="dk-fcd-amt">
              {parsed?.totalAmount
                ? money(parsed.totalAmount, parsed.currencyCode ?? undefined)
                : ""}
            </span>
            <Badge variant="neutral">{t("Waiting")}</Badge>
          </div>
        );
      })}
    </div>
  );
}

/** The quarter's IFTA fuel on the card by jurisdiction, each as a bar against the largest. */
export function FuelJurisdictionBars({ rows }: { rows: readonly DeskFuelJurisdictionGallons[] }) {
  const t = useT();
  return (
    <ol className="dk-fcd-bars">
      {rows.map((row) => (
        <li key={row.jurisdictionId}>
          <span className="dk-fpu-code">{row.jurisdiction.code}</span>
          <span className="dk-fcd-bn">{row.jurisdiction.name}</span>
          <span className="dk-fcd-track" aria-hidden>
            <i style={{ width: `${jurisdictionShare(row, rows)}%` }} />
          </span>
          <b>{formatGallons(row.gallons, t)}</b>
        </li>
      ))}
    </ol>
  );
}
