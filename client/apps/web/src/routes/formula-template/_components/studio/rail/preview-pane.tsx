import { ControlledShipmentAutocompleteField } from "@/components/autocomplete-fields";
import { ReceiptView } from "@/components/formula-editor/receipt-view";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import { previewFailure } from "@/lib/formula-preview";
import { translateMessage } from "@trenova/shared/i18n/message";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FormulaTemplateFormValues,
  FormulaValueSource,
} from "@trenova/shared/types/formula-template";
import {
  Beaker02Icon,
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  Pin01Icon,
  Shield01Icon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { formatMoney, formatStudioValue, previewAmounts, roundingLabel } from "../model/format";
import { useStudio } from "../studio-context";
import type { PreviewSource } from "../use-live-preview";
import { ValueSourceTag } from "./value-source";

function parseTyped(raw: string, type: string): unknown {
  if (raw === "") return null;
  if (["number", "integer", "float"].includes(type.toLowerCase())) {
    const amount = Number(raw);
    return Number.isFinite(amount) && !raw.endsWith(".") ? amount : raw;
  }
  return raw;
}

function ValueEditor({
  name,
  type,
  value,
  onChange,
}: {
  name: string;
  type: string;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const t = useT();
  if (type.toLowerCase() === "boolean" || typeof value === "boolean") {
    return (
      <span className="flex items-center gap-1.5">
        <Switch
          size="sm"
          checked={value === true}
          onCheckedChange={(checked) => onChange(checked)}
          aria-label={name}
        />
        <span className="text-foreground-muted font-mono text-2xs">
          {value === true ? "true" : "false"}
        </span>
      </span>
    );
  }
  return (
    <Input
      value={typeof value === "string" || typeof value === "number" ? String(value) : ""}
      onChange={(event) => onChange(parseTyped(event.target.value, type))}
      inputMode={type.toLowerCase() === "number" ? "decimal" : undefined}
      aria-label={name}
      placeholder={t("empty")}
      inputContainerClassName="w-28"
      className="h-6 text-right font-mono text-xs md:text-xs"
    />
  );
}

/**
 * The preview: a load, what the editor bills it, and the values it read.
 * The load is the sample, with any value the person types over it, or a real
 * shipment. Every amount here is the engine's answer for the editor as it is.
 */
export function PreviewPane() {
  const t = useT();
  const { preview, analysis, known, scenarios, setTab } = useStudio();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const variableDefinitions = useWatch({ control, name: "variableDefinitions" }) ?? [];
  const roundingMode = useWatch({ control, name: "roundingMode" });
  const roundingPrecision = useWatch({ control, name: "roundingPrecision" }) ?? 2;
  const [showAll, setShowAll] = useState(false);
  const [receiptOpen, setReceiptOpen] = useState(false);
  const [pinnedKey, setPinnedKey] = useState<string | null>(null);

  const result = preview.result;
  const failure = previewFailure(result);
  const amounts = previewAmounts(result);
  const usingShipment = preview.source === "shipment" && preview.shipmentId !== "";
  const durationMicros = result?.receipt?.durationMicros;

  const docs = useMemo(
    () => new Map(known.variables.map((variable) => [variable.name, variable])),
    [known.variables],
  );
  const receiptSources = useMemo(
    () =>
      new Map(
        (result?.receipt?.variables ?? []).map((variable) => [variable.name, variable.source]),
      ),
    [result?.receipt?.variables],
  );
  const customNames = new Set(variableDefinitions.map((variable) => variable.name));

  const sourceOf = (name: string): FormulaValueSource => {
    if (name in preview.overrides) return usingShipment ? "override" : "input";
    const read = receiptSources.get(name);
    if (read) return read;
    if (customNames.has(name)) return "default";
    if (usingShipment) return docs.get(name)?.category === "computed" ? "computed" : "field";
    return "sample";
  };

  const reads = analysis.reads;
  const schemaNames = known.variables.filter((variable) => !variable.custom).map((v) => v.name);
  const rest = schemaNames.filter((name) => !reads.includes(name));
  const shown = showAll ? [...reads, ...rest] : reads;
  const overrideCount = Object.keys(preview.overrides).length;
  const overrides = preview.overrides;
  const valueOf = preview.valueOf;
  const setOverride = preview.setOverride;

  // The list is virtualized: with every shipment variable shown it runs to
  // a few hundred rows, each with a field of its own.
  const valueRows = useMemo(
    (): VirtualRow[] =>
      shown.map((name) => {
        const doc = docs.get(name);
        const type = doc?.type ?? "Number";
        return {
          key: name,
          render: () => (
            <div
              className={cn(
                "flex items-center gap-2 border-b py-1",
                name in overrides && "bg-warning-subtle -mx-1.5 rounded-sm px-1.5",
              )}
            >
              <span className="min-w-0 flex-1 truncate font-mono text-xs" title={doc?.description}>
                {name}
              </span>
              <ValueEditor
                name={name}
                type={type}
                value={valueOf(name)}
                onChange={(value) => setOverride(name, value)}
              />
              <ValueSourceTag source={sourceOf(name)} />
            </div>
          ),
        };
      }),
    // sourceOf reads only what the row's props already carry, so it is not a dependency of its own.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
    [shown, docs, overrides, valueOf, setOverride, receiptSources, customNames, usingShipment],
  );

  const pinKey = amounts
    ? JSON.stringify([preview.source, preview.shipmentId, preview.overrides, amounts.billed])
    : null;
  const pinned = pinKey !== null && pinKey === pinnedKey;

  const pin = async () => {
    if (!amounts || !pinKey) return;
    const overrides = Object.entries(preview.overrides).slice(0, 2);
    const name = usingShipment
      ? [preview.valueOf("proNumber"), preview.valueOf("origin.city")]
          .filter(Boolean)
          .map(formatStudioValue)
          .join(" · ") || t("Picked shipment")
      : overrides.length > 0
        ? t(
            "Sample load · {0}",
            overrides.map(([key, value]) => `${key} ${formatStudioValue(value)}`).join(", "),
          )
        : t(
            "Sample load, {0} mi · {1} stops",
            formatStudioValue(preview.valueOf("totalDistance")),
            formatStudioValue(preview.valueOf("totalStops")),
          );
    try {
      await scenarios.add([
        {
          name: name.slice(0, 100),
          variables: preview.loadValues(),
          expectedAmount: amounts.billed,
        },
      ]);
      setPinnedKey(pinKey);
      setTab("scenarios");
    } catch (error) {
      toast.error(t("Could not pin the scenario"), {
        description: error instanceof Error ? error.message : undefined,
      });
    }
  };

  return (
    <div className="flex flex-col gap-3 p-3">
      <div className="flex items-center gap-2">
        <SegmentedControl<PreviewSource>
          aria-label={t("Preview load")}
          value={preview.source}
          onValueChange={preview.setSource}
          items={[
            { value: "sample", label: t("Sample load") },
            { value: "shipment", label: t("Real shipment") },
          ]}
        />
        <span className="flex-1" />
        {durationMicros !== undefined && (
          <span className="text-foreground-subtle font-mono text-2xs tabular-nums">
            {durationMicros < 1000 ? "<1" : (durationMicros / 1000).toFixed(1)} ms
          </span>
        )}
      </div>

      {preview.source === "shipment" && (
        <ControlledShipmentAutocompleteField
          label=""
          value={preview.shipmentId}
          onValueChange={preview.setShipmentId}
          clearable
        />
      )}

      {preview.requestError && (
        <Alert variant="destructive" size="sm">
          <AlertTitle>{t("Preview could not run")}</AlertTitle>
          <AlertDescription>{preview.requestError}</AlertDescription>
          <AlertAction>
            <Button type="button" variant="outline" size="xs" onClick={preview.runNow}>
              {t("Retry")}
            </Button>
          </AlertAction>
        </Alert>
      )}

      <div
        aria-busy={preview.isPending || undefined}
        className={cn(
          "bg-card rounded-lg border",
          result && !result.valid && "border-danger-border",
          preview.isPending && "opacity-80",
        )}
      >
        {!result ? (
          <div className="text-foreground-muted flex flex-col items-center gap-2 px-4 py-8 text-center text-sm">
            <Beaker02Icon className="text-foreground-subtle size-5" />
            {preview.source === "shipment" && !preview.shipmentId
              ? t("Pick a shipment to price it with the editor's formula.")
              : t("Write an expression and the charge appears here as you type.")}
          </div>
        ) : amounts ? (
          <div className="flex flex-col gap-2 p-3.5">
            <div className="flex items-center gap-2">
              <span className="text-foreground-muted text-xs">{t("Billed")}</span>
              <span className="flex-1" />
              <Button
                type="button"
                variant="ghost"
                size="xs"
                className="gap-1"
                disabled={pinned}
                onClick={() => void pin()}
              >
                {pinned ? <CheckIcon className="size-3" /> : <Pin01Icon className="size-3" />}
                {pinned ? t("Pinned") : t("Pin as scenario")}
              </Button>
            </div>
            <span key={amounts.billed} className="fs-settle font-mono text-4xl tabular-nums">
              {formatMoney(
                amounts.billed,
                roundingMode === "None" ? 4 : Math.max(roundingPrecision, 0),
              )}
            </span>
            <div className="flex flex-col gap-0.5 text-xs">
              {amounts.bound && amounts.limit !== null ? (
                <span className="text-warning flex items-center gap-1.5">
                  <Shield01Icon className="size-3" />
                  {amounts.bound === "min"
                    ? t(
                        "Formula produced {0}; raised to the {1} minimum",
                        formatMoney(amounts.raw),
                        formatMoney(amounts.limit),
                      )
                    : t(
                        "Formula produced {0}; lowered to the {1} maximum",
                        formatMoney(amounts.raw),
                        formatMoney(amounts.limit),
                      )}
                </span>
              ) : (
                <span className="text-success flex items-center gap-1.5">
                  <Shield01Icon className="size-3" />
                  {t("Within the charge policy")}
                </span>
              )}
              <span className="text-foreground-subtle">
                {roundingMode === "None"
                  ? t("Not rounded")
                  : amounts.rounded
                    ? t(
                        "Rounded {0} from {1}",
                        roundingLabel(roundingMode, t),
                        formatStudioValue(result.rounding?.unroundedAmount),
                      )
                    : t("Rounding made no change")}
              </span>
            </div>

            {(result.breakdown?.length ?? 0) > 0 && (
              <div className="mt-1 flex flex-col border-t pt-2">
                {(result.breakdown ?? []).map((line) => (
                  <div
                    key={line.name}
                    className="flex items-center justify-between gap-2 py-0.5 text-xs"
                  >
                    <span className="text-foreground-muted truncate">
                      {line.label || line.name}
                    </span>
                    <span
                      className={cn("font-mono tabular-nums", line.error && "text-danger")}
                      title={line.error}
                    >
                      {line.error ? t("Fails") : formatMoney(line.amount)}
                    </span>
                  </div>
                ))}
              </div>
            )}

            {result.receipt && (
              <div className="flex flex-col gap-1.5 border-t pt-2">
                <button
                  type="button"
                  onClick={() => setReceiptOpen((open) => !open)}
                  aria-expanded={receiptOpen}
                  className="ui-focus-ring text-foreground-muted hover:text-foreground flex items-center gap-1 self-start rounded-sm text-xs"
                >
                  {receiptOpen ? (
                    <ChevronDownIcon className="size-3" />
                  ) : (
                    <ChevronRightIcon className="size-3" />
                  )}
                  {t(
                    "Receipt · {0, plural, one {# value read} other {# values read}}",
                    result.receipt.variables.length,
                  )}
                </button>
                {receiptOpen && (
                  <ScrollArea maskVariant="card" className="max-h-64">
                    <ReceiptView receipt={result.receipt} />
                  </ScrollArea>
                )}
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-start gap-2 p-3.5">
            <XCircleIcon className="text-danger mt-0.5 size-4 shrink-0" />
            <div className="flex min-w-0 flex-col gap-1">
              <span className="text-sm font-medium">
                {result.valid
                  ? t("Not a charge")
                  : analysis.errorCount > 0
                    ? t("The expression has errors")
                    : t("Fails on this load")}
              </span>
              <p className="text-foreground-muted font-mono text-xs break-words whitespace-pre-wrap">
                {result.valid
                  ? t(
                      "The formula returned {0}. It needs to return a number.",
                      formatStudioValue(result.result),
                    )
                  : failure && translateMessage(failure, t)}
              </p>
            </div>
          </div>
        )}
      </div>

      <div className="flex items-center gap-2 pt-1">
        <span className="text-sm font-semibold">
          {showAll ? t("All variables") : t("Values this formula reads")}
        </span>
        <span className="flex-1" />
        {overrideCount > 0 && (
          <Button type="button" variant="link" size="xs" onClick={preview.resetOverrides}>
            {t("Reset {0}", overrideCount)}
          </Button>
        )}
      </div>
      <VirtualRows
        rows={valueRows}
        estimateSize={33}
        initialHeight={480}
        className="max-h-120"
        aria-label={showAll ? t("All variables") : t("Values this formula reads")}
        empty={
          <p className="text-foreground-muted text-xs">
            {t("Nothing yet. Values appear here as the expression names them.")}
          </p>
        }
      />
      <Button
        type="button"
        variant="quiet"
        size="xs"
        className="self-start"
        onClick={() => setShowAll((all) => !all)}
      >
        {showAll
          ? t("Show only what the formula reads")
          : t("Show all {0} shipment variables", schemaNames.length)}
      </Button>
    </div>
  );
}
