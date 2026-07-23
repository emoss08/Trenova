import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import type { GeneratedFuelRow } from "@/lib/graphql/fuel-surcharge";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import { useDebounce } from "@/hooks/use-debounce";
import { useQuery } from "@tanstack/react-query";
import { LoaderCircle, TriangleAlert, Wand2 } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { NumericFormat } from "react-number-format";
import { toast } from "sonner";

export type GenerateValueMeta = {
  label: string;
  prefix?: string;
  suffix?: string;
  decimalScale: number;
};

type GenerateTableDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  valueMeta: GenerateValueMeta;
  replaceCount: number;
  onApply: (rows: GeneratedFuelRow[]) => void;
};

type WizardState = {
  minPrice: number | null;
  maxPrice: number | null;
  increment: number | null;
  startValue: number | null;
  valueStep: number | null;
  openEnded: boolean;
};

const DEFAULT_STATE: WizardState = {
  minPrice: 1.2,
  maxPrice: 6,
  increment: 0.05,
  startValue: 0,
  valueStep: 0.01,
  openEnded: true,
};

const MAX_BANDS = 500;

function isValidState(state: WizardState): boolean {
  return (
    state.minPrice !== null &&
    state.maxPrice !== null &&
    state.increment !== null &&
    state.startValue !== null &&
    state.valueStep !== null &&
    state.increment > 0 &&
    state.maxPrice > state.minPrice
  );
}

function estimatedBands(state: WizardState): number {
  if (!isValidState(state)) return 0;
  const core = Math.ceil(
    ((state.maxPrice as number) - (state.minPrice as number)) / (state.increment as number),
  );
  return core + (state.openEnded ? 2 : 0);
}

function WizardField({
  label,
  helper,
  value,
  onChange,
  prefix,
  suffix,
  decimalScale,
  invalid,
}: {
  label: string;
  helper: string;
  value: number | null;
  onChange: (value: number | null) => void;
  prefix?: string;
  suffix?: string;
  decimalScale: number;
  invalid?: boolean;
}) {
  const id = useId();
  return (
    <div className="space-y-1">
      <Label htmlFor={id} className="text-xs font-medium">
        {label}
      </Label>
      <NumericFormat
        id={id}
        value={value ?? ""}
        onValueChange={(values) => onChange(values.floatValue ?? null)}
        decimalScale={decimalScale}
        allowNegative={false}
        prefix={prefix}
        suffix={suffix}
        inputMode="decimal"
        className={cn(
          "flex h-8 w-full rounded-md border border-input bg-muted px-2.5 text-sm tabular-nums outline-none",
          "transition-[border-color,box-shadow] duration-150 ease-in-out",
          "focus-visible:border-brand focus-visible:bg-background focus-visible:ring-4 focus-visible:ring-brand/20",
          invalid && "border-red-500/60 bg-red-500/10",
        )}
      />
      <p className="text-xs leading-snug text-muted-foreground">{helper}</p>
    </div>
  );
}

