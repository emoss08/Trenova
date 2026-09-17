import { useT } from "@trenova/shared/i18n/use-t";
import { FreshnessIndicator } from "@/components/carrier-intelligence/freshness-indicator";
import { ProviderErrorAlert } from "@/components/carrier-intelligence/provider-error-alert";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { InputField } from "@/components/fields/input-field";
import {
  availableVetDepths,
  carrierIntelProviderLabel,
  carrierIntelVetCost,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelProviderCapabilities } from "@/lib/carrier-sourcing";
import {
  CARRIER_INTEL_LOOKUP_KEY,
  lookupCarrierIntelProspect,
} from "@/lib/graphql/carrier-sourcing";
import { zodResolver } from "@hookform/resolvers/zod";
import type { CarrierIntelLookupInput } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { CoinsIcon, ScanSearchIcon } from "lucide-react";
import { useEffect, useMemo } from "react";
import { Controller, FormProvider, useForm, type Resolver } from "react-hook-form";
import { SourcingResultCard, type SourcingCandidate } from "./sourcing-result-card";
import { SourcingResultsSketch } from "./sourcing-results";
import {
  prospectLookupSchema,
  toLookupInput,
  type LookupKind,
  type ProspectLookupFormValues,
} from "./sourcing-schema";

export type ProspectLookupRequest = {
  input: CarrierIntelLookupInput;
  requestedAt: number;
};

export type ProspectLookupProps = {
  provider: CarrierIntelProviderCapabilities | null;
  request: ProspectLookupRequest | null;
  prefill: { kind: LookupKind; number: string } | null;
  canImport: boolean;
  ruleLabels: Readonly<Record<string, string>>;
  onLookup: (request: ProspectLookupRequest) => void;
  onImport: (candidate: SourcingCandidate) => void;
};

