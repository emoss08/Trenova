import { useT } from "@trenova/shared/i18n/use-t";
import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  EQUIPMENT_IDENTIFIERS,
  EQUIPMENT_UNIT_TYPES,
  PLATE_NUMBER_MAX_LENGTH,
  UNIT_NUMBER_MAX_LENGTH,
  VIN_LENGTH,
  emptyVerifyEquipmentForm,
  normalizeVin,
  toVerifyCarrierEquipmentInput,
  verifyEquipmentFormSchema,
  type EquipmentIdentifier,
  type VerifyEquipmentFormValues,
} from "@/lib/equipment-verification";
import {
  CARRIER_EQUIPMENT_VERIFICATIONS_KEY,
  fetchCarrierEquipmentVerifications,
  verifyCarrierEquipment,
  type CarrierEquipmentVerification,
} from "@/lib/graphql/carrier-intelligence";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { CarrierIntelUnitType } from "@trenova/graphql/generated/graphql";
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
import { Label } from "@trenova/shared/components/ui/label";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { RefreshCwIcon, ScanLineIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { EquipmentOverrideDialog } from "./equipment-override-dialog";
import {
  EquipmentVerificationCard,
  useEquipmentVerificationLabels,
} from "./equipment-verification-card";

export function carrierEquipmentVerificationsQueryKey(carrierAssignmentId: string) {
  return [CARRIER_EQUIPMENT_VERIFICATIONS_KEY, carrierAssignmentId] as const;
}

function VerificationHistory({
  carrierAssignmentId,
  latestId,
  canApprove,
  onOverride,
}: {
  carrierAssignmentId: string;
  latestId: string | null;
  canApprove: boolean;
  onOverride: (verification: CarrierEquipmentVerification) => void;
}) {
  const t = useT();

  const history = useQuery({
    queryKey: carrierEquipmentVerificationsQueryKey(carrierAssignmentId),
    queryFn: ({ signal }) => fetchCarrierEquipmentVerifications(carrierAssignmentId, { signal }),
  });

  if (history.isPending) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  if (history.isError) {
    return (
      <div className="text-destructive flex items-center justify-between gap-2 rounded-lg border border-dashed p-3 text-sm">
        <span>{t("Prior verifications could not be loaded. {0}", history.error.message)}</span>
        <Button type="button" size="xs" variant="outline" onClick={() => void history.refetch()}>
          <RefreshCwIcon />
          {t("Retry")}
        </Button>
      </div>
    );
  }

  if (history.data.length === 0) {
    return (
      <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-center text-sm">
        {t("No equipment has been verified on this assignment yet.")}
      </p>
    );
  }

  return (
    <ul className="flex flex-col gap-2" aria-label={t("Prior verifications")}>
      {history.data.map((verification) => (
        <li key={verification.id}>
          <EquipmentVerificationCard
            verification={verification}
            canApprove={canApprove}
            highlighted={verification.id === latestId}
            onOverride={onOverride}
          />
        </li>
      ))}
    </ul>
  );
}

export type VerifyEquipmentDialogProps = {
  carrierAssignmentId: string;
  carrierName: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function VerifyEquipmentDialog({
  carrierAssignmentId,
  carrierName,
  open,
  onOpenChange,
}: VerifyEquipmentDialogProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const labels = useEquipmentVerificationLabels();
  const { allowed: canRead } = usePermission(Resource.EquipmentVerification, Operation.Read);
  const { allowed: canApprove } = usePermission(Resource.EquipmentVerification, Operation.Approve);
  const [latest, setLatest] = useState<CarrierEquipmentVerification | null>(null);
  const [overriding, setOverriding] = useState<CarrierEquipmentVerification | null>(null);

  const form = useForm<VerifyEquipmentFormValues>({
    resolver: zodResolver(verifyEquipmentFormSchema),
    defaultValues: emptyVerifyEquipmentForm,
  });
  const { control, handleSubmit, setValue, clearErrors } = form;
  const unitType = useWatch({ control, name: "unitType" });
  const identifyBy = useWatch({ control, name: "identifyBy" });
  const vin = useWatch({ control, name: "vin" });

  const unitTypeItems = useMemo(
    () => EQUIPMENT_UNIT_TYPES.map((value) => ({ value, label: labels.unitType[value] })),
    [labels],
  );

  const identifierItems = useMemo(() => {
    const identifierLabels: Record<EquipmentIdentifier, string> = {
      vin: t("VIN"),
      plate: t("Plate"),
      unit: t("Unit number"),
    };
    return EQUIPMENT_IDENTIFIERS.map((value) => ({ value, label: identifierLabels[value] }));
  }, [t]);

  const invalidateHistory = useCallback(() => {
    void queryClient.invalidateQueries({
      queryKey: carrierEquipmentVerificationsQueryKey(carrierAssignmentId),
    });
  }, [carrierAssignmentId, queryClient]);

  const { mutateAsync, isPending } = useApiMutation<
    CarrierEquipmentVerification,
    VerifyEquipmentFormValues,
    unknown,
    VerifyEquipmentFormValues
  >({
    form,
    resourceName: "Equipment verification",
    mutationFn: (values) =>
      verifyCarrierEquipment(toVerifyCarrierEquipmentInput(carrierAssignmentId, values)),
    onSuccess: (verification) => {
      setLatest(verification);
      invalidateHistory();
      const meta = labels.result[verification.result];
      if (verification.result === "Match") {
        toast.success(t("Equipment verified"), { description: meta.description });
      } else {
        toast.warning(meta.label, {
          description: verification.mismatchReason ?? meta.description,
        });
      }
    },
  });

  const handleOverridden = useCallback(
    (updated: CarrierEquipmentVerification) => {
      setLatest((current) => (current?.id === updated.id ? updated : current));
      invalidateHistory();
    },
    [invalidateHistory],
  );

  const selectIdentifier = (value: EquipmentIdentifier) => {
    setValue("identifyBy", value, { shouldDirty: true });
    clearErrors(["vin", "plateNumber", "plateStateId", "unitNumber"]);
  };

  const vinLength = normalizeVin(vin).length;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>{t("Verify equipment")}</DialogTitle>
          <DialogDescription>
            {carrierName
              ? t("Confirm the unit at the dock is registered to {0} before it loads.", carrierName)
              : t("Confirm the unit at the dock is registered to the assigned carrier.")}
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className="max-h-[65vh] pr-2">
          <div className="flex flex-col gap-4 pb-2">
            <FormProvider {...form}>
              <Form
                aria-label={t("Verify equipment")}
                onSubmit={(submitEvent) => {
                  submitEvent.preventDefault();
                  submitEvent.stopPropagation();
                  void handleSubmit((values) => mutateAsync(values))(submitEvent);
                }}
              >
                <div className="flex flex-col gap-3">
                  <div className="flex flex-wrap items-end gap-4">
                    <div className="flex flex-col gap-1">
                      <Label className="text-xs">{t("Unit type")}</Label>
                      <SegmentedControl<CarrierIntelUnitType>
                        items={unitTypeItems}
                        value={unitType}
                        onValueChange={(value) =>
                          setValue("unitType", value, { shouldDirty: true })
                        }
                        aria-label={t("Unit type")}
                      />
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label className="text-xs">{t("Identify by")}</Label>
                      <SegmentedControl<EquipmentIdentifier>
                        items={identifierItems}
                        value={identifyBy}
                        onValueChange={selectIdentifier}
                        aria-label={t("Identify by")}
                      />
                    </div>
                  </div>
                  <FormGroup cols={2}>
                    {identifyBy === "vin" ? (
                      <FormControl cols="full">
                        <InputField<VerifyEquipmentFormValues>
                          control={control}
                          name="vin"
                          label={t("VIN")}
                          placeholder={t("17-character VIN")}
                          rules={{ required: true }}
                          maxLength={VIN_LENGTH + 4}
                          autoComplete="off"
                          description={t(
                            "{0} of 17 characters. VINs never use the letters I, O or Q.",
                            vinLength,
                          )}
                        />
                      </FormControl>
                    ) : null}
                    {identifyBy === "plate" ? (
                      <>
                        <FormControl>
                          <InputField<VerifyEquipmentFormValues>
                            control={control}
                            name="plateNumber"
                            label={t("Plate number")}
                            placeholder={t("e.g., ABC-1234")}
                            rules={{ required: true }}
                            maxLength={PLATE_NUMBER_MAX_LENGTH}
                            autoComplete="off"
                          />
                        </FormControl>
                        <FormControl>
                          <UsStateAutocompleteField<VerifyEquipmentFormValues>
                            control={control}
                            name="plateStateId"
                            label={t("Plate state")}
                            placeholder={t("State")}
                            rules={{ required: true }}
                            onOptionChange={(option) =>
                              setValue(
                                "plateState",
                                option ? selectOptionMetaString(option, "abbreviation") : "",
                                { shouldDirty: true },
                              )
                            }
                          />
                        </FormControl>
                      </>
                    ) : null}
                    {identifyBy === "unit" ? (
                      <FormControl cols="full">
                        <InputField<VerifyEquipmentFormValues>
                          control={control}
                          name="unitNumber"
                          label={t("Unit number")}
                          placeholder={t("e.g., T-4521")}
                          rules={{ required: true }}
                          maxLength={UNIT_NUMBER_MAX_LENGTH}
                          autoComplete="off"
                          description={t(
                            "Unit numbers are the carrier's own labels, so a match is weaker evidence than a VIN or plate.",
                          )}
                        />
                      </FormControl>
                    ) : null}
                  </FormGroup>
                  <div className="flex justify-end">
                    <Button type="submit" size="sm" isLoading={isPending}>
                      <ScanLineIcon />
                      {t("Verify")}
                    </Button>
                  </div>
                </div>
              </Form>
            </FormProvider>

            {latest ? (
              <section aria-label={t("Latest result")} className="flex flex-col gap-2">
                <h4 className="text-sm font-medium">{t("Result")}</h4>
                <EquipmentVerificationCard
                  verification={latest}
                  canApprove={canApprove}
                  highlighted
                  onOverride={setOverriding}
                />
              </section>
            ) : null}

            {canRead ? (
              <section className="flex flex-col gap-2">
                <h4 className="text-sm font-medium">{t("Verification history")}</h4>
                <VerificationHistory
                  carrierAssignmentId={carrierAssignmentId}
                  latestId={latest?.id ?? null}
                  canApprove={canApprove}
                  onOverride={setOverriding}
                />
              </section>
            ) : null}
          </div>
        </ScrollArea>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Close")}
          </Button>
        </DialogFooter>
        <EquipmentOverrideDialog
          verification={overriding}
          open={overriding !== null}
          onOpenChange={(next) => {
            if (!next) {
              setOverriding(null);
            }
          }}
          onOverridden={handleOverridden}
        />
      </DialogContent>
    </Dialog>
  );
}
