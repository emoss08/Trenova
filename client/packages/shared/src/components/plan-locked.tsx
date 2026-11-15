import { Lock01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  canManagePlan,
  planLockedGuidance,
  planLockedSentence,
} from "@trenova/shared/lib/plan-copy";
import { cn } from "@trenova/shared/lib/utils";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { type ReactNode, useId } from "react";

export type PlanLockedVariant = "card" | "inline" | "page";

/** Whether the person manages the organization, and so adds an add-on themselves. */
export function useCanManagePlan(): boolean {
  return usePermissionStore((state) => canManagePlan(state.hasPermission));
}

export type PlanLockedNoticeProps = {
  /** The feature's name as the server gave it; the thing the sentence is about unless `title` is set. */
  featureName: string;
  /** The add-on's name as the server gave it; empty for a feature of the base plan. */
  addOnName: string;
  variant?: PlanLockedVariant;
  /** What is locked, in the caller's words (a report's or a widget's name). */
  title?: string;
  className?: string;
  /** The page variant's way forward (a link home), after the explanation. */
  actions?: ReactNode;
};

/**
 * The explanation for something the organization's plan does not include, from names the
 * caller already has (a refusal's add-on, a manifest entry). `PlanLocked` is the usual way in.
 */
export function PlanLockedNotice({
  featureName,
  addOnName,
  variant = "inline",
  title,
  className,
  actions,
}: PlanLockedNoticeProps) {
  const t = useT();
  const titleId = useId();
  const managesPlan = useCanManagePlan();
  const thing = title || featureName;
  const sentence = planLockedSentence(t, thing, addOnName);
  const guidance = planLockedGuidance(t, managesPlan);

  if (variant === "inline") {
    return (
      <span
        data-slot="plan-locked"
        data-variant="inline"
        className={cn("text-foreground-muted text-sm text-pretty", className)}
      >
        {sentence}
      </span>
    );
  }

  if (variant === "card") {
    return (
      <section
        data-slot="plan-locked"
        data-variant="card"
        aria-labelledby={titleId}
        aria-description={`${sentence} ${guidance}`}
        className={cn(
          "border-border bg-card flex h-full min-h-28 flex-col gap-2 rounded-lg border p-3",
          className,
        )}
      >
        <div className="flex items-center gap-2">
          <Lock01Icon aria-hidden className="text-foreground-muted size-3.5 shrink-0" />
          <h3 id={titleId} className="min-w-0 flex-1 truncate text-sm font-medium">
            {thing}
          </h3>
        </div>
        {addOnName ? (
          <Badge variant="neutral" className="max-w-full">
            <span className="truncate">{t("Part of {0}", addOnName)}</span>
          </Badge>
        ) : null}
        <p className="text-foreground-muted mt-auto text-xs">{t("Not on your plan")}</p>
      </section>
    );
  }

  return (
    <section
      data-slot="plan-locked"
      data-variant="page"
      aria-labelledby={titleId}
      className={cn(
        "animate-rise flex w-full flex-1 flex-col items-center justify-center px-4 py-16 text-center",
        className,
      )}
    >
      <div className="flex w-full max-w-md flex-col items-center">
        <span
          aria-hidden
          className="border-neutral-border bg-neutral-subtle text-neutral-subtle-foreground flex size-12 items-center justify-center rounded-lg border"
        >
          <Lock01Icon className="size-5" />
        </span>
        {addOnName ? (
          <Badge variant="neutral" className="mt-4 max-w-full">
            <span className="truncate">{t("Part of {0}", addOnName)}</span>
          </Badge>
        ) : null}
        <h2 id={titleId} className="mt-3 text-2xl font-semibold text-balance">
          {t("Not on your plan")}
        </h2>
        <p className="text-foreground-muted mt-1.5 max-w-[52ch] text-base text-pretty">
          {sentence}
        </p>
        <p className="text-foreground-muted mt-1 max-w-[52ch] text-sm text-pretty">{guidance}</p>
        {actions ? (
          <div className="mt-5 flex flex-wrap items-center justify-center gap-2">{actions}</div>
        ) : null}
      </div>
    </section>
  );
}

export type PlanLockedProps = {
  /** The feature key; never shown. */
  feature: string;
  variant?: PlanLockedVariant;
  title?: string;
  className?: string;
};

/**
 * What stands in for something the organization's plan does not include, naming the add-on
 * it belongs to. Renders nothing when the plan includes the feature (or the permission
 * manifest has not loaded), so a caller can put it in front of the real content:
 * `<PlanLocked feature="tms.factoring" variant="card" title={t("Factoring")} />`.
 */
export function PlanLocked({ feature, variant = "inline", title, className }: PlanLockedProps) {
  const plan = usePlanFeature(feature);
  if (plan.available) return null;
  return (
    <PlanLockedNotice
      featureName={plan.featureName}
      addOnName={plan.addOnName}
      variant={variant}
      title={title}
      className={className}
    />
  );
}
