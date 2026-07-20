import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { FuelSurchargeProgramFormValues } from "@/types/fuel-surcharge";
import { useQuery } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowDownUp, CircleAlert, Plus, Table2, TriangleAlert, Trash2, Wand2 } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { memo, useCallback, useDeferredValue, useMemo, useRef, useState } from "react";
import {
  Controller,
  useFieldArray,
  useFormContext,
  useFormState,
  useWatch,
  type Control,
  type FieldPath,
} from "react-hook-form";
import { NumericFormat } from "react-number-format";
import { GenerateTableDialog } from "./generate-table-dialog";

const ROW_HEIGHT = 40;

type BandValueMeta = {
  header: string;
  prefix?: string;
  suffix?: string;
  decimalScale: number;
  readingHint: string;
};

function valueMetaForMethod(method: string): BandValueMeta {
  switch (method) {
    case "TablePercent":
      return {
        header: "% of Charge",
        suffix: "%",
        decimalScale: 2,
        readingHint: "add that percentage of the freight charge",
      };
    case "TableFlat":
      return {
        header: "Flat Amount",
        prefix: "$",
        decimalScale: 2,
        readingHint: "add that flat dollar amount",
      };
    default:
      return {
        header: "Rate per Mile",
        prefix: "$",
        decimalScale: 4,
        readingHint: "charge that rate for every mile",
      };
  }
}

type WatchedRow = {
  priceMin?: number | null;
  priceMax?: number | null;
  value?: number | null;
};

function toNumber(value: number | null | undefined): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function roundPrice(value: number) {
  return Math.round(value * 10000) / 10000;
}

function money(value: number, digits = 2) {
  return `$${value.toFixed(digits)}`;
}

type BandIssue = {
  severity: "error" | "warning";
  message: string;
  fillGap?: { afterIndex: number; from: number; to: number };
};

function computeBandIssues(rows: WatchedRow[]): BandIssue[] {
  const issues: BandIssue[] = [];

  rows.forEach((row, index) => {
    const min = toNumber(row.priceMin);
    const max = toNumber(row.priceMax);
    if (min !== null && max !== null && max <= min) {
      issues.push({
        severity: "error",
        message: `Band ${index + 1}: the "up to" price must be higher than the "from" price.`,
      });
    }
  });

  const sorted = rows
    .map((row, index) => ({ row, index }))
    .filter(({ row }) => toNumber(row.priceMin) !== null)
    .sort((a, b) => (toNumber(a.row.priceMin) ?? 0) - (toNumber(b.row.priceMin) ?? 0));

  for (let i = 1; i < sorted.length; i++) {
    const prev = sorted[i - 1];
    const curr = sorted[i];
    const prevMax = toNumber(prev.row.priceMax);
    const currMin = toNumber(curr.row.priceMin);
    if (currMin === null) continue;

    if (prevMax === null || currMin < prevMax) {
      issues.push({
        severity: "error",
        message:
          prevMax === null
            ? `Bands ${prev.index + 1} and ${curr.index + 1} overlap — band ${prev.index + 1} has no upper limit.`
            : `Bands ${prev.index + 1} and ${curr.index + 1} overlap — a price like ${money(currMin)} would match both.`,
      });
    } else if (currMin > prevMax) {
      issues.push({
        severity: "warning",
        message: `Prices from ${money(prevMax)} to ${money(currMin)} aren't covered — no surcharge would apply there.`,
        fillGap: { afterIndex: prev.index, from: prevMax, to: currMin },
      });
    }
  }

  return issues;
}

function isSortedByPrice(rows: WatchedRow[]): boolean {
  let last: number | null = null;
  for (const row of rows) {
    const min = toNumber(row.priceMin);
    if (min === null) continue;
    if (last !== null && min < last) return false;
    last = min;
  }
  return true;
}

function bandContainsPrice(row: WatchedRow, price: number): boolean {
  const min = toNumber(row.priceMin);
  const max = toNumber(row.priceMax);
  if (min !== null && price < min) return false;
  if (max !== null && price >= max) return false;
  return min !== null || max !== null;
}

type BandCellProps = {
  control: Control<FuelSurchargeProgramFormValues>;
  name: FieldPath<FuelSurchargeProgramFormValues>;
  placeholder: string;
  ariaLabel: string;
  prefix?: string;
  suffix?: string;
  decimalScale: number;
  disabled?: boolean;
};

