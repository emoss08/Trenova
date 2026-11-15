import { Lock01Icon } from "@trenova/shared/components/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { useT } from "@trenova/shared/i18n/use-t";
import { withheldAccessLabel, type WithheldAccess } from "@trenova/shared/lib/reference-access";
import { planLockedSentence } from "@trenova/shared/lib/plan-copy";
import { cn } from "@trenova/shared/lib/utils";
import type { PlanWithholding } from "@trenova/shared/lib/withheld";
import type { ReactNode } from "react";

type LockedReferenceProps = WithheldAccess & {
  children: ReactNode;
  className?: string;
  /**
   * `inline` (the default) draws the lock before a name and truncates the name; `row` keeps
   * the caller's own row layout (an avatar, a name and a caption) and ends it with the lock.
   */
  layout?: "inline" | "row";
};

/**
 * Something about a linked record drawn without its kind's read: a lock, the content, and a
 * tooltip saying whose read is missing. See docs/engineering/reference-views.md.
 */
export function LockedReference({
  children,
  className,
  layout = "inline",
  ...access
}: LockedReferenceProps) {
  const t = useT();
  const reason = t("You don't have access to {0}", withheldAccessLabel(access));
  const lock = (
    <Lock01Icon
      aria-hidden
      data-slot="locked-reference-icon"
      className="size-3 shrink-0 text-foreground-muted"
    />
  );

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          layout === "row" ? (
            <div
              data-slot="locked-reference"
              tabIndex={0}
              aria-description={reason}
              className={cn("cursor-default ui-focus-ring", className)}
            />
          ) : (
            <span
              data-slot="locked-reference"
              tabIndex={0}
              aria-description={reason}
              className={cn(
                "inline-flex min-w-0 cursor-default items-center gap-1 rounded-badge ui-focus-ring",
                className,
              )}
            />
          )
        }
      >
        {layout === "row" ? (
          <>
            {children}
            {lock}
          </>
        ) : (
          <>
            {lock}
            <span className="truncate">{children}</span>
          </>
        )}
      </TooltipTrigger>
      <TooltipContent>{reason}</TooltipContent>
    </Tooltip>
  );
}

type WithheldPanelProps = {
  /** What the person cannot see and whose access it needs, as one translated sentence. */
  children: ReactNode;
  className?: string;
};

/**
 * A whole card or section withheld because the person may not read a kind of record it is
 * about: a lock and a sentence naming the access it needs, in place of the content, so it
 * never reads as an empty card.
 */
export function WithheldPanel({ children, className }: WithheldPanelProps) {
  return (
    <div
      data-slot="withheld-panel"
      className={cn(
        "flex flex-1 flex-col items-center justify-center gap-2 px-3 py-6 text-center",
        className,
      )}
    >
      <Lock01Icon aria-hidden className="size-4 shrink-0 text-foreground-muted" />
      <p className="max-w-64 text-xs text-foreground-muted">{children}</p>
    </div>
  );
}

/**
 * A value the organization's plan withheld: the feature key it belongs to (`"tms.factoring"`),
 * or the response's withholding for its path (`planWithholdingAt(withholdingsOf(data), path)`).
 */
export type WithheldByPlan = {
  plan: string | PlanWithholding;
  kind?: never;
  group?: never;
};

type WithheldValueProps = (WithheldAccess | WithheldByPlan) & {
  className?: string;
};

function isWithheldByPlan(access: WithheldAccess | WithheldByPlan): access is WithheldByPlan {
  return "plan" in access && access.plan !== undefined;
}

/**
 * A value the plan withheld, in the same compact form as "Hidden": a lock and the add-on it is
 * part of, with the whole sentence in a tooltip. The names come from the permission manifest;
 * when the manifest does not list the feature (it has not loaded, or it is older than the
 * response), it says only that the value is not on the plan.
 */
function PlanWithheldValue({
  plan,
  className,
}: {
  plan: string | PlanWithholding;
  className?: string;
}) {
  const t = useT();
  const feature = typeof plan === "string" ? plan : plan.feature;
  const { featureName, addOnName } = usePlanFeature(feature);
  const label = addOnName ? t("Part of {0}", addOnName) : t("Not on your plan");
  const reason = planLockedSentence(t, featureName, addOnName);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            data-slot="locked-reference"
            data-withheld="plan"
            tabIndex={0}
            aria-description={reason}
            className={cn(
              "inline-flex min-w-0 cursor-default items-center gap-1 rounded-badge text-xs text-foreground-muted ui-focus-ring",
              className,
            )}
          />
        }
      >
        <Lock01Icon
          aria-hidden
          data-slot="locked-reference-icon"
          className="size-3 shrink-0 text-foreground-muted"
        />
        <span className="truncate">{label}</span>
      </TooltipTrigger>
      <TooltipContent>{reason}</TooltipContent>
    </Tooltip>
  );
}

/**
 * A field the server withheld: a linked record's field the person may not read the kind of
 * (`kind`, `null` and named in the response's `extensions.withheldByAccess`), a protected
 * field whose group the person's role does not grant (`group`, such as a margin), or a value
 * the organization's plan does not include (`plan`, named in `extensions.withheldByPlan` or the
 * `Withheld-By-Plan` header). It says so rather than drawing a dash, a zero or "None", which
 * would read as the actual value.
 */
export function WithheldValue({ className, ...access }: WithheldValueProps) {
  const t = useT();
  if (isWithheldByPlan(access)) {
    return <PlanWithheldValue plan={access.plan} className={className} />;
  }
  return (
    <LockedReference {...access} className={cn("text-xs text-foreground-muted", className)}>
      {t("Hidden")}
    </LockedReference>
  );
}
