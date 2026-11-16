import { SectionPanel } from "@/components/section-panel";
import type { ManagedPlanMeter, ManagedPlanUsage } from "@/lib/graphql/managed-plan";
import { Progress } from "@trenova/shared/components/ui/progress";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { formatFileSize } from "@trenova/shared/lib/utils";

/** A metered allowance this far used is drawn amber, so it is noticed before it runs out. */
const NEAR_RATIO = 0.8;

const ORGANIZATIONS_METER = "organizations.total";

function formatAmount(meter: ManagedPlanMeter, value: number): string {
  return meter.unit === "BYTES" ? formatFileSize(value) : formatNumber(value);
}

/** Near or past a metered allowance; a hard cap reached by design stays neutral. */
function isNear(meter: ManagedPlanMeter): boolean {
  if (meter.hardCap || meter.included <= 0) {
    return false;
  }
  return meter.stopped || meter.used >= NEAR_RATIO * meter.included;
}

function MeterNote({ meter }: { meter: ManagedPlanMeter }) {
  const t = useT();

  if (meter.hardCap) {
    return (
      <p className="mt-1.5 text-2xs text-muted-foreground">
        {meter.key === ORGANIZATIONS_METER
          ? t(
              "{0, plural, one {Your plan includes one organization.} other {Your plan includes # organizations.}}",
              meter.included,
            )
          : t("This is a limit of your plan, never billed as overage.")}
      </p>
    );
  }
  if (meter.stopped) {
    return (
      <p className="mt-1.5 text-2xs text-muted-foreground">
        {t("Your plan's allowance is used up.")}
      </p>
    );
  }
  return null;
}

function UsageMeter({ meter }: { meter: ManagedPlanMeter }) {
  const rt = useRichT();

  return (
    <div className="min-w-0 border-t border-border-subtle p-3 first:border-t-0 sm:border-t-0 sm:border-l sm:first:border-l-0">
      <div className="flex items-baseline justify-between gap-2">
        <span className="truncate text-xs font-medium text-foreground">{meter.name}</span>
        <span className="shrink-0 text-xs text-muted-foreground tabular-nums">
          {rt(
            "<b>{0}</b> of {1}",
            { b: (children) => <span className="font-medium text-foreground">{children}</span> },
            formatAmount(meter, meter.used),
            formatAmount(meter, meter.included),
          )}
        </span>
      </div>
      <Progress
        className="mt-2"
        size="sm"
        variant={isNear(meter) ? "warning" : "default"}
        value={meter.used}
        max={meter.included > 0 ? meter.included : 1}
        aria-label={meter.name}
      />
      <MeterNote meter={meter} />
    </div>
  );
}

/** What the plan allows this billing period and how much of it is used. */
export function PlanUsage({ usage }: { usage: ManagedPlanUsage }) {
  const t = useT();

  if (usage.meters.length === 0) {
    return null;
  }

  return (
    <SectionPanel
      title={t("Usage")}
      hint={t(
        "{0} to {1}",
        formatUnixMonthDay(usage.periodStart),
        formatUnixMonthDay(usage.periodEnd),
      )}
    >
      <div className="grid grid-cols-1 sm:grid-cols-[repeat(auto-fit,minmax(200px,1fr))]">
        {usage.meters.map((meter) => (
          <UsageMeter key={meter.key} meter={meter} />
        ))}
      </div>
    </SectionPanel>
  );
}
