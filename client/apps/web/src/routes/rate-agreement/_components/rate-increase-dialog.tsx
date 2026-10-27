import { useT } from "@trenova/shared/i18n/use-t";
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
import { cn } from "@trenova/shared/lib/utils";
import type { RateAgreementRow } from "@/lib/graphql/rate-tables";
import type { RateIncreasePlan, RatePartyType } from "@trenova/shared/types/rate";
import type { TariffWireValue } from "@trenova/shared/types/tariff";
import { humanizeTariffName } from "@trenova/shared/lib/tariff-inputs";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { isTariffDate, isTariffNumber, isTariffQuantity } from "@trenova/shared/rules/tariff-wire";
import { AlertCircleIcon, TrendUp01Icon } from "@trenova/shared/components/icons";
import { useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";

type IncreaseScope = "selected" | "customer" | "carrier" | "party";
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
  readonly selectedAgreements?: RateAgreementRow[];
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
  const t = useT();

  const queryClient = useQueryClient();
  const hasSelection = selectedAgreements.length > 0;

  const [scope, setScope] = useState<IncreaseScope>(hasSelection ? "selected" : "customer");
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
      setProblem(t("Say how far the rates move, as a percent that is not zero."));
      return undefined;
    }
    if (amount <= -100) {
      setProblem(
        t("A decrease of 100% or more would take every rate it touches to nothing or below."),
      );
      return undefined;
    }

    const payload: RateIncreaseRequestPayload = {
      adjustment: { percentChange: amount },
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
          setProblem(t("Choose the customer whose agreements this touches."));
          return undefined;
        }
        payload.customerId = values.customerId;
        payload.partyType = "Customer";
        break;
      case "carrier":
        if (!values.carrierId) {
          setProblem(t("Choose the carrier whose agreements this touches."));
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
    onError: () => toast.error(t("The increase could not be previewed")),
  });

  const apply = useMutation({
    mutationFn: (payload: RateIncreaseRequestPayload) =>
      apiService.rateAgreementService.applyRateIncrease(payload),
    onSuccess: (applied) => {
      toast.success(
        t(
          "Rates raised on {0, plural, one {# lane} other {# lanes}} across {1, plural, one {# agreement} other {# agreements}}",
          applied.lines?.length ?? 0,
          applied.agreementCount,
        ),
      );
      void queryClient.invalidateQueries({ queryKey: ["rate-agreement-list"] });
      void queryClient.invalidateQueries({ queryKey: ["rate-agreement"] });
      onOpenChange(false);
    },
    onError: () => toast.error(t("The increase could not be applied")),
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
  const canApply = Boolean(plan) && lines.length > 0;

  const scopeOptions: { value: IncreaseScope; label: string; description: string }[] = [
    ...(hasSelection
      ? [
          {
            value: "selected" as const,
            label: t("Selected Agreements ({0})", selectedAgreements.length),
            description: t("Only the agreements picked in the table"),
          },
        ]
      : []),
    {
      value: "customer",
      label: t("One customer"),
      description: t("Every active agreement billing one customer"),
    },
    {
      value: "carrier",
      label: t("One carrier"),
      description: t("Every active agreement paying one carrier"),
    },
    {
      value: "party",
      label: t("Across the board"),
      description: t("Every active agreement of a party type"),
    },
  ];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg" className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <TrendUp01Icon className="size-4" />
            {t("General Rate Increase")}
          </DialogTitle>
          <DialogDescription>
            {t(
              "Every affected lane is closed out and succeeded at the new rate from the effective date. The old rates stay in history, and nothing moves until you have read the preview.",
            )}
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
              <p className="text-xs font-medium">{t(option.label)}</p>
              <p className="text-2xs text-muted-foreground mt-0.5">{t(option.description)}</p>
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
                    label={t("Customer")}
                    placeholder={t("Select customer")}
                    description={t(
                      "Every active agreement billing this customer takes the increase.",
                    )}
                  />
                </FormControl>
              )}
              {scope === "carrier" && (
                <FormControl>
                  <CarrierAutocompleteField
                    control={form.control}
                    name="carrierId"
                    label={t("Carrier")}
                    placeholder={t("Select carrier")}
                    description={t(
                      "Every active agreement paying this carrier takes the increase.",
                    )}
                  />
                </FormControl>
              )}
              {scope === "party" && (
                <FormControl>
                  <SelectField
                    control={form.control}
                    name="partyType"
                    label={t("Party type")}
                    options={ratePartyTypeChoices}
                    description={t(
                      "Customer agreements raise what you bill; carrier agreements raise what you pay.",
                    )}
                  />
                </FormControl>
              )}
              <FormControl>
                <AutoCompleteDateField
                  control={form.control}
                  name="effectiveFrom"
                  label={t("Takes effect")}
                  rules={{ required: true }}
                  description={t(
                    "The announced date. Shipments before it keep pricing at the old rates.",
                  )}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>

        <div className="bg-muted/30 rounded-lg border p-3">
          <div className="flex flex-wrap items-end gap-3">
            <div className="w-36">
              <label className="text-muted-foreground mb-1.5 block text-xs font-medium">
                {t("Change (%)")}
              </label>
              <NumberFieldRoot
                value={amount}
                onValueChange={(value) => {
                  setAmount(value ?? undefined);
                  invalidatePlan();
                }}
                step={0.5}
                size="sm"
              >
                <NumberFieldGroup>
                  <NumberFieldInput aria-label={t("Change")} className="text-right" />
                </NumberFieldGroup>
              </NumberFieldRoot>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={runPreview}
              isLoading={preview.isPending}
              loadingText={t("Reading...")}
            >
              {t("Preview changes")}
            </Button>
            <p className="text-2xs text-muted-foreground">
              {t(
                "Every money term a lane sets moves by this share of itself, weight break rates and minimums included; counts, percents and weights stay as they are. A negative change is a decrease.",
              )}
            </p>
          </div>
        </div>

        {problem && (
          <Alert variant="destructive">
            <AlertCircleIcon className="size-4" />
            <AlertDescription>{problem}</AlertDescription>
          </Alert>
        )}

        {plan && (
          <div className="flex flex-col gap-3">
            <p className="text-sm">
              {lines.length > 0
                ? t("{0} lanes across {1} agreements move.", lines.length, plan.agreementCount)
                : t("No lane in scope sets a money term this change could move.")}
              {plan.skippedNoMoney > 0 &&
                ` ${t("{0, plural, one {# lane sets no money term of its own and is untouched.} other {# lanes set no money term of their own and are untouched.}}", plan.skippedNoMoney)}`}
            </p>

            {lines.length > 0 && (
              <div className="overflow-hidden rounded-lg border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="text-xs">{t("Agreement")}</TableHead>
                      <TableHead className="text-xs">{t("Lane")}</TableHead>
                      <TableHead className="text-xs">{t("What moves")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {lines.map((line) => (
                      <TableRow key={line.ruleId}>
                        <TableCell className="align-top text-xs">{line.agreementCode}</TableCell>
                        <TableCell className="align-top">
                          <span className="text-xs font-medium">{line.label || line.laneKey}</span>
                          {line.agreementName && (
                            <p className="text-2xs text-muted-foreground">{line.agreementName}</p>
                          )}
                        </TableCell>
                        <TableCell className="align-top">
                          <ul className="flex flex-col gap-1">
                            {line.changed.map((name) => (
                              <InputChangeRow
                                key={name}
                                name={name}
                                before={line.before[name]}
                                after={line.after[name]}
                              />
                            ))}
                          </ul>
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
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            disabled={!canApply}
            isLoading={apply.isPending}
            loadingText={t("Applying...")}
            onClick={runApply}
          >
            {t("Apply Increase")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** The first money amount inside a wire value, as a number, for the size of a change. */
function firstAmount(value: TariffWireValue): number | null {
  if (isTariffQuantity(value)) return Number(value.$q);
  if (Array.isArray(value)) {
    for (const item of value) {
      const found = firstAmount(item);
      if (found !== null) return found;
    }
    return null;
  }
  if (value && typeof value === "object" && !isTariffNumber(value) && !isTariffDate(value)) {
    for (const field of Object.values(value)) {
      const found = firstAmount(field);
      if (found !== null) return found;
    }
  }
  return null;
}

/**
 * One input a GRI moves: its name, the value before and after, and by how much. The increase
 * scales only values the lane sets itself, so a moved input is set on both sides.
 */
function InputChangeRow({
  name,
  before: beforeValue,
  after: afterValue,
}: {
  name: string;
  before: TariffWireValue | undefined;
  after: TariffWireValue | undefined;
}) {
  const t = useT();
  if (beforeValue === undefined || afterValue === undefined) return null;
  const before = firstAmount(beforeValue);
  const after = firstAmount(afterValue);
  const isList = Array.isArray(afterValue);

  return (
    <li className="flex flex-wrap items-baseline gap-x-2 text-xs">
      <span className="text-muted-foreground">{humanizeTariffName(name)}</span>
      {isList ? (
        <span>
          {t(
            "{0, plural, one {# row moves} other {# rows move}}",
            (afterValue as TariffWireValue[]).length,
          )}
        </span>
      ) : (
        <span className="font-mono tabular-nums">
          {t("{0} → {1}", formatTariffValue(beforeValue, t), formatTariffValue(afterValue, t))}
        </span>
      )}
      {before !== null && after !== null && <DeltaValue delta={after - before} />}
    </li>
  );
}
