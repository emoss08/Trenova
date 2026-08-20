import {
  CarrierAutocompleteField,
  CustomerAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { SelectField } from "@/components/fields/select-field";
import { DeltaValue } from "@/components/metric-tiles";
import { ratePartyTypeChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { RateIncreaseRequestPayload } from "@/services/rate";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  NumberFieldGroup,
  NumberFieldInput,
  NumberField as NumberFieldRoot,
} from "@trenova/shared/components/ui/number-field";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { getTodayDate } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { RateAgreement, RateIncreasePlan, RatePartyType } from "@trenova/shared/types/rate";
import { CircleAlertIcon, TrendingUpIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";

type IncreaseScope = "selected" | "customer" | "carrier" | "party";
type AdjustmentKind = "percent" | "flat";

const ADJUSTMENT_OPTIONS: { value: AdjustmentKind; label: string; description: string }[] = [
  {
    value: "percent",
    label: "Percent",
    description: "Every rate moves by this share of itself — the classic GRI",
  },
  {
    value: "flat",
    label: "Flat Amount",
    description: "Every rate moves by this amount per rating unit",
  },
];

type ScopeForm = {
  customerId: string;
  carrierId: string;
  partyType: RatePartyType;
  effectiveFrom: number;
};

type RateIncreaseDialogProps = {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  /** Agreements picked in the table, when the dialog was opened from a selection. */
  readonly selectedAgreements?: RateAgreement[];
};

/**
 * A general rate increase, previewed before it lands.
 *
 * Every affected lane is closed out and succeeded at the new rate from the
 * effective date — nothing is edited in place, so "what did this lane cost
 * before the GRI" stays answerable forever. The preview is the point: nobody
 * should move a thousand rates without reading what will move.
 */
export function RateIncreaseDialog({
  open,
  onOpenChange,
  selectedAgreements = [],
}: RateIncreaseDialogProps) {
  const queryClient = useQueryClient();
  const hasSelection = selectedAgreements.length > 0;

  const [scope, setScope] = useState<IncreaseScope>(hasSelection ? "selected" : "customer");
  const [kind, setKind] = useState<AdjustmentKind>("percent");
  const [amount, setAmount] = useState<number | undefined>();
  const [plan, setPlan] = useState<RateIncreasePlan | undefined>();
  const [problem, setProblem] = useState("");

  const form = useForm<ScopeForm>({
    defaultValues: {
      customerId: "",
      carrierId: "",
      partyType: "Customer",
      effectiveFrom: getTodayDate(),
    },
  });

  useEffect(() => {
    if (open) {
      setScope(hasSelection ? "selected" : "customer");
      setKind("percent");
      setAmount(undefined);
      setPlan(undefined);
      setProblem("");
      form.reset({
        customerId: "",
        carrierId: "",
        partyType: "Customer",
        effectiveFrom: getTodayDate(),
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, hasSelection, form.reset]);

  const buildPayload = (): RateIncreaseRequestPayload | undefined => {
    const values = form.getValues();

    if (amount === undefined || Number.isNaN(amount) || amount === 0) {
      setProblem("Say how far the rates move — a percent or a flat amount, and not zero.");
      return undefined;
    }

    const payload: RateIncreaseRequestPayload = {
      adjustment: kind === "percent" ? { percentChange: amount } : { flatChange: amount },
      effectiveFrom: values.effectiveFrom,
    };

    switch (scope) {
      case "selected":
        payload.agreementIds = selectedAgreements
          .map((agreement) => agreement.id ?? "")
          .filter(Boolean);
        break;
      case "customer":
        if (!values.customerId) {
          setProblem("Choose the customer whose agreements this touches.");
          return undefined;
        }
        payload.customerId = values.customerId;
        payload.partyType = "Customer";
        break;
      case "carrier":
        if (!values.carrierId) {
          setProblem("Choose the carrier whose agreements this touches.");
          return undefined;
        }
        payload.carrierId = values.carrierId;
        payload.partyType = "Carrier";
        break;
      case "party":
        payload.partyType = values.partyType;
        break;
    }

    setProblem("");

    return payload;
  };

  const preview = useMutation({
    mutationFn: (payload: RateIncreaseRequestPayload) =>
      apiService.rateAgreementService.previewRateIncrease(payload),
    onSuccess: setPlan,
    onError: () => toast.error("The increase could not be previewed"),
  });

  const apply = useMutation({
    mutationFn: (payload: RateIncreaseRequestPayload) =>
      apiService.rateAgreementService.applyRateIncrease(payload),
    onSuccess: (applied) => {
      toast.success(
        `Rates raised on ${applied.lines?.length ?? 0} lanes across ${applied.agreementCount} agreements`,
      );
      void queryClient.invalidateQueries({ queryKey: ["rate-agreement-list"] });
      void queryClient.invalidateQueries({ queryKey: ["rate-agreement"] });
      onOpenChange(false);
    },
    onError: () => toast.error("The increase could not be applied"),
  });

  const runPreview = () => {
    const payload = buildPayload();
    if (payload) preview.mutate(payload);
  };

  const runApply = () => {
    const payload = buildPayload();
    if (payload) apply.mutate(payload);
  };

  // Any change to what the increase would touch makes the last preview stale.
  const invalidatePlan = () => setPlan(undefined);

  const lines = plan?.lines ?? [];
  const canApply = Boolean(plan) && lines.length > 0 && (plan?.negativeCount ?? 0) === 0;

  const scopeOptions: { value: IncreaseScope; label: string; description: string }[] = [
    ...(hasSelection
      ? [
          {
            value: "selected" as const,
            label: `Selected Agreements (${selectedAgreements.length})`,
            description: "Only the agreements picked in the table",
          },
        ]
      : []),
    {
      value: "customer",
      label: "One Customer",
      description: "Every active agreement billing one customer",
    },
    {
      value: "carrier",
      label: "One Carrier",
      description: "Every active agreement paying one carrier",
    },
    {
      value: "party",
      label: "Across the Board",
      description: "Every active agreement of a party type",
    },
  ];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <TrendingUpIcon className="size-4" />
            General Rate Increase
          </DialogTitle>
          <DialogDescription>
            Every affected lane is closed out and succeeded at the new rate from the effective date.
            The old rates stay in history, and nothing moves until you have read the preview.
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-2 gap-2">
          {scopeOptions.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => {
                setScope(option.value);
                invalidatePlan();
              }}
              className={cn(
                "rounded-lg border p-2 text-left transition-colors",
                scope === option.value
                  ? "border-primary bg-primary/5"
                  : "border-border bg-background hover:bg-muted/50",
              )}
            >
              <p className="text-xs font-medium">{option.label}</p>
              <p className="text-2xs text-muted-foreground mt-0.5">{option.description}</p>
            </button>
          ))}
        </div>

        <FormProvider {...form}>
          <Form onSubmit={(event) => event.preventDefault()}>
            <FormGroup cols={2}>
              {scope === "customer" && (
                <FormControl>
                  <CustomerAutocompleteField
                    control={form.control}
                    name="customerId"
                    label="Customer"
                    placeholder="Select customer"
                    description="Every active agreement billing this customer takes the increase."
                  />
                </FormControl>
              )}
              {scope === "carrier" && (
                <FormControl>
                  <CarrierAutocompleteField
                    control={form.control}
                    name="carrierId"
                    label="Carrier"
                    placeholder="Select carrier"
                    description="Every active agreement paying this carrier takes the increase."
                  />
                </FormControl>
              )}
              {scope === "party" && (
                <FormControl>
                  <SelectField
                    control={form.control}
                    name="partyType"
                    label="Party Type"
                    options={ratePartyTypeChoices}
                    description="Customer agreements raise what you bill; carrier agreements raise what you pay."
                  />
                </FormControl>
              )}
              <FormControl>
                <AutoCompleteDateField
                  control={form.control}
                  name="effectiveFrom"
                  label="Takes Effect"
                  rules={{ required: true }}
                  description="The announced date. Shipments before it keep pricing at the old rates."
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>

        <div className="bg-muted/30 rounded-lg border p-3">
          <div className="mb-3 grid grid-cols-2 gap-2">
            {ADJUSTMENT_OPTIONS.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={() => {
                  setKind(option.value);
                  invalidatePlan();
                }}
                className={cn(
                  "rounded-lg border p-2 text-left transition-colors",
                  kind === option.value
                    ? "border-primary bg-primary/5"
                    : "border-border bg-background hover:bg-muted/50",
                )}
              >
                <p className="text-xs font-medium">{option.label}</p>
                <p className="text-2xs text-muted-foreground mt-0.5">{option.description}</p>
              </button>
            ))}
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <div className="w-36">
              <label className="text-muted-foreground mb-1.5 block text-xs font-medium">
                {kind === "percent" ? "Change (%)" : "Change ($)"}
              </label>
              <NumberFieldRoot
                value={amount}
                onValueChange={(value) => {
                  setAmount(value ?? undefined);
                  invalidatePlan();
                }}
                step={kind === "percent" ? 0.5 : 0.05}
                size="sm"
              >
                <NumberFieldGroup>
                  <NumberFieldInput aria-label="Change" className="text-right" />
                </NumberFieldGroup>
              </NumberFieldRoot>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={runPreview}
              isLoading={preview.isPending}
              loadingText="Reading..."
            >
              Preview Changes
            </Button>
            <p className="text-2xs text-muted-foreground">
              A negative change is a decrease. Weight breaks move with their lane.
            </p>
          </div>
        </div>

        {problem && (
          <Alert variant="destructive">
            <CircleAlertIcon className="size-4" />
            <AlertDescription>{problem}</AlertDescription>
          </Alert>
        )}

        {plan && (
          <div className="flex flex-col gap-3">
            <p className="text-sm">
              {lines.length > 0
                ? `${lines.length} lanes across ${plan.agreementCount} agreements move.`
                : "No lane in scope carries a rate this change could move."}
              {plan.skippedNoRate > 0 &&
                ` ${plan.skippedNoRate} matrix-priced lanes are untouched — their rates live in the matrix cells.`}
            </p>

            {plan.negativeCount > 0 && (
              <Alert variant="destructive">
                <CircleAlertIcon className="size-4" />
                <AlertDescription>
                  This decrease would push {plan.negativeCount} lanes below zero, and a negative
                  rate is not a discount. Narrow the scope or soften the change.
                </AlertDescription>
              </Alert>
            )}

            {lines.length > 0 && (
              <div className="overflow-hidden rounded-lg border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="text-xs">Agreement</TableHead>
                      <TableHead className="text-xs">Lane</TableHead>
                      <TableHead className="text-right text-xs">Before</TableHead>
                      <TableHead className="text-right text-xs">After</TableHead>
                      <TableHead className="text-right text-xs">Change</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {lines.map((line) => (
                      <TableRow key={line.ruleId}>
                        <TableCell className="text-xs">{line.agreementCode}</TableCell>
                        <TableCell>
                          <span className="text-xs font-medium">{line.label || line.laneKey}</span>
                          {line.breakCount > 0 && (
                            <p className="text-2xs text-muted-foreground">
                              {line.breakCount} weight breaks move with it
                            </p>
                          )}
                        </TableCell>
                        <TableCell className="text-right font-mono text-xs tabular-nums">
                          {formatCurrency(line.before)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-xs tabular-nums">
                          {formatCurrency(line.after)}
                        </TableCell>
                        <TableCell className="text-right text-xs">
                          <DeltaValue delta={line.after - line.before} />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!canApply}
            isLoading={apply.isPending}
            loadingText="Applying..."
            onClick={runApply}
          >
            Apply Increase
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
