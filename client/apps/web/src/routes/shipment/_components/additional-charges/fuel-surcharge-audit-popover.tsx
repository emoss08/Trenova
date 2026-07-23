import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import type { FuelSurchargeDetail } from "@/types/shipment";
import { AlertTriangle, FuelIcon } from "lucide-react";

function DetailRow({ label, value }: { label: string; value: string | null }) {
  if (value === null) return null;
  return (
    <div className="flex items-center justify-between gap-4 text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums">{value}</span>
    </div>
  );
}

function money(value: number | null | undefined, digits = 2) {
  return value === null || value === undefined ? null : `$${value.toFixed(digits)}`;
}

export function FuelSurchargeAuditPopover({ detail }: { detail: FuelSurchargeDetail }) {
  const derivation: Array<{ label: string; value: string | null }> = [];

  if (detail.ratePerMile != null) {
    derivation.push(
      { label: "Rate per mile", value: money(detail.ratePerMile, 4) },
      {
        label: "Billed miles",
        value: detail.miles != null ? detail.miles.toFixed(1) : null,
      },
    );
  }
  if (detail.percent != null) {
    const grossBasis = detail.percentBasis === "LinehaulPlusAccessorials";
    derivation.push(
      {
        label: grossBasis ? "Percent of linehaul + accessorials" : "Percent of linehaul",
        value: `${detail.percent.toFixed(2)}%`,
      },
      { label: "Linehaul base", value: money(detail.linehaulBase) },
    );
    if (grossBasis) {
      derivation.push({ label: "Accessorial base", value: money(detail.accessorialBase) });
    }
  }
  if (detail.bandValue != null) {
    derivation.push({
      label: "Matched band",
      value: `${money(detail.bandMin) ?? "Open"} – ${money(detail.bandMax) ?? "Open"}`,
    });
  }
  if (detail.pegPrice != null) {
    derivation.push({ label: "Peg price", value: money(detail.pegPrice, 4) });
  }
  if (detail.increment != null && detail.incrementRate != null) {
    derivation.push({
      label: "Escalator",
      value: `${money(detail.incrementRate, 4)}/mi per ${money(detail.increment)}`,
    });
  }
  if (detail.milesPerGallon != null) {
    derivation.push({ label: "MPG divisor", value: detail.milesPerGallon.toFixed(2) });
  }

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-7"
            title="Fuel surcharge audit detail"
          >
            <FuelIcon className="size-3.5 text-primary" />
          </Button>
        }
      />
      <PopoverContent align="end" className="w-80 space-y-3">
        <div>
          <div className="flex items-center justify-between gap-2">
            <p className="text-sm font-medium">{detail.programName ?? "Fuel Surcharge"}</p>
            <Badge variant="secondary" className="text-2xs">
              {detail.method ?? ""}
            </Badge>
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Every input frozen at rating time — the full defense for a fuel surcharge dispute
          </p>
        </div>

        {(detail.usedFallback || detail.stale) && (
          <div className="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-xs text-amber-700 dark:text-amber-400">
            <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
            <span>
              {detail.stale
                ? "Rated with a price more than 3 weeks old."
                : "Rated before this week's DOE price published — it will re-rate automatically once the price arrives."}
            </span>
          </div>
        )}

        <div className="space-y-1.5">
          <DetailRow
            label="Fuel index"
            value={detail.indexCode ? `${detail.indexCode} (${detail.indexSource ?? ""})` : null}
          />
          <DetailRow label="Region" value={detail.indexRegion ?? null} />
          <DetailRow label="Fuel type" value={detail.indexFuelType ?? null} />
          <DetailRow label="Price week" value={detail.priceDate ?? null} />
          <DetailRow label="Fuel price" value={money(detail.price ?? null, 3)} />
          <DetailRow
            label="Basis"
            value={
              detail.basisDate ? `${detail.basisDate} (${detail.dateBasis ?? ""})` : null
            }
          />
        </div>

        {derivation.length > 0 && (
          <>
            <Separator />
            <div className="space-y-1.5">
              {derivation.map((row) => (
                <DetailRow key={row.label} label={row.label} value={row.value} />
              ))}
            </div>
          </>
        )}

        <Separator />
        <div className="space-y-1.5">
          {detail.rawAmount != null && detail.rawAmount !== detail.amount && (
            <DetailRow label="Before cap/floor" value={money(detail.rawAmount)} />
          )}
          {detail.capApplied && <DetailRow label="Cap applied" value="Yes" />}
          {detail.floorApplied && <DetailRow label="Floor applied" value="Yes" />}
          <div className="flex items-center justify-between gap-4 text-sm">
            <span className="font-medium">Surcharge</span>
            <span className="font-semibold tabular-nums">{money(detail.amount ?? null)}</span>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