function BandCell({
  control,
  name,
  placeholder,
  ariaLabel,
  prefix,
  suffix,
  decimalScale,
  disabled,
}: BandCellProps) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <NumericFormat
          value={typeof field.value === "number" ? field.value : ""}
          onValueChange={(values) => field.onChange(values.floatValue ?? null)}
          onBlur={field.onBlur}
          getInputRef={field.ref}
          decimalScale={decimalScale}
          allowNegative={false}
          prefix={prefix}
          suffix={suffix}
          placeholder={placeholder}
          aria-label={ariaLabel}
          disabled={disabled}
          className={cn(
            "h-8 w-full rounded-md border border-transparent bg-transparent px-2.5 text-sm tabular-nums outline-none",
            "placeholder:text-muted-foreground/50",
            "transition-[border-color,box-shadow,background-color] duration-150 ease-in-out",
            "hover:bg-muted/70",
            "focus-visible:border-brand focus-visible:bg-background focus-visible:ring-4 focus-visible:ring-brand/20",
            "disabled:cursor-not-allowed disabled:opacity-50",
            fieldState.invalid &&
              "border-red-500/60 bg-red-500/10 focus-visible:border-red-500 focus-visible:ring-red-400/20",
          )}
        />
      )}
    />
  );
}

type BandRowProps = {
  control: Control<FuelSurchargeProgramFormValues>;
  index: number;
  meta: BandValueMeta;
  currentPrice: number | null;
  disabled?: boolean;
  onInsertAfter: (index: number) => void;
  onRemove: (index: number) => void;
};

const BandRow = memo(function BandRow({
  control,
  index,
  meta,
  currentPrice,
  disabled,
  onInsertAfter,
  onRemove,
}: BandRowProps) {
  const row = useWatch({ control, name: `tableRows.${index}` }) as WatchedRow | undefined;
  const isCurrent = currentPrice !== null && !!row && bandContainsPrice(row, currentPrice);

  return (
    <tr
      style={{ height: ROW_HEIGHT }}
      className={cn(
        "group border-t transition-colors hover:bg-muted/40",
        isCurrent && "bg-primary/5",
      )}
    >
      <td className="px-3 text-center text-xs text-muted-foreground tabular-nums">
        {isCurrent ? (
          <span
            className="inline-block size-2 rounded-full bg-primary"
            title="This week's price falls in this band"
          />
        ) : (
          index + 1
        )}
      </td>
      <td className="px-1">
        <BandCell
          control={control}
          name={`tableRows.${index}.priceMin`}
          placeholder="Any price"
          ariaLabel={`Band ${index + 1} price from`}
          prefix="$"
          decimalScale={4}
          disabled={disabled}
        />
      </td>
      <td className="px-1">
        <BandCell
          control={control}
          name={`tableRows.${index}.priceMax`}
          placeholder="No limit"
          ariaLabel={`Band ${index + 1} price up to`}
          prefix="$"
          decimalScale={4}
          disabled={disabled}
        />
      </td>
      <td className="px-1">
        <BandCell
          control={control}
          name={`tableRows.${index}.value`}
          placeholder="0.00"
          ariaLabel={`Band ${index + 1} ${meta.header}`}
          prefix={meta.prefix}
          suffix={meta.suffix}
          decimalScale={meta.decimalScale}
          disabled={disabled}
        />
      </td>
      <td className="px-2">
        <div className="flex justify-end gap-0.5 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onInsertAfter(index)}
            disabled={disabled}
            title="Insert a band below this one"
            className="size-7 p-0 text-muted-foreground hover:text-foreground"
          >
            <Plus className="size-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onRemove(index)}
            disabled={disabled}
            title="Delete this band"
            className="size-7 p-0 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </td>
    </tr>
  );
});

function SortButton({
  control,
  disabled,
  onSort,
}: {
  control: Control<FuelSurchargeProgramFormValues>;
  disabled?: boolean;
  onSort: () => void;
}) {
  const watched = useWatch({ control, name: "tableRows" });
  const rows = useDeferredValue(watched);
  const sorted = useMemo(() => isSortedByPrice(rows ?? []), [rows]);

  if (sorted) return null;

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={onSort}
      disabled={disabled}
      className="gap-1.5"
    >
      <ArrowDownUp className="size-3.5" />
      Sort by Price
    </Button>
  );
}

