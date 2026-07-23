import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { FuelSurchargeProgramFormValues } from "@/types/fuel-surcharge";
import { useQuery } from "@tanstack/react-query";
import { Eye, PencilRuler } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";

type MatrixRow = {
  from: number;
  to: number;
  rate: number;
  current: boolean;
};

const MAX_CONVERTED_BANDS = 500;
const MPG_INCREMENT = 0.25;

function roundRate(value: number, precision: number, mode: string) {
  const factor = 10 ** precision;
  switch (mode) {
    case "Up":
      return Math.ceil(value * factor) / factor;
    case "Down":
      return Math.floor(value * factor) / factor;
    default:
      return Math.round(value * factor) / factor;
  }
}

function roundSteps(steps: number, mode: string) {
  switch (mode) {
    case "Down":
      return Math.floor(steps + 1e-9);
    case "Nearest":
      return Math.round(steps);
    default:
      return Math.ceil(steps - 1e-9);
  }
}

type FormulaParams = {
  method: string;
  peg: number;
  increment: number;
  incrementRate: number;
  milesPerGallon: number;
  stepRounding: string;
  rateRounding: string;
  precision: number;
};

function formulaIncrement(params: FormulaParams) {
  return params.method === "PerMileMPG" ? MPG_INCREMENT : params.increment;
}

function hasValidFormulaParams(params: FormulaParams) {
  if (!Number.isFinite(params.peg)) return false;
  if (params.method === "PerMileStep") {
    return (
      Number.isFinite(params.increment) &&
      params.increment > 0 &&
      Number.isFinite(params.incrementRate)
    );
  }
  if (params.method === "PerMileMPG") {
    return Number.isFinite(params.milesPerGallon) && params.milesPerGallon > 0;
  }
  return false;
}

function bandRate(params: FormulaParams, step: number) {
  const inc = formulaIncrement(params);
  const from = params.peg + step * inc;
  const to = from + inc;

  if (params.method === "PerMileMPG") {
    const mid = (from + to) / 2;
    return roundRate(
      Math.max(0, mid - params.peg) / params.milesPerGallon,
      params.precision,
      params.rateRounding,
    );
  }

  return roundRate(
    roundSteps((to - params.peg) / inc - 1e-9, params.stepRounding) * params.incrementRate,
    params.precision,
    params.rateRounding,
  );
}

function roundPrice(value: number) {
  return Math.round(value * 10000) / 10000;
}

function buildBand(params: FormulaParams, step: number, currentPrice: number | null): MatrixRow {
  const inc = formulaIncrement(params);
  const from = roundPrice(params.peg + step * inc);
  const to = roundPrice(from + inc);
  return {
    from,
    to,
    rate: bandRate(params, step),
    current: currentPrice !== null && currentPrice >= from && currentPrice < to,
  };
}

