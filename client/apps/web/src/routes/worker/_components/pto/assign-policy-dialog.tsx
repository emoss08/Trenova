import { useT } from "@trenova/shared/i18n/use-t";
import { PtoPolicyAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { ptoTypeChoices } from "@/lib/choices";
import { useSelectOption } from "@/hooks/use-select-option";
import type { SelectOption } from "@/lib/graphql/select-options";
import { assignWorkerPtoPolicy } from "@/lib/graphql/pto-policy";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  assignPtoPolicyFormSchema,
  type AssignPTOPolicyFormValues,
} from "@trenova/shared/types/pto-policy";
import { useCallback, useEffect } from "react";
import { FormProvider, useFieldArray, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type AssignPolicyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  currentPolicyId?: string | null;
  onAssigned?: () => void;
};

function trackedTypes(policy: SelectOption | null): string[] {
  const types = policy?.meta?.ptoTypes;
  if (!Array.isArray(types)) return [];

  return types.filter((ptoType): ptoType is string => typeof ptoType === "string");
}

export function AssignPolicyDialog({
  open,
  onOpenChange,
  workerId,
  currentPolicyId,
  onAssigned,
}: AssignPolicyDialogProps) {
  const t = useT();

  const form = useForm<AssignPTOPolicyFormValues>({
    resolver: zodResolver(assignPtoPolicyFormSchema) as Resolver<AssignPTOPolicyFormValues>,
    defaultValues: {
      ptoPolicyId: "",
      effectiveFrom: getTodayDate(),
      note: null,
      openingBalances: [],
    },
  });
  const { control, handleSubmit, reset, setValue } = form;
  const openingArray = useFieldArray({ control, name: "openingBalances" });
  const selectedPolicyId = useWatch({ control, name: "ptoPolicyId" });

  useEffect(() => {
    if (open) {
      reset({ ptoPolicyId: "", effectiveFrom: getTodayDate(), note: null, openingBalances: [] });
    }
  }, [open, reset]);

  const { option: selectedPolicy } = useSelectOption("PTO_POLICY", selectedPolicyId);

  useEffect(() => {
    const types = trackedTypes(selectedPolicy);
    setValue(
      "openingBalances",
      types.map((ptoType) => ({
        ptoType: ptoType as AssignPTOPolicyFormValues["openingBalances"][number]["ptoType"],
        days: "0",
      })),
    );
  }, [selectedPolicy, setValue]);

  // The policy already assigned is left out: "changing" a worker onto the
  // policy they are already on is not a change the server will accept.
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
        toast.error(t("Policy not assigned"), { description: failure.error });
        return;
      }
      toast.success(t("Policy assigned"), {
        description: t("Accruals start from the effective date on the next nightly run."),
      });
      onAssigned?.();
      onOpenChange(false);
    },
  });

  const onSubmit = useCallback(
    async (values: AssignPTOPolicyFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {currentPolicyId ? t("Change PTO Policy") : t("Assign PTO Policy")}
          </DialogTitle>
          <DialogDescription>
            {currentPolicyId
              ? t(
                  "The current assignment ends the day before the new one starts. Balances carry across unchanged.",
                )
              : t("Enrol this worker in a policy so their time off accrues and is tracked.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              void handleSubmit(onSubmit)(e);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl cols="full">
                <PtoPolicyAutocompleteField<AssignPTOPolicyFormValues>
                  control={control}
                  name="ptoPolicyId"
                  label={t("Policy")}
                  rules={{ required: true }}
                  placeholder={t("Pick a policy")}
                  filterOption={hideCurrentPolicy}
                  description={t("Decides how time off accrues and which PTO types are tracked.")}
                />
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<AssignPTOPolicyFormValues>
                  control={control}
                  name="effectiveFrom"
                  label={t("Effective from")}
                  placeholder={t("Today")}
                  rules={{ required: true }}
                  description={
                    currentPolicyId
                      ? "The new policy applies from this date; it must be after the current assignment started."
                      : "Accruals count from this date."
                  }
                />
              </FormControl>
              {openingArray.fields.length > 0 ? (
                <FormControl cols="full">
                  <div className="flex flex-col gap-2">
                    <p className="text-sm font-medium">{t("Opening balances")}</p>
                    <p className="text-muted-foreground text-xs">
                      {t("Days the worker already has banked. Posted once as an opening balance.")}
                    </p>
                    <div className="grid grid-cols-2 gap-2">
                      {openingArray.fields.map((field, index) => (
                        <InputField<AssignPTOPolicyFormValues>
                          key={field.id}
                          control={control}
                          name={`openingBalances.${index}.days`}
                          label={
                            ptoTypeChoices.find((choice) => choice.value === field.ptoType)
                              ?.label ?? field.ptoType
                          }
                          placeholder="0"
                          description={t("Posted on the effective date; leave 0 for none.")}
                        />
                      ))}
                    </div>
                  </div>
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <TextareaField<AssignPTOPolicyFormValues>
                  control={control}
                  name="note"
                  label={t("Assignment note")}
                  placeholder={t("e.g. Moved to the regional driver policy")}
                  maxLength={255}
                  description={t("Optional; kept on the assignment record.")}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="button"
                onClick={() => void handleSubmit(onSubmit)()}
                isLoading={isPending}
                loadingText={t("Assigning...")}
              >
                {t("Assign Policy")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
