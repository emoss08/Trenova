import {
  NumberFieldGroup,
  NumberFieldInput,
  NumberField as NumberFieldRoot,
} from "@/components/ui/number-field";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn, formatCurrency } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  BacktestResult,
  BacktestSummary,
  FormulaTemplate,
  FormulaTemplateFormValues,
} from "@/types/formula-template";
import { useMutation } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  ArrowDownIcon,
  ArrowRightIcon,
  ArrowUpIcon,
  HistoryIcon,
  PlayIcon,
  ShieldIcon,
} from "lucide-react";
import { useState } from "react";
import { useWatch, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";

type CandidateSource = "editor" | "version";

type FormulaTemplateBacktestTabProps = {
  form: UseFormReturn<FormulaTemplateFormValues>;
  template: FormulaTemplate | null;
};

const SOURCE_OPTIONS: { value: CandidateSource; label: string; description: string }[] = [
  {
    value: "editor",
    label: "Current Expression",
    description: "Use the expression currently in the editor",
  },
  {
    value: "version",
    label: "Saved Version",
    description: "Use a previously saved version snapshot",
  },
];

function formatDeltaPct(deltaPct: number): string {
  return `${deltaPct >= 0 ? "+" : ""}${deltaPct.toFixed(2)}%`;
}

function DeltaValue({ delta, deltaPct }: { delta: number; deltaPct?: number }) {
  const isZero = delta === 0;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 font-mono tabular-nums",
        isZero
          ? "text-muted-foreground"
          : delta > 0
            ? "text-emerald-600 dark:text-emerald-400"
            : "text-red-600 dark:text-red-400",
      )}
    >
      {!isZero &&
        (delta > 0 ? <ArrowUpIcon className="size-3" /> : <ArrowDownIcon className="size-3" />)}
      {formatCurrency(Math.abs(delta))}
      {deltaPct !== undefined && !isZero && (
        <span className="text-2xs opacity-80">({formatDeltaPct(deltaPct)})</span>
      )}
    </span>
  );
}

function StatTile({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <div className="rounded-lg border bg-muted/30 px-3 py-2">
      <p className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">{label}</p>
      <p className={cn("mt-0.5 text-lg font-semibold tabular-nums", tone)}>{value}</p>
    </div>
  );
}

function BacktestSummaryRow({ summary }: { summary: BacktestSummary }) {
  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
        <StatTile label="Shipments" value={`${summary.evaluatedCount}/${summary.shipmentCount}`} />
        <StatTile label="Changed" value={String(summary.changedCount)} />
        <StatTile
          label="Increased"
          value={String(summary.increasedCount)}
          tone="text-emerald-600 dark:text-emerald-400"
        />
        <StatTile
          label="Decreased"
          value={String(summary.decreasedCount)}
          tone="text-red-600 dark:text-red-400"
        />
        <StatTile
          label="Errors"
          value={String(summary.errorCount)}
          tone={summary.errorCount > 0 ? "text-destructive" : undefined}
        />
      </div>

      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-sm">
        <span className="text-muted-foreground">Total</span>
        <span className="font-mono font-medium tabular-nums">
          {formatCurrency(summary.currentTotal)}
        </span>
        <ArrowRightIcon className="size-3.5 text-muted-foreground" />
        <span className="font-mono font-medium tabular-nums">
          {formatCurrency(summary.candidateTotal)}
        </span>
        <DeltaValue delta={summary.totalDelta} deltaPct={summary.totalDeltaPct} />
        <span className="ml-auto text-xs text-muted-foreground">
          Max increase {formatCurrency(summary.maxIncrease)} · Max decrease{" "}
          {formatCurrency(summary.maxDecrease)}
        </span>
      </div>
    </div>
  );
}

