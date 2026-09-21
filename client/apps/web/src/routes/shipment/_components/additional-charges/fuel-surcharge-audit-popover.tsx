import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import type { FuelSurchargeDetail } from "@trenova/shared/types/shipment";
import { AlertTriangle, FuelIcon } from "lucide-react";

function DetailRow({ label, value }: { label: string; value: string | null }) {
  if (value === null) return null;
  return (
    <DescriptionItem label={label} numeric>
      {value}
    </DescriptionItem>
  );
}

function money(value: number | null | undefined, digits = 2) {
  return value === null || value === undefined ? null : `$${value.toFixed(digits)}`;
}

export function FuelSurchargeAuditPopover({ detail }: { detail: FuelSurchargeDetail }) {
  const t = useT();

  const derivation: Array<{ label: string; value: string | null }> = [];

  if (detail.ratePerMile != null) {
    derivation.push(
      { label: t("Rate per mile"), value: money(detail.ratePerMile, 4) },
      {
        label: t("Billed miles"),
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
      { label: t("Linehaul base"), value: money(detail.linehaulBase) },
    );
    if (grossBasis) {
      derivation.push({ label: t("Accessorial base"), value: money(detail.accessorialBase) });
    }
  }
  if (detail.bandValue != null) {
    derivation.push({
      label: t("Matched band"),
      value: `${money(detail.bandMin) ?? "Open"} – ${money(detail.bandMax) ?? "Open"}`,
    });
  }
  if (detail.pegPrice != null) {
    derivation.push({ label: t("Peg price"), value: money(detail.pegPrice, 4) });
  }
  if (detail.increment != null && detail.incrementRate != null) {
    derivation.push({
      label: t("Escalator"),
      value: `${money(detail.incrementRate, 4)}/mi per ${money(detail.increment)}`,
    });
  }
  if (detail.milesPerGallon != null) {
    derivation.push({ label: t("MPG divisor"), value: detail.milesPerGallon.toFixed(2) });
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
            title={t("Fuel surcharge audit detail")}
          >
            <FuelIcon className="text-primary size-3.5" />
          </Button>
        }
      />
      <PopoverContent align="end" className="w-80 space-y-3">
        <div>
          <div className="flex items-center justify-between gap-2">
            <p className="text-sm font-medium">{detail.programName ?? t("Fuel Surcharge")}</p>
            <Badge variant="neutral" className="text-2xs">
              {detail.method ?? ""}
            </Badge>
          </div>
          <p className="text-muted-foreground mt-0.5 text-xs">
            {t("Every input frozen at rating time — the full defense for a fuel surcharge dispute")}
          </p>
        </div>

        {(detail.usedFallback || detail.stale) && (
          <Alert variant="warning" size="sm">
            <AlertTriangle />
            <AlertDescription>
              {detail.stale
                ? t("Rated with a price more than 3 weeks old.")
                : t(
                    "Rated before this week's DOE price published — it will re-rate automatically once the price arrives.",
                  )}
            </AlertDescription>
          </Alert>
        )}

        <DescriptionList layout="split">
          <DetailRow
            label={t("Fuel index")}
            value={detail.indexCode ? `${detail.indexCode} (${detail.indexSource ?? ""})` : null}
          />
          <DetailRow label={t("Region")} value={detail.indexRegion ?? null} />
          <DetailRow label={t("Fuel type")} value={detail.indexFuelType ?? null} />
          <DetailRow label={t("Price week")} value={detail.priceDate ?? null} />
          <DetailRow label={t("Fuel price")} value={money(detail.price ?? null, 3)} />
          <DetailRow
            label={t("Basis")}
            value={detail.basisDate ? `${detail.basisDate} (${detail.dateBasis ?? ""})` : null}
          />
        </DescriptionList>

        {derivation.length > 0 && (
          <DescriptionList layout="split">
            {derivation.map((row) => (
              <DetailRow key={row.label} label={t(row.label)} value={row.value} />
            ))}
          </DescriptionList>
        )}

        <DescriptionList layout="split">
          {detail.rawAmount != null && detail.rawAmount !== detail.amount && (
            <DetailRow label={t("Before cap/floor")} value={money(detail.rawAmount)} />
          )}
          {detail.capApplied && <DetailRow label={t("Cap applied")} value="Yes" />}
          {detail.floorApplied && <DetailRow label={t("Floor applied")} value="Yes" />}
          <DescriptionItem label={t("Surcharge")} numeric valueClassName="font-semibold">
            {money(detail.amount ?? null)}
          </DescriptionItem>
        </DescriptionList>
      </PopoverContent>
    </Popover>
  );
}