export function ProspectLookup({
  provider,
  request,
  prefill,
  canImport,
  ruleLabels,
  onLookup,
  onImport,
}: ProspectLookupProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const depths = useMemo(
    () => (provider ? availableVetDepths(provider.capabilities) : []),
    [provider],
  );
  const defaultDepth = depths.includes("Lite") ? "Lite" : (depths[0] ?? null);

  const form = useForm<ProspectLookupFormValues>({
    resolver: zodResolver(prospectLookupSchema) as Resolver<ProspectLookupFormValues>,
    defaultValues: {
      kind: prefill?.kind ?? "dot",
      number: prefill?.number ?? "",
      depth: defaultDepth,
    },
  });
  const { control, handleSubmit, setValue, getValues } = form;

  useEffect(() => {
    if (getValues("depth") === null && defaultDepth !== null) {
      setValue("depth", defaultDepth);
    }
  }, [defaultDepth, getValues, setValue]);

  useEffect(() => {
    if (prefill) {
      setValue("kind", prefill.kind);
      setValue("number", prefill.number);
    }
  }, [prefill, setValue]);

  const lookupQuery = useQuery({
    queryKey: [CARRIER_INTEL_LOOKUP_KEY, request?.input ?? null, request?.requestedAt ?? 0],
    queryFn: ({ signal }) => {
      if (!request) {
        throw new Error("No lookup requested");
      }
      return lookupCarrierIntelProspect(request.input, { signal });
    },
    enabled: request !== null,
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
    refetchOnWindowFocus: false,
  });

  return (
    <div className="flex flex-col gap-4">
      <FormProvider {...form}>
        <Form
          className="bg-card flex flex-col gap-3 rounded-lg border p-4"
          aria-label={t("Carrier lookup")}
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void handleSubmit((values) =>
              onLookup({ input: toLookupInput(values), requestedAt: Date.now() }),
            )(event);
          }}
        >
          <FormGroup cols={3} className="items-end gap-y-3">
            <FormControl>
              <div className="flex flex-col gap-1.5">
                <Label>{t("Look up by")}</Label>
                <Controller
                  control={control}
                  name="kind"
                  render={({ field }) => (
                    <SegmentedControl<LookupKind>
                      items={[
                        { value: "dot", label: t("USDOT number") },
                        { value: "mc", label: t("MC number") },
                      ]}
                      value={field.value}
                      onValueChange={field.onChange}
                      fullWidth
                      aria-label={t("Identifier type")}
                    />
                  )}
                />
              </div>
            </FormControl>
            <FormControl>
              <InputField<ProspectLookupFormValues>
                control={control}
                name="number"
                label={t("Number")}
                placeholder={t("e.g. 1234567")}
                inputMode="numeric"
                autoComplete="off"
                maxLength={12}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <Button type="submit" size="sm" isLoading={lookupQuery.isFetching}>
                <ScanSearchIcon className="size-3.5" />
                {t("Preview carrier")}
              </Button>
            </FormControl>
            {depths.length > 0 ? (
              <FormControl cols="full">
                <Controller
                  control={control}
                  name="depth"
                  render={({ field }) => {
                    const value = field.value ?? depths[0];
                    const cost = carrierIntelVetCost(provider?.provider, value);
                    return (
                      <div className="flex flex-col gap-1.5">
                        <Label>{t("Depth")}</Label>
                        <SegmentedControl
                          items={depths.map((depth) => ({
                            value: depth,
                            label: labels.depth[depth],
                          }))}
                          value={value}
                          onValueChange={field.onChange}
                          aria-label={t("Lookup depth")}
                          className="self-start"
                        />
                        <p className="text-muted-foreground flex items-start gap-1.5 text-xs">
                          <CoinsIcon className="mt-0.5 size-3.5 shrink-0" aria-hidden />
                          <span>
                            {labels.depthHint[value]}{" "}
                            {cost.basis === "Free"
                              ? t(
                                  "{0} does not charge per lookup.",
                                  carrierIntelProviderLabel(provider?.provider),
                                )
                              : cost.basis === "PerDOTMonth"
                                ? t(
                                    "About {0} per carrier per month at this depth.",
                                    formatCurrency(cost.amount),
                                  )
                                : t(
                                    "About {0} per matched lookup at this depth.",
                                    formatCurrency(cost.amount),
                                  )}
                          </span>
                        </p>
                      </div>
                    );
                  }}
                />
              </FormControl>
            ) : null}
          </FormGroup>
        </Form>
      </FormProvider>
      {request === null ? (
        <EmptySheet
          title={t("Preview a carrier before importing it")}
          description={t(
            "Enter a USDOT or MC number to pull the carrier's authority, insurance, safety and fleet from the provider and see how it scores against your vetting rules.",
          )}
          sketch={<SourcingResultsSketch />}
        />
      ) : lookupQuery.isPending ? (
        <Skeleton className="h-48 w-full" />
      ) : lookupQuery.isError ? (
        <ProviderErrorAlert
          error={lookupQuery.error}
          title={t("The lookup did not run")}
          onRetry={() => void lookupQuery.refetch()}
        />
      ) : (
        <div className="flex flex-col gap-2">
          <FreshnessIndicator
            fetchedAt={lookupQuery.data.snapshot.fetchedAt}
            confirmedAt={lookupQuery.data.snapshot.confirmedAt}
            effectiveAsOf={lookupQuery.data.snapshot.effectiveAsOf}
            sourceAsOf={lookupQuery.data.snapshot.sourceAsOf}
          />
          <ul>
            <SourcingResultCard
              candidate={{
                dotNumber: lookupQuery.data.snapshot.dotNumber,
                legalName: lookupQuery.data.snapshot.profile.identity?.legalName ?? null,
                existingCarrierId: lookupQuery.data.existingCarrierId,
                riskLevel: lookupQuery.data.snapshot.riskLevel,
                findings: lookupQuery.data.snapshot.findings,
                profile: lookupQuery.data.snapshot.profile,
                notFound: lookupQuery.data.snapshot.notFound,
              }}
              provider={lookupQuery.data.snapshot.provider}
              canImport={canImport && !lookupQuery.data.snapshot.notFound}
              ruleLabels={ruleLabels}
              onImport={onImport}
              defaultExpanded
            />
          </ul>
          {lookupQuery.data.snapshot.notFound ? (
            <p className="text-muted-foreground text-xs">
              {t(
                "{0} has no record for this number, so there is nothing to import.",
                carrierIntelProviderLabel(lookupQuery.data.snapshot.provider),
              )}
            </p>
          ) : null}
        </div>
      )}
    </div>
  );
}
