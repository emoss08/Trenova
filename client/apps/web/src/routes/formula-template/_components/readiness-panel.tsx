import { queries } from "@/lib/queries";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ReadinessCheck, ReadinessResponse } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, CheckCircle2Icon, XCircleIcon } from "lucide-react";
import { useEffect } from "react";

export type ReadinessStep = "submit" | "approve";

const STATUS_STYLES: Record<
  ReadinessCheck["status"],
  { icon: typeof CheckCircle2Icon; className: string }
> = {
  pass: { icon: CheckCircle2Icon, className: "text-emerald-600 dark:text-emerald-400" },
  warn: { icon: AlertTriangleIcon, className: "text-amber-600 dark:text-amber-400" },
  fail: { icon: XCircleIcon, className: "text-destructive" },
};

function CheckRow({ check }: { check: ReadinessCheck }) {
  const { icon: Icon, className } = STATUS_STYLES[check.status];
  return (
    <li className="flex items-start gap-2 px-3 py-1.5 text-xs">
      <Icon className={cn("mt-0.5 size-3.5 shrink-0", className)} aria-hidden />
      <div className="min-w-0">
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
    </li>
  );
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
        The readiness check could not run. The server will still enforce every rule when you
        confirm.
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
          ready ? "bg-emerald-500/10" : "bg-destructive/10",
        )}
      >
        <span>{ready ? "Ready to " + step : "Not ready to " + step}</span>
        {failing.length > 0 && (
          <span className="text-destructive font-normal">
            {failing.length} blocking {failing.length === 1 ? "issue" : "issues"}
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