function RowsErrorBanner({ control }: { control: Control<FuelSurchargeProgramFormValues> }) {
  const { errors } = useFormState({ control, name: "tableRows" });
  const message = errors.tableRows?.message ?? errors.tableRows?.root?.message;

  if (!message) return null;

  return (
    <p className="flex items-center gap-1.5 border-b bg-red-500/5 px-4 py-2 text-xs text-red-500">
      <CircleAlert className="size-3.5 shrink-0" />
      {message}
    </p>
  );
}

function IssuesStrip({
  control,
  disabled,
  onFillGap,
}: {
  control: Control<FuelSurchargeProgramFormValues>;
  disabled?: boolean;
  onFillGap: (gap: { afterIndex: number; from: number; to: number }) => void;
}) {
  const watched = useWatch({ control, name: "tableRows" });
  const rows = useDeferredValue(watched);
  const issues = useMemo(() => computeBandIssues(rows ?? []), [rows]);

  return (
    <AnimatePresence initial={false}>
      {issues.slice(0, 3).map((issue) => (
        <m.div
          key={issue.message}
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.15 }}
          className="overflow-hidden border-t"
        >
          <div
            className={cn(
              "flex items-center justify-between gap-3 px-4 py-2 text-xs",
              issue.severity === "error"
                ? "bg-red-500/5 text-red-600 dark:text-red-400"
                : "bg-amber-500/5 text-amber-600 dark:text-amber-400",
            )}
          >
            <span className="flex items-center gap-1.5">
              {issue.severity === "error" ? (
                <CircleAlert className="size-3.5 shrink-0" />
              ) : (
                <TriangleAlert className="size-3.5 shrink-0" />
              )}
              {issue.message}
            </span>
            {issue.fillGap && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => onFillGap(issue.fillGap!)}
                disabled={disabled}
                className="h-6 shrink-0 px-2 text-xs"
              >
                Fill Gap
              </Button>
            )}
          </div>
        </m.div>
      ))}
    </AnimatePresence>
  );
}

function FooterSummary({
  control,
  currentPrice,
}: {
  control: Control<FuelSurchargeProgramFormValues>;
  currentPrice: number | null;
}) {
  const watched = useWatch({ control, name: "tableRows" });
  const deferred = useDeferredValue(watched);
  const rows = useMemo<WatchedRow[]>(() => deferred ?? [], [deferred]);

  const coverage = useMemo(() => {
    if (rows.length === 0) return null;
    const mins = rows.map((row) => toNumber(row.priceMin));
    const maxes = rows.map((row) => toNumber(row.priceMax));
    const openBottom = mins.some((value) => value === null);
    const openTop = maxes.some((value) => value === null);
    const finiteMins = mins.filter((value): value is number => value !== null);
    const finiteMaxes = maxes.filter((value): value is number => value !== null);
    const low = openBottom ? null : finiteMins.length ? Math.min(...finiteMins) : null;
    const high = openTop ? null : finiteMaxes.length ? Math.max(...finiteMaxes) : null;
    if (low === null && high === null) return "covers every fuel price";
    if (low === null) return `covers every price up to ${money(high as number)}`;
    if (high === null) return `covers ${money(low)} and up`;
    return `covers ${money(low)} to ${money(high)}`;
  }, [rows]);

  const uncovered = useMemo(() => {
    if (currentPrice === null || rows.length === 0) return false;
    return !rows.some((row) => bandContainsPrice(row, currentPrice));
  }, [rows, currentPrice]);

  return (
    <div className="flex items-center justify-between border-t px-4 py-2 text-xs text-muted-foreground">
      <span>
        {rows.length} {rows.length === 1 ? "band" : "bands"}
        {coverage ? ` · ${coverage}` : ""}
      </span>
      {currentPrice !== null && (
        <span>
          This week&apos;s price:{" "}
          <span className="font-medium tabular-nums">{money(currentPrice, 3)}</span>
          {uncovered && (
            <span className="ml-1 text-amber-600 dark:text-amber-400">— no band covers it</span>
          )}
        </span>
      )}
    </div>
  );
}

