import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { getTodayDate } from "@/lib/date";
import {
  assignPayProfileToWorker,
  fetchPayProfileDetail,
  fetchPayProfileOptions,
} from "@/lib/graphql/driver-settlement";
import { assignPayProfileFormSchema, type AssignPayProfileFormValues } from "@/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export function AssignPayProfileDialog({
  open,
  onOpenChange,
  workerId,
  payProfileId,
  onAssigned,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId?: string;
  payProfileId?: string;
  onAssigned?: () => void;
}) {
  const queryClient = useQueryClient();
  const [overrides, setOverrides] = useState<Record<string, string>>({});

  const form = useForm<AssignPayProfileFormValues>({
    resolver: zodResolver(assignPayProfileFormSchema) as Resolver<AssignPayProfileFormValues>,
    defaultValues: {
      workerId: workerId ?? "",
      payProfileId: payProfileId ?? "",
      effectiveFrom: getTodayDate(),
      effectiveTo: null,
      splitPercent: 100,
      notes: "",
    },
  });
  const {
    control,
    setError,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  useEffect(() => {
    if (open) {
      reset({
        workerId: workerId ?? "",
        payProfileId: payProfileId ?? "",
        effectiveFrom: getTodayDate(),
        effectiveTo: null,
        splitPercent: 100,
        notes: "",
      });
      setOverrides({});
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, reset, workerId, payProfileId]);

  const selectedProfileId = useWatch({ control, name: "payProfileId" });

  const { data: profileOptions } = useQuery({
    queryKey: ["pay-profile-options"],
    queryFn: () => fetchPayProfileOptions(),
    enabled: open && !payProfileId,
  });

  const { data: selectedProfile } = useQuery({
    queryKey: ["pay-profile-detail", selectedProfileId],
    queryFn: () => fetchPayProfileDetail(selectedProfileId),
    enabled: open && !!selectedProfileId,
  });

  const mutation = useApiMutation({
    mutationFn: (values: AssignPayProfileFormValues) =>
      assignPayProfileToWorker({
        workerId: values.workerId,
        payProfileId: values.payProfileId,
        effectiveFrom: values.effectiveFrom,
        effectiveTo: values.effectiveTo ?? undefined,
        splitPercent: String(values.splitPercent),
        rateOverrides: Object.entries(overrides)
          .filter(([, rate]) => rate !== "" && !Number.isNaN(Number(rate)))
          .map(([componentId, rate]) => ({ componentId, rate })),
        notes: values.notes || undefined,
      }),
    onSuccess: () => {
      toast.success("Pay profile assigned");
      void queryClient.invalidateQueries({ queryKey: ["worker-pay"] });
      void queryClient.invalidateQueries({ queryKey: ["pay-profile-list"] });
      void queryClient.invalidateQueries({ queryKey: ["pay-profile-assignments"] });
      onOpenChange(false);
      onAssigned?.();
    },
    setFormError: setError,
    resourceName: "Pay Assignment",
  });

  const onSubmit = handleSubmit((values) => mutation.mutate(values));

  const activeComponents = (selectedProfile?.components ?? []).filter(
    (component) => component.isActive,
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Assign Pay Profile</DialogTitle>
          <DialogDescription>
            The driver&apos;s pay is computed from this profile for every shipment delivered on or
            after the effective date. Any currently-open assignment ends automatically — no cleanup
            needed.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form id="assign-pay-profile-form" onSubmit={onSubmit}>
            <FormGroup cols={2}>
              {!workerId && (
                <FormControl className="col-span-2">
                  <WorkerAutocompleteField
                    control={control}
                    name="workerId"
                    label="Driver"
                    placeholder="Select driver"
                    rules={{ required: true }}
                    description="The driver who will be paid under this profile from the effective date forward."
                  />
                </FormControl>
              )}
              {!payProfileId && (
                <FormControl className="col-span-2">
                  <SelectField
                    control={control}
                    name="payProfileId"
                    label="Pay Profile"
                    placeholder="Select pay profile"
                    options={(profileOptions ?? []).map((option) => ({
                      label: `${option.name}${
                        option.classification === "OwnerOperator" ? " (O-O)" : ""
                      }`,
                      value: option.id,
                    }))}
                    rules={{ required: true }}
                    description="Profiles are shared templates — set driver-specific rates below instead of cloning profiles."
                  />
                </FormControl>
              )}
              <FormControl>
                <AutoCompleteDateField
                  control={control}
                  name="effectiveFrom"
                  label="Effective From"
                  rules={{ required: true }}
                  description="Pay for shipments delivered on or after this date uses this assignment."
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="splitPercent"
                  label="Split Percent"
                  sideText="%"
                  description="100 for solo drivers; 50 each for an even team split."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl className="col-span-2">
                <TextareaField
                  control={control}
                  name="notes"
                  label="Notes"
                  placeholder="e.g. Negotiated rate bump at 1-year anniversary"
                  description="Why this assignment or rate was set — kept in the assignment history for audits."
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>

        {activeComponents.length > 0 && (
          <div className="rounded-lg border p-3">
            <p className="text-xs font-medium">Driver-Specific Rate Overrides</p>
            <p className="mb-2 text-[11px] text-muted-foreground">
              Leave blank to use the profile rate. An override replaces the component&apos;s base
              rate and any mileage bands for this driver only.
            </p>
            <div className="flex flex-col gap-2">
              {activeComponents.map((component) => (
                <div
                  key={component.id}
                  className="grid grid-cols-[1fr_auto_120px] items-center gap-2 text-xs"
                >
                  <span className="font-medium">
                    {component.description || `${component.kind} (${component.method})`}
                  </span>
                  <span className="text-muted-foreground tabular-nums">
                    profile: {Number(component.rate)}
                    {component.method === "PercentOfRevenue" ? "%" : ""}
                  </span>
                  <Input
                    value={overrides[component.id] ?? ""}
                    onChange={(e) =>
                      setOverrides((prev) => ({ ...prev, [component.id]: e.target.value }))
                    }
                    placeholder="Override"
                    inputMode="decimal"
                    className="h-7 text-xs"
                  />
                </div>
              ))}
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button form="assign-pay-profile-form" type="submit" disabled={isSubmitting}>
            Assign Profile
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
