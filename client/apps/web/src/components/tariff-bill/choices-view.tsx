import { refusalText } from "@/components/tariff-bill/refusal-text";
import { CheckIcon, MinusIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import type {
  TariffChoice,
  TariffChoiceOption,
  TariffChoiceStatus,
  TariffRange,
} from "@trenova/shared/types/tariff";

const KIND_LABELS: Record<TariffChoice["kind"], string> = defineLabels({
  best: "Lowest wins",
  worst: "Highest wins",
});

const STATUS_LABELS: Record<TariffChoiceStatus, string> = defineLabels({
  chosen: "Chosen",
  lost: "Not chosen",
  refused: "Refused",
});

const STATUS_ICONS = { chosen: CheckIcon, lost: MinusIcon, refused: XCloseIcon } as const;

export type ChoicesViewProps = {
  /** Every `best` and `worst` the rating evaluated, in evaluation order. */
  choices: readonly TariffChoice[];
  /**
   * The rating currency's ISO code, when known; subtotals then show its symbol. A choice that
   * names its own currency (a stored rating's) is shown in that one.
   */
  currency?: string | null;
  /** Reveals an option's or a choice's source; without it the view is read-only text. */
  onSelectRange?: (range: TariffRange) => void;
  className?: string;
};

function OptionRow({
  option,
  currency,
  onSelectRange,
}: {
  option: TariffChoiceOption;
  currency?: string | null;
  onSelectRange?: (range: TariffRange) => void;
}) {
  const t = useT();
  const chosen = option.status === "chosen";
  const Icon = STATUS_ICONS[option.status];
  const body = (
    <>
      <Icon
        aria-hidden
        className={cn(
          "mt-0.5 size-3.5 shrink-0",
          chosen ? "text-success" : "text-foreground-subtle",
        )}
      />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span
          className={cn("truncate text-sm", !chosen && "text-foreground-muted")}
          title={option.label}
        >
          {option.label}
          <span className="sr-only"> · {STATUS_LABELS[option.status]}</span>
        </span>
        {option.refusal && (
          <span className="text-foreground-subtle text-xs break-words">
            {refusalText(option.refusal, t)}
          </span>
        )}
      </span>
      <span
        className={cn(
          "shrink-0 text-sm",
          option.subtotal && "font-mono tabular-nums",
          chosen ? "text-foreground" : "text-foreground-subtle",
          option.status === "lost" && "line-through decoration-foreground-subtle",
        )}
      >
        {option.subtotal ? formatTariffValue(option.subtotal, t, { currency }) : t("Refused")}
      </span>
    </>
  );
  const rowClass = "flex w-full items-start gap-2 px-(--cell-px) py-1.5 text-left";
  return (
    <li data-status={option.status}>
      {onSelectRange && option.range ? (
        <button
          type="button"
          className={cn(rowClass, "hover:bg-surface-hover ui-focus-ring rounded-md")}
          onClick={() => option.range && onSelectRange(option.range)}
        >
          {body}
        </button>
      ) : (
        <div className={rowClass}>{body}</div>
      )}
    </li>
  );
}

/**
 * What each `best` or `worst` did: its label and rule, which option it kept, and every option
 * with its subtotal or why it refused, the options that lost drawn muted. Labels are the
 * tariff's text and are shown as written; a refusal written by the module is translated.
 */
export function ChoicesView({ choices, currency, onSelectRange, className }: ChoicesViewProps) {
  const t = useT();
  if (choices.length === 0) return null;
  return (
    <section aria-label={t("Choices")} className={cn("flex flex-col gap-3", className)}>
      {choices.map((choice, index) => (
        <div
          // oxlint-disable-next-line react/no-array-index-key -- a choice inside a loop repeats its label, so its place in the order is its identity
          key={index}
          className="flex flex-col gap-1"
        >
          <div className="flex min-w-0 items-center gap-2">
            <span className="truncate text-xs font-medium" title={choice.label}>
              {choice.label}
            </span>
            <Badge variant="neutral" appearance="outline" className="shrink-0">
              {KIND_LABELS[choice.kind]}
            </Badge>
            <span className="flex-1" />
            <span className="text-foreground-muted shrink-0 text-xs">
              {choice.chosen === null ? t("Every option refused") : t("Chose {0}", choice.chosen)}
            </span>
          </div>
          <ul className="divide-border -mx-(--cell-px) flex flex-col divide-y">
            {choice.options.map((option) => (
              <OptionRow
                key={option.label}
                option={option}
                currency={choice.currency ?? currency}
                onSelectRange={onSelectRange}
              />
            ))}
          </ul>
        </div>
      ))}
    </section>
  );
}
