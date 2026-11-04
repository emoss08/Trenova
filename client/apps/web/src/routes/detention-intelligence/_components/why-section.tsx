import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import { SectionPanel } from "@/components/section-panel";
import type {
  DetentionFacility,
  DetentionIntelligence,
} from "@/lib/graphql/detention-intelligence";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import {
  CAUSES,
  CAUSE_LABELS,
  CAUSE_LEVER,
  CAUSE_MEANING,
  causeMinutes,
  hours,
  shareOf,
  type CauseKey,
} from "./intelligence-model";
import { FacilityHoverCard, useFacilityMark } from "./facility-hover";
import { Finding, IntelligenceSection, percent } from "./intelligence-parts";

const strong = (chunk: React.ReactNode) => (
  <span className="text-foreground font-medium">{chunk}</span>
);

type CauseSlice = { key: CauseKey; minutes: number; total: number; bgClass: string };

function CauseSliceDetail({ slice }: { slice: CauseSlice }) {
  const t = useT();
  return (
    <>
      <HoverCardHeading>{CAUSE_LABELS[slice.key]}</HoverCardHeading>
      <HoverCardRow
        swatchClass={slice.bgClass}
        label={t("{0} h", hours(slice.minutes).toLocaleString())}
        value={percent(shareOf(slice.minutes, slice.total))}
      />
    </>
  );
}

function CauseSegment({ slice }: { slice: CauseSlice }) {
  const hover = usePointerHoverTarget(slice, slice.key);
  return (
    <span
      aria-label={CAUSE_LABELS[slice.key]}
      className={cn("block h-full", slice.bgClass)}
      style={{ flexGrow: slice.minutes }}
      {...hover}
    />
  );
}

function CauseCard({ totals }: { totals: DetentionIntelligence["totals"] }) {
  const t = useT();
  const rt = useRichT();
  const total = totals.pastFreeMinutes;
  const ours = totals.earlyMinutes + totals.lateMinutes;

  return (
    <SectionPanel
      title={t("Who caused the wait")}
      description={t("Every hour past free time, by whether the truck kept its appointment.")}
    >
      <div className="p-3">
        <Finding>
          {rt(
            "On <b>{0}</b> of the hours the truck was on time and the dock was slow. <b>{1}</b> is ours to fix: early arrivals waiting for their slot, and late ones that forfeited free time.",
            { b: strong },
            percent(shareOf(totals.dockMinutes, total)),
            percent(shareOf(ours, total)),
          )}
        </Finding>
        <PointerHoverCard<CauseSlice>
          follow="pointer"
          itemId={(slice) => slice.key}
          render={(slice) => <CauseSliceDetail slice={slice} />}
        >
          <div className="flex h-3 gap-0.5 overflow-hidden rounded-sm">
            {CAUSES.map((cause) => {
              const minutes = causeMinutes(totals, cause.key);
              return minutes > 0 ? (
                <CauseSegment
                  key={cause.key}
                  slice={{ key: cause.key, minutes, total, bgClass: cause.bgClass }}
                />
              ) : null;
            })}
          </div>
        </PointerHoverCard>
        <ul className="mt-2 divide-y">
          {CAUSES.map((cause) => {
            const minutes = causeMinutes(totals, cause.key);
            return (
              <li
                key={cause.key}
                className="grid grid-cols-[10px_minmax(0,1fr)_auto_3rem] items-start gap-3 py-2.5"
              >
                <span className={cn("mt-1 size-2.5 rounded-sm", cause.bgClass)} />
                <div className="min-w-0">
                  <p className="text-sm font-medium">{CAUSE_LABELS[cause.key]}</p>
                  <p className="text-muted-foreground mt-0.5 text-xs">
                    {CAUSE_MEANING[cause.key]}{" "}
                    <span className="text-foreground">{CAUSE_LEVER[cause.key]}</span>
                  </p>
                </div>
                <span className="font-mono text-sm tabular-nums">
                  {t("{0} h", hours(minutes).toLocaleString())}
                </span>
                <span className="text-muted-foreground text-right text-xs tabular-nums">
                  {percent(shareOf(minutes, total))}
                </span>
              </li>
            );
          })}
        </ul>
      </div>
    </SectionPanel>
  );
}

