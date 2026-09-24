import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { concernSeverityMeta, workerStandingMeta } from "@trenova/shared/lib/worker-standing";
import { useQuery } from "@tanstack/react-query";
import { PALETTE_ENTITIES } from "../../palette-entities";
import type { PaletteAction, PaletteIntent, PaletteRecord } from "../../palette-model";
import { PreviewError, PreviewFrame, PreviewSection, PreviewSkeleton } from "./preview-frame";
import { workerPreviewQuery } from "./preview-queries";

const CONCERNS_SHOWN = 3;
const PTO_SHOWN = 3;

function Ratio({ done, total }: { done: number; total: number }) {
  return (
    <span className="tabular-nums">
      {done}
      <span className="text-foreground-subtle"> / {total}</span>
    </span>
  );
}

export function WorkerPreview({
  record,
  actions,
  onRun,
}: {
  record: PaletteRecord;
  actions: readonly PaletteAction[];
  onRun: (intent: PaletteIntent) => void;
}) {
  const t = useT();
  const { data: overview, isLoading, isError } = useQuery(workerPreviewQuery(record.id));
  const entity = PALETTE_ENTITIES.worker;

  if (isLoading) {
    return <PreviewSkeleton />;
  }
  if (isError || !overview) {
    return <PreviewError />;
  }

  const { worker, credentials, training, safety } = overview;
  const standing = workerStandingMeta(overview.standing);
  const name = `${worker.firstName} ${worker.lastName}`.trim();
  const concerns = overview.concerns.slice(0, CONCERNS_SHOWN);
  const pto = (overview.pto ?? []).filter((balance) => balance.tracked).slice(0, PTO_SHOWN);

  return (
    <PreviewFrame
      initials={getNameInitials(name)}
      tileClass={entity.tileClass}
      title={name}
      subtitle={[worker.type, worker.driverType, worker.fleetCode?.code]
        .filter(Boolean)
        .join(" · ")}
      badge={<Badge variant={standing.badgeVariant}>{t(standing.label)}</Badge>}
      actions={actions}
      onRun={onRun}
    >
      {concerns.length > 0 && (
        <PreviewSection title={t("Needs a look")}>
          <ul className="flex flex-col gap-2">
            {concerns.map((concern) => {
              const meta = concernSeverityMeta(concern.severity);
              return (
                <li key={concern.code} className="flex gap-2">
                  <span
                    aria-hidden
                    className={cn("mt-1.5 size-1.5 shrink-0 rounded-full", meta.dotClass)}
                  />
                  <span className="flex min-w-0 flex-col">
                    <span className="text-foreground text-sm">{concern.headline}</span>
                    {concern.detail && (
                      <span className="text-foreground-subtle text-xs">{concern.detail}</span>
                    )}
                  </span>
                </li>
              );
            })}
          </ul>
        </PreviewSection>
      )}
      <PreviewSection title={t("Readiness")}>
        <DescriptionList columns={2}>
          <DescriptionItem label={t("Status")}>
            <StatusBadge status={worker.status} />
          </DescriptionItem>
          <DescriptionItem label={t("Dispatch")}>
            <Badge variant={worker.canBeAssigned ? "success" : "warning"}>
              {worker.canBeAssigned ? t("Assignable") : t("Held")}
            </Badge>
          </DescriptionItem>
          <DescriptionItem label={t("Credentials")}>
            {credentials ? (
              <Ratio done={credentials.validCount} total={credentials.requiredCount} />
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Training")}>
            {training ? (
              <Ratio done={training.currentCount} total={training.requiredCount} />
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          {credentials && credentials.expiringCount + credentials.expiredCount > 0 && (
            <DescriptionItem label={t("Expiring or expired")} span="full">
              <span className="text-warning tabular-nums">
                {t(
                  "{0, plural, one {# credential} other {# credentials}}",
                  credentials.expiringCount + credentials.expiredCount,
                )}
              </span>
            </DescriptionItem>
          )}
        </DescriptionList>
      </PreviewSection>
      {safety && (
        <PreviewSection title={t("Safety")}>
          <DescriptionList columns={2}>
            <DescriptionItem label={t("Score")} numeric>
              {safety.score.toLocaleString(undefined, { maximumFractionDigits: 1 })}
            </DescriptionItem>
            <DescriptionItem label={t("Active points")} numeric>
              {safety.activePoints}
            </DescriptionItem>
            <DescriptionItem label={t("Open events")} numeric>
              {safety.openEvents}
            </DescriptionItem>
            <DescriptionItem label={t("Days since last event")} numeric>
              {safety.daysSinceLastEvent ?? <DescriptionEmpty />}
            </DescriptionItem>
          </DescriptionList>
        </PreviewSection>
      )}
      {pto.length > 0 && (
        <PreviewSection title={t("Time off available")}>
          <DescriptionList layout="split">
            {pto.map((balance) => (
              <DescriptionItem key={balance.ptoType} label={balance.ptoType} numeric>
                {t("{0, plural, one {# day} other {# days}}", Number(balance.availableDays))}
              </DescriptionItem>
            ))}
          </DescriptionList>
        </PreviewSection>
      )}
      {(overview.nextReviewAt || worker.profile?.hireDate) && (
        <PreviewSection title={t("Dates")}>
          <DescriptionList columns={2}>
            {worker.profile?.hireDate ? (
              <DescriptionItem label={t("Hired")}>
                {formatUnixDateMedium(worker.profile.hireDate)}
              </DescriptionItem>
            ) : null}
            {overview.nextReviewAt ? (
              <DescriptionItem label={t("Next review")}>
                {formatUnixDateMedium(overview.nextReviewAt)}
              </DescriptionItem>
            ) : null}
          </DescriptionList>
        </PreviewSection>
      )}
    </PreviewFrame>
  );
}
