import { useT } from "@trenova/shared/i18n/use-t";
import { ChevronUpIcon, Hourglass01Icon } from "@trenova/shared/components/icons";

/**
 * What is left of the approval box once its decisions were put off: a pill
 * above the composer saying how many wait. Selecting it opens the box again
 * on the oldest of them.
 */
export function DeferredDecisionsPill({
  count,
  onReopen,
}: {
  count: number;
  onReopen: () => void;
}) {
  const t = useT();

  return (
    <button
      type="button"
      onClick={onReopen}
      data-slot="deferred-decisions"
      className="ui-focus-ring ui-press bg-warning-subtle text-warning-subtle-foreground border-warning-border flex h-6 items-center gap-1.5 self-start rounded-full border px-2.5 text-xs transition-colors"
    >
      <Hourglass01Icon aria-hidden className="size-3 shrink-0" />
      <span>{t("{0, plural, one {# decision waiting} other {# decisions waiting}}", count)}</span>
      <span className="font-medium">{t("Review")}</span>
      <ChevronUpIcon aria-hidden className="size-3 shrink-0" />
    </button>
  );
}