function BacktestResultRow({ result }: { result: BacktestResult }) {
  const hasError = !!result.currentError || !!result.candidateError;

  return (
    <TableRow>
      <TableCell className="font-mono text-xs">{result.proNumber || result.shipmentId}</TableCell>
      <TableCell className="text-right font-mono text-xs tabular-nums">
        {result.currentError ? (
          <span className="text-muted-foreground">—</span>
        ) : (
          formatCurrency(result.currentAmount)
        )}
      </TableCell>
      <TableCell className="text-right font-mono text-xs tabular-nums">
        {result.candidateError ? (
          <span className="text-muted-foreground">—</span>
        ) : (
          formatCurrency(result.candidateAmount)
        )}
      </TableCell>
      <TableCell className="text-right text-xs">
        {hasError ? (
          <span className="text-muted-foreground">—</span>
        ) : (
          <DeltaValue delta={result.delta} deltaPct={result.deltaPct} />
        )}
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-1.5">
          {result.guardrailApplied && (
            <Tooltip>
              <TooltipTrigger
                render={<ShieldIcon className="size-3.5 text-blue-500 dark:text-blue-400" />}
              />
              <TooltipContent side="left" className="text-xs">
                Guardrail clamped the candidate amount
              </TooltipContent>
            </Tooltip>
          )}
          {hasError && (
            <Tooltip>
              <TooltipTrigger
                render={<AlertTriangleIcon className="size-3.5 text-destructive" />}
              />
              <TooltipContent side="left" className="max-w-72 text-xs">
                {result.currentError && <p>Current: {result.currentError}</p>}
                {result.candidateError && <p>Candidate: {result.candidateError}</p>}
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}

export default function FormulaTemplateBacktestTab({
  form,
  template,
}: FormulaTemplateBacktestTabProps) {
  const [source, setSource] = useState<CandidateSource>("editor");
  const [versionNumber, setVersionNumber] = useState<number>(template?.currentVersionNumber ?? 1);
  const [limit, setLimit] = useState<number>(50);

  const expression = useWatch({ control: form.control, name: "expression" });

  const mutation = useMutation({
    mutationFn: () =>
      apiService.formulaTemplateService.backtest(template?.id ?? "", {
        ...(source === "editor" ? { expression } : { versionNumber }),
        limit,
      }),
    onError: () => {
      toast.error("Backtest failed", {
        description: "Please try again or contact your system administrator.",
      });
    },
  });

  const canRun = !!template?.id && (source === "editor" ? !!expression?.trim() : versionNumber > 0);

  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-muted/30 p-3">
        <div className="mb-3">
          <p className="text-sm font-medium">Backtest Candidate</p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Re-rate recent shipments priced by this template and compare against their current
            amounts. Nothing is saved.
          </p>
        </div>

        <div className="mb-3 grid grid-cols-2 gap-2">
          {SOURCE_OPTIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => setSource(option.value)}
              className={cn(
                "rounded-lg border p-2 text-left transition-colors",
                source === option.value
                  ? "border-primary bg-primary/5"
                  : "border-border bg-background hover:bg-muted/50",
              )}
            >
              <p className="text-xs font-medium">{option.label}</p>
              <p className="mt-0.5 text-2xs text-muted-foreground">{option.description}</p>
            </button>
          ))}
        </div>

        <div className="flex flex-wrap items-end gap-3">
          {source === "version" && (
            <div className="w-32">
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                Version
              </label>
              <NumberFieldRoot
                value={versionNumber}
                onValueChange={(val) => setVersionNumber(val ?? 1)}
                min={1}
                step={1}
                size="sm"
              >
                <NumberFieldGroup>
                  <NumberFieldInput className="text-right" />
                </NumberFieldGroup>
              </NumberFieldRoot>
            </div>
          )}
          <div className="w-32">
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
              Shipment Limit
            </label>
            <NumberFieldRoot
              value={limit}
              onValueChange={(val) => setLimit(Math.min(Math.max(val ?? 50, 1), 500))}
              min={1}
              max={500}
              step={10}
              size="sm"
            >
              <NumberFieldGroup>
                <NumberFieldInput className="text-right" />
              </NumberFieldGroup>
            </NumberFieldRoot>
          </div>
          <Button
            type="button"
            size="sm"
            onClick={() => mutation.mutate()}
            disabled={!canRun}
            isLoading={mutation.isPending}
            loadingText="Running..."
            className="gap-1.5"
          >
            <PlayIcon className="size-3.5" />
            Run Backtest
          </Button>
        </div>
      </div>

      {mutation.data ? (
        <>
          <BacktestSummaryRow summary={mutation.data.summary} />

          <div className="overflow-hidden rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="text-xs">Pro #</TableHead>
                  <TableHead className="text-right text-xs">Current</TableHead>
                  <TableHead className="text-right text-xs">Candidate</TableHead>
                  <TableHead className="text-right text-xs">Delta</TableHead>
                  <TableHead className="w-16" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {mutation.data.results.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className="py-8 text-center text-sm text-muted-foreground"
                    >
                      No shipments have been rated with this template yet
                    </TableCell>
                  </TableRow>
                ) : (
                  mutation.data.results.map((result) => (
                    <BacktestResultRow key={result.shipmentId} result={result} />
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </>
      ) : (
        !mutation.isPending && (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12 text-center">
            <HistoryIcon className="mb-3 size-8 text-muted-foreground" />
            <p className="text-sm font-medium">No backtest results yet</p>
            <p className="mt-1 max-w-sm text-xs text-muted-foreground">
              Run a backtest to preview how the candidate expression would change charges on
              shipments already rated by this template.
              {template?.currentVersionNumber ? (
                <Badge variant="outline" className="ml-1 font-mono text-2xs">
                  head v{template.currentVersionNumber}
                </Badge>
              ) : null}
            </p>
          </div>
        )
      )}
    </div>
  );
}