export function GenerateTableDialog({
  open,
  onOpenChange,
  valueMeta,
  replaceCount,
  onApply,
}: GenerateTableDialogProps) {
  const [state, setState] = useState<WizardState>(DEFAULT_STATE);
  const openEndedId = useId();

  useEffect(() => {
    if (open) {
      setState(DEFAULT_STATE);
    }
  }, [open]);

  const debounced = useDebounce(state, 300);
  const valid = isValidState(debounced);
  const estimate = estimatedBands(debounced);
  const tooMany = estimate > MAX_BANDS;

  const { data: preview, isFetching } = useQuery({
    ...queries.fuelSurcharge.generateTable({
      minPrice: String(debounced.minPrice),
      maxPrice: String(debounced.maxPrice),
      increment: String(debounced.increment),
      startValue: String(debounced.startValue),
      valueStep: String(debounced.valueStep),
      openEnded: debounced.openEnded,
    }),
    enabled: open && valid && !tooMany,
    staleTime: 60_000,
    placeholderData: (previous) => previous,
  });

  const update = (patch: Partial<WizardState>) => setState((prev) => ({ ...prev, ...patch }));

  const rangeInvalid =
    state.minPrice !== null && state.maxPrice !== null && state.maxPrice <= state.minPrice;

  const handleApply = () => {
    if (!preview || preview.length === 0) return;
    onApply(preview);
    onOpenChange(false);
    toast.success(`${preview.length} price bands created`, {
      description: "Every band is editable — adjust any range or value before saving.",
    });
  };

  const formatValue = (value: string) => {
    const numeric = Number(value);
    const text = Number.isFinite(numeric) ? numeric.toFixed(valueMeta.decimalScale) : value;
    return `${valueMeta.prefix ?? ""}${text}${valueMeta.suffix ?? ""}`;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Wand2 className="size-4" />
            Generate Price Bands
          </DialogTitle>
          <DialogDescription>
            Answer a few questions and the whole table is built for you — the preview updates as you
            type. Every band stays editable afterward.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-5 sm:grid-cols-[260px_1fr]">
          <div className="space-y-3.5">
            <WizardField
              label="Lowest fuel price"
              helper="Bands start here — usually your peg (base) price."
              value={state.minPrice}
              onChange={(value) => update({ minPrice: value })}
              prefix="$"
              decimalScale={4}
              invalid={rangeInvalid}
            />
            <WizardField
              label="Highest fuel price"
              helper="Bands stop here. Pick a price fuel is unlikely to exceed."
              value={state.maxPrice}
              onChange={(value) => update({ maxPrice: value })}
              prefix="$"
              decimalScale={4}
              invalid={rangeInvalid}
            />
            <WizardField
              label="Band width"
              helper="How much fuel price each band covers — 5¢ is the industry standard."
              value={state.increment}
              onChange={(value) => update({ increment: value })}
              prefix="$"
              decimalScale={4}
              invalid={state.increment !== null && state.increment <= 0}
            />
            <WizardField
              label={`Starting ${valueMeta.label.toLowerCase()}`}
              helper="Charged in the first (lowest price) band."
              value={state.startValue}
              onChange={(value) => update({ startValue: value })}
              prefix={valueMeta.prefix}
              suffix={valueMeta.suffix}
              decimalScale={valueMeta.decimalScale}
            />
            <WizardField
              label="Increase per band"
              helper={`How much the ${valueMeta.label.toLowerCase()} goes up from one band to the next.`}
              value={state.valueStep}
              onChange={(value) => update({ valueStep: value })}
              prefix={valueMeta.prefix}
              suffix={valueMeta.suffix}
              decimalScale={valueMeta.decimalScale}
            />
            <div className="flex items-start gap-2.5 rounded-lg border bg-muted/40 p-3">
              <Switch
                checked={state.openEnded}
                onCheckedChange={(checked) => update({ openEnded: checked })}
                id={openEndedId}
              />
              <div>
                <Label htmlFor={openEndedId} className="text-xs font-medium">
                  Cover prices outside the range
                </Label>
                <p className="mt-0.5 text-xs leading-snug text-muted-foreground">
                  Adds open-ended bottom and top bands so every possible fuel price matches a band.
                  Recommended.
                </p>
              </div>
            </div>
          </div>

          <div className="flex min-h-72 flex-col overflow-hidden rounded-lg border">
            <div className="flex items-center justify-between border-b bg-muted/60 px-3 py-2">
              <span className="text-xs font-medium">Preview</span>
              <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                {isFetching && <LoaderCircle className="size-3 animate-spin" />}
                {valid && !tooMany ? `${preview?.length ?? estimate} bands` : ""}
              </span>
            </div>
            {!valid ? (
              <div className="flex flex-1 items-center justify-center p-6 text-center text-xs text-muted-foreground">
                {rangeInvalid
                  ? "The highest price must be above the lowest price."
                  : "Fill in the fields on the left to see the table."}
              </div>
            ) : tooMany ? (
              <div className="flex flex-1 flex-col items-center justify-center gap-2 p-6 text-center text-xs text-muted-foreground">
                <TriangleAlert className="size-5 text-amber-500" />
                <p>
                  That would create {estimate} bands (limit {MAX_BANDS}). Widen the band width or
                  narrow the price range.
                </p>
              </div>
            ) : (
              <div className="flex-1 overflow-y-auto">
                <table className="w-full text-sm">
                  <thead className="sticky top-0 bg-muted/80 backdrop-blur">
                    <tr className="text-left text-xs text-muted-foreground">
                      <th className="px-3 py-1.5 font-medium">From</th>
                      <th className="px-3 py-1.5 font-medium">Up To</th>
                      <th className="px-3 py-1.5 font-medium">{valueMeta.label}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(preview ?? []).map((row, index) => (
                      <tr key={index} className="border-t tabular-nums">
                        <td className="px-3 py-1">
                          {row.priceMin != null ? `$${Number(row.priceMin).toFixed(2)}` : "Any"}
                        </td>
                        <td className="px-3 py-1">
                          {row.priceMax != null
                            ? `$${Number(row.priceMax).toFixed(2)}`
                            : "No limit"}
                        </td>
                        <td className="px-3 py-1">{formatValue(row.value)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="items-center gap-3 sm:justify-between">
          <span className="text-xs text-muted-foreground">
            {replaceCount > 0
              ? `Applying replaces your ${replaceCount} existing ${replaceCount === 1 ? "band" : "bands"}.`
              : ""}
          </span>
          <div className="flex gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleApply}
              disabled={!valid || tooMany || isFetching || !preview || preview.length === 0}
            >
              Use These Bands{preview && !tooMany && valid ? ` (${preview.length})` : ""}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
