import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardStatsSkeleton,
  RecordCardSkeleton,
} from "../../skeletons/record-card-skeleton";

const CHECK_ROWS = [
  [36, 72],
  [44, 64],
] as const;
const LINE_ROWS = [
  [28, 44],
  [24, 40],
  [32, 44],
  [26, 42],
] as const;

/**
 * The IFTA return card while its code or its return is read: the quarter and
 * what it comes to over where it stands, what stands before filing, the
 * quarter's figures and the jurisdiction lines, each where the card draws it.
 */
export function DeskIftaReturnSkeleton() {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the IFTA return…")}>
      <CardHeadSkeleton kicker={[72, 48]} title={64} figure meta={[48, 112]} />
      <CardSectionSkeleton title={112}>
        <CardChecksSkeleton rows={CHECK_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={72}>
        <CardStatsSkeleton cells={4} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={136}>
        <CardRowsSkeleton rows={LINE_ROWS} trailing={72} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[132]} />
    </RecordCardSkeleton>
  );
}