export function VirtualMatrixPreview({ disabled }: { disabled?: boolean }) {
  const { control, setValue } = useFormContext<FuelSurchargeProgramFormValues>();
  const method = useWatch({ control, name: "method" });
  const pegPrice = useWatch({ control, name: "pegPrice" });
  const increment = useWatch({ control, name: "increment" });
  const incrementRate = useWatch({ control, name: "incrementRate" });
  const milesPerGallon = useWatch({ control, name: "milesPerGallon" });
  const stepRounding = useWatch({ control, name: "stepRounding" });
  const rateRounding = useWatch({ control, name: "rateRounding" });
  const ratePrecision = useWatch({ control, name: "ratePrecision" });
  const fuelIndexId = useWatch({ control, name: "fuelIndexId" });
  const [convertOpen, setConvertOpen] = useState(false);

  const { data: dashboard } = useQuery(queries.fuelSurcharge.dashboard());
  const currentPrice = useMemo(() => {
    if (!fuelIndexId || !dashboard) return null;
    const entry = dashboard.find((item) => item.index.id === fuelIndexId);
    return entry?.latest ? Number(entry.latest.price) : null;
  }, [dashboard, fuelIndexId]);

  const params = useMemo<FormulaParams>(
    () => ({
      method,
      peg: Number(pegPrice),
      increment: Number(increment),
      incrementRate: Number(incrementRate),
      milesPerGallon: Number(milesPerGallon),
      stepRounding: stepRounding ?? "Up",
      rateRounding: rateRounding ?? "HalfUp",
      precision: Number(ratePrecision ?? 4),
    }),
    [
      method,
      pegPrice,
      increment,
      incrementRate,
      milesPerGallon,
      stepRounding,
      rateRounding,
      ratePrecision,
    ],
  );

  const rows = useMemo<MatrixRow[]>(() => {
    if (!hasValidFormulaParams(params)) {
      return [];
    }

    const inc = formulaIncrement(params);
    const windowSize = params.method === "PerMileMPG" ? 16 : 21;
    const lookBehind = params.method === "PerMileMPG" ? 8 : 10;
    const bandStart =
      currentPrice !== null ? Math.max(params.peg, currentPrice - inc * lookBehind) : params.peg;
    const startStep = Math.max(0, Math.floor((bandStart - params.peg) / inc));

    const result: MatrixRow[] = [];
    for (let step = startStep; step < startStep + windowSize; step++) {
      result.push(buildBand(params, step, currentPrice));
    }
    return result;
  }, [params, currentPrice]);

  const conversionRows = useMemo<FuelSurchargeProgramFormValues["tableRows"]>(() => {
    if (!convertOpen || !hasValidFormulaParams(params) || rows.length === 0) {
      return [];
    }

    const inc = formulaIncrement(params);
    const lastStep = Math.round((rows[rows.length - 1].from - params.peg) / inc);
    const bandCount = Math.min(lastStep + 1, MAX_CONVERTED_BANDS);

    const result: FuelSurchargeProgramFormValues["tableRows"] = [];
    result.push({ priceMin: null, priceMax: roundPrice(params.peg), value: 0, sortOrder: 0 });
    for (let step = 0; step < bandCount; step++) {
      const band = buildBand(params, step, null);
      result.push({
        priceMin: band.from,
        priceMax: band.to,
        value: band.rate,
        sortOrder: step + 1,
      });
    }
    result.push({
      priceMin: result[result.length - 1].priceMax,
      priceMax: null,
      value: bandRate(params, bandCount),
      sortOrder: result.length,
    });
    return result;
  }, [convertOpen, params, rows]);

  const handleConvert = useCallback(() => {
    if (conversionRows.length === 0) {
      return;
    }

    setValue("tableRows", conversionRows, { shouldDirty: true });
    setValue("method", "TablePerMile", { shouldDirty: true });
    setConvertOpen(false);
    toast.success(`${conversionRows.length} price bands created`, {
      description: "Adjust any band's range or rate, then save the program.",
    });
  }, [conversionRows, setValue]);

  if (rows.length === 0) {
    return null;
  }

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <div className="flex items-center gap-2">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
            <Eye className="size-4 text-primary" />
          </div>
          <div>
            <CardTitle className="text-sm font-medium">Live Matrix Preview</CardTitle>
            <p className="text-xs text-muted-foreground">
              Rendered from the formula parameters — no rows to maintain.
              {currentPrice !== null &&
                ` The highlighted band contains this week's price ($${currentPrice.toFixed(3)}).`}
            </p>
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setConvertOpen(true)}
          disabled={disabled}
          className="gap-1.5"
        >
          <PencilRuler className="size-3.5" />
          Customize Bands
        </Button>
      </CardHeader>
      <CardContent className="p-0">
        <div className="max-h-64 overflow-y-auto">
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-muted/80 backdrop-blur">
              <tr className="text-left text-xs text-muted-foreground">
                <th className="px-4 py-2 font-medium">Fuel Price From</th>
                <th className="px-4 py-2 font-medium">To</th>
                <th className="px-4 py-2 font-medium">Rate ($/mi)</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr
                  key={index}
                  className={cn(
                    "border-t tabular-nums",
                    row.current && "bg-primary/10 font-medium",
                  )}
                >
                  <td className="px-4 py-1.5">${row.from.toFixed(2)}</td>
                  <td className="px-4 py-1.5">${row.to.toFixed(2)}</td>
                  <td className="px-4 py-1.5">${row.rate.toFixed(4)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
      <ConvertToTableDialog
        open={convertOpen}
        onOpenChange={setConvertOpen}
        conversionRows={conversionRows}
        onConfirm={handleConvert}
      />
    </Card>
  );
}

