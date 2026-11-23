import { PtoPolicyAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSelectOption } from "@/hooks/use-select-option";
import { assignWorkerPtoPolicy } from "@/lib/graphql/pto-policy";
import type { SelectOption } from "@/lib/graphql/select-options";
import { ptoTypeLabel } from "@/routes/worker/_components/pto/worker-pto-balances";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  assignPtoPolicyFormSchema,
  type AssignPTOPolicyFormValues,
} from "@trenova/shared/types/pto-policy";
import { useCallback, useEffect } from "react";
import { FormProvider, useFieldArray, useForm, useWatch, type Resolver } from "react-hook-form";

export type ChangePolicyPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  currentPolicyId: string | null;
  onAssigned: () => void;
};

function trackedTypes(policy: SelectOption | null): string[] {
  const types = policy?.meta?.ptoTypes;
  if (!Array.isArray(types)) return [];
  return types.filter((ptoType): ptoType is string => typeof ptoType === "string");
}

function emptyValues(): AssignPTOPolicyFormValues {
  return { ptoPolicyId: "", effectiveFrom: getTodayDate(), note: null, openingBalances: [] };
}

/**
 * Enrolling the worker in a time off policy, or moving them to another from a date, beside the
 * balances it changes. A first enrolment can post the days the worker already has banked.
 */
export function ChangePolicyPane({
  open,
  onOpenChange,
  workerId,
  currentPolicyId,
  onAssigned,
}: ChangePolicyPaneProps) {
  const t = useT();
  const form = useForm<AssignPTOPolicyFormValues>({
    resolver: zodResolver(assignPtoPolicyFormSchema) as Resolver<AssignPTOPolicyFormValues>,
    defaultValues: emptyValues(),
  });
  const { control, handleSubmit, reset, setValue, setError } = form;
  const openingArray = useFieldArray({ control, name: "openingBalances" });
  const selectedPolicyId = useWatch({ control, name: "ptoPolicyId" });

  useEffect(() => {
    if (open) reset(emptyValues());
  }, [open, reset]);

  const { option: selectedPolicy } = useSelectOption("PTO_POLICY", selectedPolicyId);

  useEffect(() => {
    setValue(
      "openingBalances",
      trackedTypes(selectedPolicy).map((ptoType) => ({
        ptoType: ptoType as AssignPTOPolicyFormValues["openingBalances"][number]["ptoType"],
        days: "0",
      })),
    );
  }, [selectedPolicy, setValue]);

  const hideCurrentPolicy = useCallback(
    (option: SelectOption) => option.id !== currentPolicyId,
    [currentPolicyId],
  );

  const { mutateAsync, isPending } = useApiMutation<
    Awaited<ReturnType<typeof assignWorkerPtoPolicy>>,
    AssignPTOPolicyFormValues,
    unknown,
    AssignPTOPolicyFormValues
  >({
    form,
    resourceName: "PTO policy assignment",
    scope: { id: `worker-pto-policy:${workerId}` },
    mutationFn: (values) =>
      assignWorkerPtoPolicy({
        ptoPolicyId: values.ptoPolicyId,
        workerIds: [workerId],
        effectiveFrom: values.effectiveFrom,
        note: values.note ?? undefined,
        openingBalances: values.openingBalances
          .filter((balance) => Number(balance.days) > 0)
          .map((balance) => ({ ptoType: balance.ptoType, days: balance.days })),
      }),
    onSuccess: (result) => {
      const failure = result.failures[0];
      if (failure) {
        setError("root", { message: failure.error });
        return;
      }
      onAssigned();
      onOpenChange(false);
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={currentPolicyId ? t("Change time off policy") : t("Enroll in a time off policy")}
      description={
        currentPolicyId
          ? t(
              "Moves them to another policy from a date. The current assignment ends the day before, and balances carry across unchanged.",
            )
          : t("Enrols them in a policy so their time off accrues and is tracked.")
      }
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void handleSubmit((values) => mutateAsync(values))(event);
          }}
        >
          <WorkPaneBody>
            <FormGroup cols={1}>
              <FormControl>
                <PtoPolicyAutocompleteField<AssignPTOPolicyFormValues>
                  control={control}
                  name="ptoPolicyId"
                  label={t("Policy")}
                  required
                  placeholder={t("Pick a policy")}
                  filterOption={hideCurrentPolicy}
                  description={t("Decides how time off accrues and which types are tracked.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<AssignPTOPolicyFormValues>
                  control={control}
                  name="effectiveFrom"
                  label={t("Effective from")}
                  placeholder={t("Today")}
                  required
                  description={
                    currentPolicyId
                      ? t(
                          "The new policy applies from this date; it must be after the current one started.",
                        )
                      : t("Accruals count from this date.")
                  }
                />
              </FormControl>
              {openingArray.fields.length > 0 ? (
                <FormControl>
                  <div className="flex flex-col gap-2">
                    <p className="text-xs font-medium">{t("Opening balances")}</p>
                    <p className="text-2xs text-muted-foreground">
                      {t("Days they already have banked, posted once on the effective date.")}
                    </p>
                    <div className="grid grid-cols-2 gap-2">
                      {openingArray.fields.map((field, index) => (
                        <InputField<AssignPTOPolicyFormValues>
                          key={field.id}
                          control={control}
                          name={`openingBalances.${index}.days`}
                          label={t(ptoTypeLabel(field.ptoType))}
                          placeholder="0"
                        />
                      ))}
                    </div>
                  </div>
                </FormControl>
              ) : null}
              <FormControl>
                <TextareaField<AssignPTOPolicyFormValues>
                  control={control}
                  name="note"
                  label={t("Note")}
                  placeholder={t("e.g. Moved to the regional driver policy")}
                  maxLength={255}
                  description={t("Kept on the assignment record.")}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {currentPolicyId ? t("Change policy") : t("Enroll")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
