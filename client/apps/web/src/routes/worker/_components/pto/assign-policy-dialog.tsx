import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { ptoTypeChoices } from "@/lib/choices";
import {
  assignWorkerPtoPolicy,
  fetchPtoPolicyOptions,
  PTO_POLICY_OPTIONS_KEY,
  type PTOPolicyOption,
} from "@/lib/graphql/pto-policy";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
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
import { useCallback, useEffect, useMemo } from "react";
import { FormProvider, useFieldArray, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type AssignPolicyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  currentPolicyId?: string | null;
  onAssigned?: () => void;
};

function trackedTypes(policy: PTOPolicyOption | undefined): string[] {
  return policy ? policy.rules.map((rule) => rule.ptoType) : [];
}

export function AssignPolicyDialog({
  open,
  onOpenChange,
  workerId,
  currentPolicyId,
  onAssigned,
}: AssignPolicyDialogProps) {
  const { data: policies = [], isLoading } = useQuery({
    queryKey: [PTO_POLICY_OPTIONS_KEY],
    queryFn: ({ signal }) => fetchPtoPolicyOptions({ signal }),
    enabled: open,
  });

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

  const selectedPolicy = useMemo(
    () => policies.find((policy) => policy.id === selectedPolicyId),
    [policies, selectedPolicyId],
  );

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

  const policyOptions = useMemo(
    () =>
      policies
        .filter((policy) => policy.id !== currentPolicyId)
        .map((policy) => ({
          value: policy.id,
          label: policy.isDefault ? `${policy.name} (default)` : policy.name,
          description: policy.code,
        })),
    [currentPolicyId, policies],
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
        toast.error("Policy not assigned", { description: failure.error });
        return;
      }
      toast.success("Policy assigned", {
        description: "Accruals start from the effective date on the next nightly run.",
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
          <DialogTitle>{currentPolicyId ? "Change PTO Policy" : "Assign PTO Policy"}</DialogTitle>
          <DialogDescription>
            {currentPolicyId
              ? "The current assignment ends the day before the new one starts. Balances carry across unchanged."
              : "Enrol this worker in a policy so their time off accrues and is tracked."}
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
                <SelectField<AssignPTOPolicyFormValues>
                  control={control}
                  name="ptoPolicyId"
                  label="Policy"
                  options={policyOptions}
                  rules={{ required: true }}
                  placeholder={isLoading ? "Loading policies..." : "Pick a policy"}
                  description="Decides how time off accrues and which PTO types are tracked."
                />
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<AssignPTOPolicyFormValues>
                  control={control}
                  name="effectiveFrom"
                  label="Effective from"
                  placeholder="Today"
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
                    <p className="text-sm font-medium">Opening balances</p>
                    <p className="text-muted-foreground text-xs">
                      Days the worker already has banked. Posted once as an opening balance.
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
                          description="Posted on the effective date; leave 0 for none."
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
                  label="Assignment note"
                  placeholder="e.g. Moved to the regional driver policy"
                  maxLength={255}
                  description="Optional; kept on the assignment record."
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                type="button"
                onClick={() => void handleSubmit(onSubmit)()}
                isLoading={isPending}
                loadingText="Assigning..."
              >
                Assign Policy
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
