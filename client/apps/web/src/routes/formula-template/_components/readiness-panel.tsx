import { type TranslateFn, useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ReadinessCheck, ReadinessResponse } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, CheckCircleIcon, XCircleIcon } from "@trenova/shared/components/icons";
import { useEffect } from "react";

export type ReadinessStep = "submit" | "approve";

const STATUS_STYLES: Record<
  ReadinessCheck["status"],
  { icon: typeof CheckCircleIcon; className: string }
> = {
  pass: { icon: CheckCircleIcon, className: "text-success-foreground" },
  warn: { icon: AlertTriangleIcon, className: "text-warning-foreground" },
  fail: { icon: XCircleIcon, className: "text-destructive" },
};

function CheckRow({ check }: { check: ReadinessCheck }) {
  const t = useT();

  const { icon: Icon, className } = STATUS_STYLES[check.status];
  return (
    <li className="flex items-start gap-2 px-3 py-1.5 text-xs">
      <Icon className={cn("mt-0.5 size-3.5 shrink-0", className)} aria-hidden />
      <div className="min-w-0">
        <span className="font-medium">{t(check.label)}</span>
        {check.detail && (
          <span
            className={cn("text-muted-foreground", check.status === "fail" && "text-destructive")}
          >
            {" "}
            · {check.detail}
          </span>
        )}
      </div>
    </li>
  );
}

function readinessHeading(step: ReadinessStep, ready: boolean, t: TranslateFn): string {
  if (step === "submit") return ready ? t("Ready to submit") : t("Not ready to submit");
  return ready ? t("Ready to approve") : t("Not ready to approve");
}

export function isReadyFor(step: ReadinessStep, readiness: ReadinessResponse): boolean {
  return step === "submit" ? readiness.canSubmit : readiness.canApprove;
}

/**
 * The review gate, shown before the button is pressed. Every row is a check
 * Submit or Approve enforces server-side, so a red row here is exactly the
 * refusal the user would otherwise meet after clicking.
 */
export function ReadinessPanel({
  templateId,
  step,
  onReadinessChange,
}: {
  templateId: string;
  step: ReadinessStep;
  onReadinessChange?: (ready: boolean | null) => void;
}) {
  const t = useT();

  const { data, isLoading, isError } = useQuery({
    ...queries.formulaTemplate.readiness(templateId),
    enabled: !!templateId,
    staleTime: 0,
  });

  const ready = data ? isReadyFor(step, data) : null;
  const effectiveReady = isError ? true : ready;
  useEffect(() => {
    onReadinessChange?.(effectiveReady);
  }, [effectiveReady, onReadinessChange]);

  if (isLoading) {
    return (
      <div className="space-y-1.5 rounded-md border p-3">
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
  const relevant =
    step === "submit" ? data.checks.filter((check) => check.key !== "reviewer") : data.checks;

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
          <CheckRow key={check.key} check={check} />
        ))}
      </ul>
    </div>
  );
}
