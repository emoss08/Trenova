import { type TranslateFn, useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ReadinessCheck, ReadinessResponse } from "@trenova/shared/types/formula-template";
import { AlertTriangleIcon, CheckCircleIcon, XCircleIcon } from "@trenova/shared/components/icons";
import { TARIFF_REGION_LABELS } from "@/lib/tariff-coverage";
import { lineOf } from "@/lib/tariff-source-lines";
import type { TariffRange } from "@trenova/shared/types/tariff";

export type ReadinessStep = "submit" | "approve";

const STATUS_STYLES: Record<
  ReadinessCheck["status"],
  { icon: typeof CheckCircleIcon; className: string }
> = {
  pass: { icon: CheckCircleIcon, className: "text-success-foreground" },
  warn: { icon: AlertTriangleIcon, className: "text-warning-foreground" },
  fail: { icon: XCircleIcon, className: "text-destructive" },
};

/** Where a readiness row's regions are in the source, to name their lines and reveal them. */
export type ReadinessSourceLink = {
  source: string;
  onReveal: (range: TariffRange) => void;
};

function CheckRow({ check, link }: { check: ReadinessCheck; link?: ReadinessSourceLink }) {
  const t = useT();
  const { icon: Icon, className } = STATUS_STYLES[check.status];
  const regions = check.regions ?? [];
  return (
    <li className="flex items-start gap-2 px-3 py-1.5 text-xs">
      <Icon className={cn("mt-0.5 size-3.5 shrink-0", className)} aria-hidden />
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div>
          <span className="font-medium">{check.label}</span>
          {check.detail && (
            <span
              className={cn("text-muted-foreground", check.status === "fail" && "text-destructive")}
            >
              {" "}
              · {check.detail}
            </span>
          )}
        </div>
        {regions.length > 0 && (
          <ul aria-label={t("Regions no test reaches")} className="flex flex-col">
            {regions.map((region) => {
              const label = TARIFF_REGION_LABELS[region.kind];
              const line = link ? t("Ln {0}", lineOf(link.source, region.range.from)) : null;
              return (
                <li key={`${region.kind}:${region.range.from}`}>
                  {link ? (
                    <button
                      type="button"
                      onClick={() => link.onReveal(region.range)}
                      className="ui-focus-ring hover:bg-surface-hover text-foreground-muted flex w-full items-center gap-2 rounded-sm px-1 py-0.5 text-left"
                    >
                      <span>{label}</span>
                      <span className="text-foreground-subtle ml-auto font-mono text-2xs tabular-nums">
                        {line}
                      </span>
                    </button>
                  ) : (
                    <span className="text-foreground-muted px-1">{label}</span>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </li>
  );
}

/**
 * The server's checks with the Studio's own for whatever the server does not judge (the regions
 * the tests reach, which the browser's run knows), each key once, the server's first.
 */
export function withLiveChecks(
  checks: readonly ReadinessCheck[],
  live: readonly ReadinessCheck[] | undefined,
): ReadinessCheck[] {
  const keys = new Set(checks.map((check) => check.key));
  return [...checks, ...(live ?? []).filter((check) => !keys.has(check.key))];
}

function readinessHeading(step: ReadinessStep, ready: boolean, t: TranslateFn): string {
  if (step === "submit") return ready ? t("Ready to submit") : t("Not ready to submit");
  return ready ? t("Ready to approve") : t("Not ready to approve");
}

export function isReadyFor(step: ReadinessStep, readiness: ReadinessResponse): boolean {
  return step === "submit" ? readiness.canSubmit : readiness.canApprove;
}

/**
 * The review gate, shown before the button is pressed. Every row is a check Submit or Approve
 * enforces on the server, so a red row here is exactly the refusal the person would otherwise
 * meet after clicking.
 */
export function ReadinessView({
  data,
  isLoading,
  isError,
  step,
  liveChecks,
  link,
}: {
  data: ReadinessResponse | undefined;
  isLoading: boolean;
  isError: boolean;
  step: ReadinessStep;
  /** The Studio's checks of the saved source, shown where the server sends none of that key. */
  liveChecks?: readonly ReadinessCheck[];
  /** Makes a check's regions name their lines and go to them. */
  link?: ReadinessSourceLink;
}) {
  const t = useT();
  const ready = data ? isReadyFor(step, data) : null;

  if (isLoading) {
    return (
      <div
        className="space-y-1.5 rounded-md border p-3"
        aria-busy
        aria-label={t("Checking readiness")}
      >
        <Skeleton className="h-3.5 w-32" />
        <Skeleton className="h-3 w-56" />
        <Skeleton className="h-3 w-48" />
        <Skeleton className="h-3 w-40" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="text-muted-foreground rounded-md border px-3 py-2 text-xs">
        {t(
          "The readiness check could not run. The server will still enforce every rule when you confirm.",
        )}
      </div>
    );
  }

  const failing = data.checks.filter((check) => check.status === "fail");
  const checks = withLiveChecks(data.checks, liveChecks);
  const relevant = step === "submit" ? checks.filter((check) => check.key !== "reviewer") : checks;

  return (
    <div className="overflow-hidden rounded-md border">
      <div
        className={cn(
          "flex items-center justify-between gap-2 border-b px-3 py-2 text-xs font-semibold",
          ready
            ? "bg-success-subtle text-success-subtle-foreground"
            : "bg-danger-subtle text-danger-subtle-foreground",
        )}
      >
        <span>{readinessHeading(step, ready === true, t)}</span>
        {failing.length > 0 && (
          <span className="font-normal">
            {t("{0, plural, one {# blocking issue} other {# blocking issues}}", failing.length)}
          </span>
        )}
      </div>
      <ul className="divide-y">
        {relevant.map((check) => (
          <CheckRow key={check.key} check={check} link={link} />
        ))}
      </ul>
    </div>
  );
}