const CONVERSION_PREVIEW_LIMIT = 25;

function formatBound(value: number | null | undefined, openLabel: string) {
  return typeof value === "number" ? `$${value.toFixed(2)}` : openLabel;
}

function ConvertToTableDialog({
  open,
  onOpenChange,
  conversionRows,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  conversionRows: FuelSurchargeProgramFormValues["tableRows"];
  onConfirm: () => void;
}) {
  const shown = conversionRows.slice(0, CONVERSION_PREVIEW_LIMIT);
  const hidden = conversionRows.length - shown.length;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">Make These Bands Editable</DialogTitle>
          <DialogDescription>
            Use this when the formula almost fits but some bands need a different range or rate —
            like a customer&apos;s own fuel table with uneven brackets.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 sm:grid-cols-[1fr_260px]">
          <ol className="space-y-3 text-sm">
            <li className="flex gap-2.5">
              <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary">
                1
              </span>
              <span className="text-muted-foreground">
                Your formula&apos;s full schedule is copied into the table on the right —{" "}
                <span className="font-medium text-foreground">{conversionRows.length} bands</span>{" "}
                covering every fuel price.
              </span>
            </li>
            <li className="flex gap-2.5">
              <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary">
                2
              </span>
              <span className="text-muted-foreground">
                Every band becomes editable — change any price range or rate, add bands, or delete
                them. They don&apos;t have to be evenly spaced.
              </span>
            </li>
            <li className="flex gap-2.5">
              <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary">
                3
              </span>
              <span className="text-muted-foreground">
                The formula fields no longer apply. Nothing is saved until you save the program —
                switching the method back undoes this.
              </span>
            </li>
          </ol>

          <div className="flex max-h-64 flex-col overflow-hidden rounded-lg border">
            <div className="border-b bg-muted/60 px-3 py-1.5 text-xs font-medium">
              Your table will look like this
            </div>
            <div className="flex-1 overflow-y-auto">
              <table className="w-full text-xs">
                <thead className="sticky top-0 bg-muted/80 backdrop-blur">
                  <tr className="text-left text-muted-foreground">
                    <th className="px-2.5 py-1.5 font-medium">From</th>
                    <th className="px-2.5 py-1.5 font-medium">Up To</th>
                    <th className="px-2.5 py-1.5 font-medium">$/mi</th>
                  </tr>
                </thead>
                <tbody className="tabular-nums">
                  {shown.map((row, index) => (
                    <tr key={index} className="border-t">
                      <td className="px-2.5 py-1">{formatBound(row.priceMin, "Any")}</td>
                      <td className="px-2.5 py-1">{formatBound(row.priceMax, "No limit")}</td>
                      <td className="px-2.5 py-1">
                        {typeof row.value === "number" ? `$${row.value.toFixed(4)}` : "—"}
                      </td>
                    </tr>
                  ))}
                  {hidden > 0 && (
                    <tr className="border-t">
                      <td colSpan={3} className="px-2.5 py-1.5 text-muted-foreground">
                        …and {hidden} more bands
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={onConfirm} disabled={conversionRows.length === 0}>
            Create {conversionRows.length} Editable Bands
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