export function BandTableEditor({ method, disabled }: { method: string; disabled?: boolean }) {
  const { control, getValues } = useFormContext<FuelSurchargeProgramFormValues>();
  const { fields, append, remove, insert, replace } = useFieldArray({
    control,
    name: "tableRows",
  });
  const fuelIndexId = useWatch({ control, name: "fuelIndexId" });
  const [wizardOpen, setWizardOpen] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const meta = useMemo(() => valueMetaForMethod(method), [method]);

  const { data: dashboard } = useQuery(queries.fuelSurcharge.dashboard());
  const currentPrice = useMemo(() => {
    if (!fuelIndexId || !dashboard) return null;
    const entry = dashboard.find((item) => item.index.id === fuelIndexId);
    return entry?.latest ? Number(entry.latest.price) : null;
  }, [dashboard, fuelIndexId]);

  const virtualizer = useVirtualizer({
    count: fields.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  });

  const readRows = useCallback(
    (): WatchedRow[] => (getValues("tableRows") as WatchedRow[] | undefined) ?? [],
    [getValues],
  );

  const scrollToRow = useCallback(
    (index: number) => {
      requestAnimationFrame(() => virtualizer.scrollToIndex(index, { align: "end" }));
    },
    [virtualizer],
  );

  const handleAdd = useCallback(() => {
    const rows = readRows();
    const last = rows[rows.length - 1];
    const prev = rows[rows.length - 2];

    if (!last) {
      append({
        priceMin: null,
        priceMax: null,
        value: undefined as unknown as number,
        sortOrder: 0,
      });
      return;
    }

    const lastMin = toNumber(last.priceMin);
    const lastMax = toNumber(last.priceMax);
    const width =
      lastMin !== null && lastMax !== null && lastMax > lastMin ? lastMax - lastMin : 0.05;

    const lastValue = toNumber(last.value);
    const prevValue = prev ? toNumber(prev.value) : null;
    const valueStep = lastValue !== null && prevValue !== null ? lastValue - prevValue : 0;

    append({
      priceMin: lastMax,
      priceMax: lastMax !== null ? roundPrice(lastMax + width) : null,
      value:
        lastValue !== null
          ? (roundPrice(lastValue + valueStep) as number)
          : (undefined as unknown as number),
      sortOrder: rows.length,
    });
    scrollToRow(rows.length);
  }, [readRows, append, scrollToRow]);

  const handleInsertAfter = useCallback(
    (index: number) => {
      const rows = readRows();
      const row = rows[index];
      const next = rows[index + 1];
      insert(index + 1, {
        priceMin: toNumber(row?.priceMax),
        priceMax: next ? toNumber(next.priceMin) : null,
        value: toNumber(row?.value) as number,
        sortOrder: index + 1,
      });
    },
    [readRows, insert],
  );

  const handleFillGap = useCallback(
    (gap: { afterIndex: number; from: number; to: number }) => {
      const rows = readRows();
      insert(gap.afterIndex + 1, {
        priceMin: gap.from,
        priceMax: gap.to,
        value: toNumber(rows[gap.afterIndex]?.value) as number,
        sortOrder: gap.afterIndex + 1,
      });
      scrollToRow(gap.afterIndex + 1);
    },
    [readRows, insert, scrollToRow],
  );

  const handleSort = useCallback(() => {
    const resorted = [...readRows()]
      .map((row, index) => ({ row, index }))
      .sort((a, b) => {
        const aMin = toNumber(a.row.priceMin);
        const bMin = toNumber(b.row.priceMin);
        if (aMin === null && bMin === null) return a.index - b.index;
        if (aMin === null) return -1;
        if (bMin === null) return 1;
        return aMin - bMin;
      })
      .map(({ row }, index) => ({
        priceMin: toNumber(row.priceMin),
        priceMax: toNumber(row.priceMax),
        value: toNumber(row.value) as number,
        sortOrder: index,
      }));
    replace(resorted);
  }, [readRows, replace]);

  const handleGenerated = useCallback(
    (generated: Array<{ priceMin?: string | null; priceMax?: string | null; value: string }>) => {
      replace(
        generated.map((row, index) => ({
          priceMin: row.priceMin != null ? Number(row.priceMin) : null,
          priceMax: row.priceMax != null ? Number(row.priceMax) : null,
          value: Number(row.value),
          sortOrder: index,
        })),
      );
      requestAnimationFrame(() => virtualizer.scrollToIndex(0));
    },
    [replace, virtualizer],
  );

  const virtualItems = virtualizer.getVirtualItems();
  const paddingTop = virtualItems.length > 0 ? virtualItems[0].start : 0;
  const paddingBottom =
    virtualItems.length > 0
      ? virtualizer.getTotalSize() - virtualItems[virtualItems.length - 1].end
      : 0;

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <div className="flex items-center gap-2">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
            <Table2 className="size-4 text-primary" />
          </div>
          <div>
            <CardTitle className="text-sm font-medium">Price Band Table</CardTitle>
            <p className="text-xs text-muted-foreground">
              Read each row as: when fuel costs at least &ldquo;from&rdquo; and less than &ldquo;up
              to&rdquo;, {meta.readingHint}.
            </p>
          </div>
        </div>
        {fields.length > 0 && (
          <div className="flex gap-2">
            <SortButton control={control} disabled={disabled} onSort={handleSort} />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setWizardOpen(true)}
              disabled={disabled}
              className="gap-1.5"
            >
              <Wand2 className="size-3.5" />
              Generate
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleAdd}
              disabled={disabled}
              className="gap-1.5"
            >
              <Plus className="size-3.5" />
              Add Band
            </Button>
          </div>
        )}
      </CardHeader>
      <CardContent className="p-0">
        <RowsErrorBanner control={control} />
        {fields.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-6 py-10 text-center">
            <div className="flex size-12 items-center justify-center rounded-full bg-muted">
              <Table2 className="size-5 text-muted-foreground" />
            </div>
            <p className="mt-3 text-sm font-medium">Build your price band table</p>
            <p className="mt-1 max-w-sm text-xs text-muted-foreground">
              The fastest way is the generator — set a price range and increments, and the full
              table is built for you. Every row stays editable afterward, so uneven bands are fine.
            </p>
            <div className="mt-4 flex gap-2">
              <Button
                type="button"
                size="sm"
                onClick={() => setWizardOpen(true)}
                disabled={disabled}
                className="gap-1.5"
              >
                <Wand2 className="size-3.5" />
                Generate the Table for Me
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAdd}
                disabled={disabled}
                className="gap-1.5"
              >
                <Plus className="size-3.5" />
                Start From Scratch
              </Button>
            </div>
          </div>
        ) : (
          <>
            <div ref={scrollRef} className="max-h-96 overflow-y-auto">
              <table className="w-full text-sm">
                <thead className="sticky top-0 z-10 bg-muted/80 backdrop-blur">
                  <tr className="text-left text-xs text-muted-foreground">
                    <th className="w-10 px-3 py-2 text-center font-medium">#</th>
                    <th className="px-2 py-2 font-medium">Fuel Price From</th>
                    <th className="px-2 py-2 font-medium">Up To (not incl.)</th>
                    <th className="px-2 py-2 font-medium">{meta.header}</th>
                    <th className="w-16 px-2 py-2" />
                  </tr>
                </thead>
                <tbody>
                  {paddingTop > 0 && <tr style={{ height: paddingTop }} aria-hidden />}
                  {virtualItems.map((item) => (
                    <BandRow
                      key={fields[item.index].id}
                      control={control}
                      index={item.index}
                      meta={meta}
                      currentPrice={currentPrice}
                      disabled={disabled}
                      onInsertAfter={handleInsertAfter}
                      onRemove={remove}
                    />
                  ))}
                  {paddingBottom > 0 && <tr style={{ height: paddingBottom }} aria-hidden />}
                </tbody>
              </table>
            </div>
            <IssuesStrip control={control} disabled={disabled} onFillGap={handleFillGap} />
            <FooterSummary control={control} currentPrice={currentPrice} />
          </>
        )}
      </CardContent>
      <GenerateTableDialog
        open={wizardOpen}
        onOpenChange={setWizardOpen}
        valueMeta={{
          label: meta.header,
          prefix: meta.prefix,
          suffix: meta.suffix,
          decimalScale: meta.decimalScale,
        }}
        replaceCount={fields.length}
        onApply={handleGenerated}
      />
    </Card>
  );
}