/** The scale the free-time rows share: nine hours, which holds all but the worst tails. */
const FREE_TIME_SCALE = 9 * 60;
const FREE_TIME_TICKS = [0, 120, 240, 360, 480];
const FREE_TIME_ROWS = 9;
/** The stretch a median stop runs past its free time: the danger hue, faded into the card. */
const OVERRUN_BACKGROUND = "color-mix(in oklch, var(--danger) 32%, transparent)";

function position(minutes: number): string {
  return `${(Math.min(minutes, FREE_TIME_SCALE) / FREE_TIME_SCALE) * 100}%`;
}

function FreeTimeRow({ facility, onOpen }: { facility: DetentionFacility; onOpen: () => void }) {
  const t = useT();
  const { attributes, handlers } = useFacilityMark(facility);
  const over = facility.medianDwellMinutes > facility.medianFreeMinutes;
  const gap = Math.round(Math.abs(facility.medianDwellMinutes - facility.medianFreeMinutes));

  return (
    <button
      type="button"
      onClick={onOpen}
      className={cn(
        "ui-focus-ring hover:bg-surface-hover border-border-subtle grid min-h-8 w-full grid-cols-[minmax(5.25rem,6.875rem)_minmax(0,1fr)_4rem] items-center gap-3 border-t text-left transition-opacity",
      )}
      aria-label={t(
        "{0}: {1} free time, {2} median stop, {3} p90",
        facility.locationName,
        formatDetentionMinutes(Math.round(facility.medianFreeMinutes)),
        formatDetentionMinutes(Math.round(facility.medianDwellMinutes)),
        formatDetentionMinutes(Math.round(facility.p90DwellMinutes)),
      )}
      {...attributes}
      {...handlers}
    >
      <span className="truncate text-xs">{facility.locationName}</span>
      <span className="relative h-8">
        {FREE_TIME_TICKS.map((tick) => (
          <span
            key={tick}
            className="bg-border-subtle absolute inset-y-0 w-px"
            style={{ left: position(tick) }}
          />
        ))}
        {over ? (
          <span
            className="absolute top-1/2 h-1.5 -translate-y-1/2 rounded-full"
            style={{
              left: position(facility.medianFreeMinutes),
              width: `calc(${position(facility.medianDwellMinutes)} - ${position(facility.medianFreeMinutes)})`,
              background: OVERRUN_BACKGROUND,
            }}
          />
        ) : null}
        <span
          className="bg-border-strong absolute top-1/2 h-0.5 -translate-y-1/2"
          style={{
            left: position(facility.medianDwellMinutes),
            width: `calc(${position(facility.p90DwellMinutes)} - ${position(facility.medianDwellMinutes)})`,
          }}
        />
        <span
          className="bg-muted-foreground absolute top-1/2 h-2.5 w-px -translate-y-1/2"
          style={{ left: position(facility.p90DwellMinutes) }}
        />
        <span
          className="bg-foreground absolute inset-y-2 w-0.5 -translate-x-1/2 rounded-xs"
          style={{ left: position(facility.medianFreeMinutes) }}
        />
        <span
          className={cn(
            "ring-card absolute top-1/2 size-2.25 -translate-x-1/2 -translate-y-1/2 rounded-full ring-2",
            over ? "bg-danger" : "bg-foreground",
          )}
          style={{ left: position(facility.medianDwellMinutes) }}
        />
      </span>
      <span
        className={cn(
          "text-right font-mono text-xs font-medium tabular-nums",
          gap === 0
            ? "text-muted-foreground"
            : over
              ? "text-danger-foreground"
              : "text-success-foreground",
        )}
      >
        {gap === 0
          ? formatDetentionMinutes(0)
          : over
            ? t("+{0}", formatDetentionMinutes(gap))
            : t("−{0}", formatDetentionMinutes(gap))}
      </span>
    </button>
  );
}

