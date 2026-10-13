import { NumberField } from "@/components/fields/number-field";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  MAX_ROUNDING_PRECISION,
  type FormulaRoundingMode,
  type FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import {
  ArrowRightIcon,
  MinusIcon,
  PlusIcon,
  Shield01Icon,
} from "@trenova/shared/components/icons";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { formatMoney, previewAmounts } from "../model/format";
import { useStudio } from "../studio-context";
import { SectionHead } from "./section-head";

function PipeStep({
  label,
  amount,
  tone,
}: {
  label: string;
  amount: string;
  tone?: "hit" | "bill";
}) {
  return (
    <span
      className={cn(
        "flex flex-col rounded-md border px-2.5 py-1",
        tone === "hit" && "border-warning-border bg-warning-subtle",
        tone === "bill" && "bg-ink text-ink-foreground border-transparent",
        !tone && "bg-card",
      )}
    >
      <span
        className={cn(
          "text-2xs",
          tone === "bill"
            ? "text-ink-foreground/70"
            : tone === "hit"
              ? "text-warning-subtle-foreground"
              : "text-foreground-subtle",
        )}
      >
        {label}
      </span>
      <span key={amount} className="fs-settle font-mono text-sm tabular-nums">
        {amount}
      </span>
    </span>
  );
}

/**
 * What happens to the formula's result before it is billed: a floor and a
 * ceiling, then rounding. The strip below shows it on the preview load.
 */
export function PolicySection() {
  const t = useT();
  const { preview } = useStudio();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const roundingMode = useWatch({ control, name: "roundingMode" });
  const roundingPrecision = useWatch({ control, name: "roundingPrecision" }) ?? 2;
  const amounts = previewAmounts(preview.result);
  const notRounded = roundingMode === "None";
  const places = notRounded ? 4 : Math.max(roundingPrecision, 0);

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        section="policy"
        icon={Shield01Icon}
        title={t("Charge policy")}
        description={t(
          "Clamp the formula's result to a range, then round it to what gets billed. Preview, scenarios and production all apply it the same way.",
        )}
      />
      <div className="bg-card flex flex-col gap-4 rounded-lg border p-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <NumberField
            label={t("Minimum charge")}
            name="minCharge"
            control={control}
            placeholder={t("No minimum")}
            prefix="$"
            decimalScale={2}
            thousandSeparator
          />
          <NumberField
            label={t("Maximum charge")}
            name="maxCharge"
            control={control}
            placeholder={t("No maximum")}
            prefix="$"
            decimalScale={2}
            thousandSeparator
          />
          <div className="flex flex-col gap-1.5">
            <span className="text-foreground-subtle text-xs font-medium">{t("Rounding")}</span>
            <Controller
              control={control}
              name="roundingMode"
              render={({ field }) => (
                <SegmentedControl<FormulaRoundingMode>
                  aria-label={t("Rounding")}
                  value={field.value ?? "HalfUp"}
                  onValueChange={field.onChange}
                  items={[
                    { value: "HalfUp", label: t("Half up") },
                    { value: "HalfEven", label: t("Half even") },
                    { value: "Up", label: t("Up") },
                    { value: "Down", label: t("Down") },
                    { value: "None", label: t("None") },
                  ]}
                />
              )}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <span className="text-foreground-subtle text-xs font-medium">
              {t("Decimal places")}
            </span>
            <Controller
              control={control}
              name="roundingPrecision"
              render={({ field }) => {
                const value = typeof field.value === "number" ? field.value : 2;
                return (
                  <div className="flex items-center gap-1">
                    <Button
                      type="button"
                      variant="outline"
                      size="icon-sm"
                      aria-label={t("Fewer decimal places")}
                      disabled={notRounded || value <= 0}
                      onClick={() => field.onChange(value - 1)}
                    >
                      <MinusIcon className="size-3.5" />
                    </Button>
                    <span
                      aria-live="polite"
                      className="w-8 text-center font-mono text-sm tabular-nums"
                    >
                      {notRounded ? "—" : value}
                    </span>
                    <Button
                      type="button"
                      variant="outline"
                      size="icon-sm"
                      aria-label={t("More decimal places")}
                      disabled={notRounded || value >= MAX_ROUNDING_PRECISION}
                      onClick={() => field.onChange(value + 1)}
                    >
                      <PlusIcon className="size-3.5" />
                    </Button>
                  </div>
                );
              }}
            />
          </div>
        </div>

        {amounts && (
          <div className="bg-sunken flex flex-wrap items-center gap-2 rounded-md px-3 py-2">
            <span className="text-foreground-subtle mr-1 text-xs">{t("On the preview load")}</span>
            <PipeStep label={t("Formula")} amount={formatMoney(amounts.raw, 4)} />
            <ArrowRightIcon className="text-foreground-subtle size-3" />
            <PipeStep
              label={
                amounts.bound === "min"
                  ? t("Raised to minimum")
                  : amounts.bound === "max"
                    ? t("Lowered to maximum")
                    : t("Within range")
              }
              amount={formatMoney(amounts.clamped, amounts.bound ? 2 : 4)}
              tone={amounts.bound ? "hit" : undefined}
            />
            <ArrowRightIcon className="text-foreground-subtle size-3" />
            <PipeStep
              label={t("Billed")}
              amount={formatMoney(amounts.billed, places)}
              tone="bill"
            />
          </div>
        )}
      </div>
    </section>
  );
}
