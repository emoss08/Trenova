import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  CardShipmentsSkeleton,
  CardTimelineSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { INVOICE_SECTIONS } from "./invoice-sections";

/** The shipments the card reads at a time; the next page is read on request. */
export const INVOICE_SHIPMENT_ROWS = 3;
/** The activity entries the card reads at a time. */
export const INVOICE_ACTIVITY_ROWS = 5;

const KEYS = 4;
const CHARGES = [44, 36, 30] as const;
const FOOT = [56, 40, 64] as const;
const STUBS = 2;
const CHANNELS = 3;
const ACTIVITY_ENTRIES = [64, 52, 58] as const;

/** What is owed while the invoice is read: the strip, its keys and the terms under them. */
function OwedSkeleton() {
  return (
    <div className="dk-inv-owed">
      <div className="dk-inv-strip dk-sk" />
      <dl className="dk-inv-keys">
        {Array.from({ length: KEYS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <div key={index}>
            <SkeletonBar w="60%" className="dk-rec-sk-label" />
            <SkeletonBar w="70%" className="dk-rec-sk-line" />
          </div>
        ))}
      </dl>
      <div className="dk-inv-terms">
        <div className="dk-inv-ruler dk-sk" />
        <div className="dk-inv-marks">
          <span className="dk-start">
            <SkeletonBar w="56px" className="dk-rec-sk-label" />
          </span>
          <span className="dk-end">
            <SkeletonBar w="72px" className="dk-rec-sk-label" />
          </span>
        </div>
      </div>
    </div>
  );
}

/** The charges while the invoice is read: a row for each charge, then the foot. */
function ChargesSkeleton() {
  return (
    <div className="dk-inv-bill">
      <table>
        <tbody>
          {CHARGES.map((width, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <tr key={index}>
              <td>
                <SkeletonBar w={`${width}%`} className="dk-rec-sk-line" />
              </td>
              <td className="dk-inv-basis">
                <SkeletonBar w="96px" className="dk-rec-sk-label" />
              </td>
              <td className="dk-inv-amt">
                <SkeletonBar w="64px" className="dk-rec-sk-line" />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <dl className="dk-inv-foot">
        {FOOT.map((width, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <div key={index} className={index === FOOT.length - 1 ? "dk-due" : "dk-line"}>
            <dt>
              <SkeletonBar w={`${width}px`} className="dk-rec-sk-label" />
            </dt>
            <dd>
              <SkeletonBar w="64px" className="dk-rec-sk-line" />
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

/** The payments while the invoice is read: a stub for each. */
function RemitSkeleton() {
  return (
    <ul className="dk-inv-remit">
      {Array.from({ length: STUBS }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <li key={index}>
          <div className="dk-inv-stub">
            <span className="dk-inv-mark dk-sk" />
            <span className="dk-inv-what">
              <SkeletonBar w="44%" className="dk-rec-sk-line" />
              <SkeletonBar w="30%" className="dk-rec-sk-label" />
            </span>
            <SkeletonBar w="64px" className="dk-rec-sk-line" />
          </div>
        </li>
      ))}
    </ul>
  );
}

/** Where the invoice went while it is read: a slip for each way out. */
function ChannelsSkeleton() {
  return (
    <div className="dk-inv-went">
      {Array.from({ length: CHANNELS }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} className="dk-inv-chan dk-idle">
          <SkeletonBar w="40%" className="dk-rec-sk-label" />
          <SkeletonBar w="56%" className="dk-rec-sk-line" />
          <SkeletonBar w="80%" className="dk-rec-sk-label" />
        </div>
      ))}
    </div>
  );
}

/**
 * The invoice card while its code or its invoice is read: the slip and the
 * bill-to over what it covers, what is owed and its terms, the charges and
 * their foot, what settled it, where it went, the shipments it bills and its
 * activity, each where the card draws it. A section the person may not read
 * is left out here as it is on the card.
 */
export function DeskInvoiceSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(INVOICE_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the invoice…")}>
      <CardHeadSkeleton
        kicker={[84, 40, 64]}
        stepper={stepper}
        lead={<i className="dk-sk dk-inv-slip" />}
        title={40}
        subtitle={56}
        figure
        meta={[96, 72]}
      />
      <CardSectionSkeleton title={80} aside={56}>
        <OwedSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={56} aside={48}>
        <ChargesSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={128} aside={56}>
        <RemitSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={80}>
        <ChannelsSkeleton />
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={104} aside={72}>
          <CardShipmentsSkeleton layout="route" rows={INVOICE_SHIPMENT_ROWS} />
        </CardSectionSkeleton>
      )}
      {access.activity && (
        <CardSectionSkeleton title={56} aside={64}>
          <CardTimelineSkeleton entries={ACTIVITY_ENTRIES} />
        </CardSectionSkeleton>
      )}
      <CardBarSkeleton actions={[110]} />
    </RecordCardSkeleton>
  );
}