/** A facility's free time against its stop, as the mockup's free-time tooltip reads. */
function FreeTimeDetail({ facility }: { facility: DetentionFacility }) {
  const t = useT();
  return (
    <>
      <HoverCardHeading>{facility.locationName}</HoverCardHeading>
      <HoverCardRow
        label={t(
          "Free time {0} · p90 {1}",
          formatDetentionMinutes(Math.round(facility.medianFreeMinutes)),
          formatDetentionMinutes(Math.round(facility.p90DwellMinutes)),
        )}
        value={t("median {0}", formatDetentionMinutes(Math.round(facility.medianDwellMinutes)))}
      />
    </>
  );
}

function FreeTimeCard({
  facilities,
  onOpen,
}: {
  facilities: readonly DetentionFacility[];
  onOpen: (locationId: string) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const over = facilities.filter((f) => f.medianDwellMinutes > f.medianFreeMinutes);
  const worst = [...over].sort(
    (a, b) =>
      b.medianDwellMinutes - b.medianFreeMinutes - (a.medianDwellMinutes - a.medianFreeMinutes),
  )[0];
  const rows = [...facilities]
    .sort(
      (a, b) =>
        b.medianDwellMinutes - b.medianFreeMinutes - (a.medianDwellMinutes - a.medianFreeMinutes),
    )
    .slice(0, FREE_TIME_ROWS);

  return (
    <SectionPanel
      title={t("Free time against how long the dock takes")}
      description={t(
        "Where the median stop runs past the free time granted, the stop is detained before anyone is late.",
      )}
    >
      <div className="p-3">
        <Finding>
          {worst
            ? rt(
                "At <b>{0, plural, one {# facility} other {# facilities}}</b> the median stop takes longer than its free time, so detention is built into the schedule. <b>{1}</b> grants {2} for a {3} stop.",
                { b: strong },
                over.length,
                worst.locationName,
                formatDetentionMinutes(Math.round(worst.medianFreeMinutes)),
                formatDetentionMinutes(Math.round(worst.medianDwellMinutes)),
              )
            : t("Every facility's median stop finishes inside its free time.")}
        </Finding>
        <div className="text-foreground-subtle text-2xs grid h-4.5 grid-cols-[minmax(5.25rem,6.875rem)_minmax(0,1fr)_4rem] items-center gap-3">
          <span />
          <span className="relative h-full">
            {FREE_TIME_TICKS.map((tick) => (
              <span
                key={tick}
                className="absolute top-1/2 -translate-x-1/2 -translate-y-1/2 font-mono"
                style={{ left: position(tick) }}
              >
                {t("{0}h", tick / 60)}
              </span>
            ))}
          </span>
          <span className="text-right">{t("Median vs free")}</span>
        </div>
        <FacilityHoverCard render={(facility) => <FreeTimeDetail facility={facility} />}>
          {rows.map((facility) => (
            <FreeTimeRow
              key={facility.locationId}
              facility={facility}
              onOpen={() => onOpen(facility.locationId)}
            />
          ))}
        </FacilityHoverCard>
        <div className="text-muted-foreground text-2xs mt-2.5 flex flex-wrap items-center gap-x-3.5 gap-y-1">
          <span className="inline-flex items-center gap-1.5">
            <span aria-hidden className="bg-foreground h-3 w-0.5 rounded-xs" />
            {t("Free time")}
          </span>
          <span className="inline-flex items-center gap-1.5">
            <span aria-hidden className="bg-foreground size-2.25 rounded-full" />
            {t("Median stop")}
          </span>
          <span className="inline-flex items-center gap-1.5">
            <span aria-hidden className="bg-border-strong h-0.5 w-3.5" />
            {t("Out to p90")}
          </span>
        </div>
      </div>
    </SectionPanel>
  );
}

export function WhySection({
  data,
  onOpenFacility,
}: {
  data: DetentionIntelligence;
  onOpenFacility: (locationId: string) => void;
}) {
  const t = useT();

  return (
    <IntelligenceSection
      id="why"
      title={t("Why trucks wait")}
      answer={t(
        "The dock's wait and ours need different fixes, and free time shorter than the dock's real pace guarantees detention whatever anyone does.",
      )}
    >
      <div className="grid gap-3 xl:grid-cols-2">
        <CauseCard totals={data.totals} />
        <FreeTimeCard facilities={data.facilities} onOpen={onOpenFacility} />
      </div>
    </IntelligenceSection>
  );
}
