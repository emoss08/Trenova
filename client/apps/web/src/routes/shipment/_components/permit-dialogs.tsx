import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { permitViewQueryKeys } from "@/lib/queries/shipment";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { describeRequirement } from "@trenova/shared/lib/permit";
import {
  permitCreateSchema,
  waiveRequirementSchema,
  type Permit,
  type PermitCreateInput,
  type PermitRequirement,
  type WaiveRequirementInput,
} from "@trenova/shared/types/permit";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { TriangleAlertIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";

function useInvalidatePermitViews(shipmentId: string) {
  const queryClient = useQueryClient();

  return () => {
    for (const queryKey of permitViewQueryKeys(shipmentId)) {
      void queryClient.invalidateQueries({ queryKey });
    }
  };
}

/**
 * The dialog's starting values.
 *
 * Editing an existing permit fills from it. Recording a new one seeds the
 * jurisdiction from the requirement it was opened from — that is the point of
 * opening the dialog from a row, since the operator already told us which
 * jurisdiction by clicking it.
 */
function permitDefaults(
  requirement: PermitRequirement | null,
  permit: Permit | null,
): PermitCreateInput {
  if (permit) {
    return {
      stateId: permit.stateId,
      permitNumber: permit.permitNumber,
      status: permit.status,
      issuedAt: permit.issuedAt ?? null,
      expiresAt: permit.expiresAt ?? null,
      cost: permit.cost ?? null,
      notes: permit.notes ?? "",
    };
  }

  return {
    stateId: requirement?.stateId ?? "",
    permitNumber: "",
    status: "Active",
    issuedAt: null,
    expiresAt: null,
    cost: null,
    notes: "",
  };
}

export function PermitRecordDialog({
  open,
  onOpenChange,
  shipmentId,
  requirement,
  permit = null,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId: string;
  requirement: PermitRequirement | null;
  /** When set the dialog edits this permit instead of recording a new one. */
  permit?: Permit | null;
}) {
  const invalidate = useInvalidatePermitViews(shipmentId);
  const isEdit = permit !== null;

  const form = useForm<PermitCreateInput>({
    resolver: zodResolver(permitCreateSchema),
    defaultValues: permitDefaults(requirement, permit),
  });

  const { control, handleSubmit, reset } = form;

  // defaultValues only apply on mount, and this dialog stays mounted while the
  // operator moves between rows. Without this, opening it for Georgia and then
  // for Tennessee would file the Tennessee permit against Georgia.
  useEffect(() => {
    if (open) reset(permitDefaults(requirement, permit));
  }, [open, requirement, permit, reset]);

  const mutation = useApiMutation({
    mutationFn: async (values: PermitCreateInput) =>
      isEdit && permit?.id
        ? apiService.shipmentService.updatePermit(shipmentId, permit.id, values)
        : apiService.shipmentService.createPermit(shipmentId, values),
    onSuccess: () => {
      toast.success(isEdit ? "Permit updated" : "Permit recorded", {
        // Both paths re-derive server-side, so a correction can release a hold
        // just as recording one can — or re-raise it, if the expiry moved in.
        description: "Requirements are re-checked against this permit.",
      });
      invalidate();
      reset();
      onOpenChange(false);
    },
    form,
    resourceName: "Permit",
  });

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) reset();
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Permit" : "Record Permit"}</DialogTitle>
          <DialogDescription>
            {requirement
              ? describeRequirement(requirement)
              : isEdit
                ? `${permit?.state?.abbreviation ?? ""} ${permit?.permitNumber ?? ""}`.trim()
                : "Record a permit against this shipment"}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <FormGroup cols={2}>
            <FormControl cols="full">
              <InputField
                control={control}
                name="permitNumber"
                label="Permit Number"
                placeholder="Enter the number issued by the state"
                rules={{ required: true }}
                maxLength={100}
              />
            </FormControl>
            <FormControl cols="full">
              <UsStateAutocompleteField
                control={control}
                name="stateId"
                label="Issuing State"
                placeholder="Select issuing state"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <DateField
                control={control}
                name="issuedAt"
                label="Issued"
                placeholder="Select issue date"
                clearable
              />
            </FormControl>
            <FormControl>
              <DateField
                control={control}
                name="expiresAt"
                label="Expires"
                description="Must still cover the final stop, not just dispatch."
                placeholder="Select expiry date"
                rules={{ required: true }}
                clearable
              />
            </FormControl>
            <FormControl cols="full">
              <MoneyField control={control} name="cost" label="Cost" placeholder="0.00" />
            </FormControl>
            <FormControl cols="full">
              <TextareaField
                control={control}
                name="notes"
                label="Notes"
                placeholder="Routing conditions, escort details, anything the driver needs"
              />
            </FormControl>
          </FormGroup>
        </FormProvider>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit((values) => mutation.mutate(values))}
            disabled={mutation.isPending}
          >
            {mutation.isPending
              ? isEdit
                ? "Saving..."
                : "Recording..."
              : isEdit
                ? "Save Permit"
                : "Record Permit"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function PermitWaiveDialog({
  open,
  onOpenChange,
  shipmentId,
  requirement,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId: string;
  requirement: PermitRequirement | null;
}) {
  const invalidate = useInvalidatePermitViews(shipmentId);

  const form = useForm<WaiveRequirementInput>({
    resolver: zodResolver(waiveRequirementSchema),
    defaultValues: { reason: "" },
  });

  const { control, handleSubmit, reset } = form;

  const mutation = useApiMutation({
    mutationFn: async (values: WaiveRequirementInput) =>
      apiService.shipmentService.waivePermitRequirement(
        shipmentId,
        requirement?.id ?? "",
        values.reason,
      ),
    onSuccess: () => {
      toast.success("Requirement waived", {
        description: "The waiver and its reason are recorded against this shipment.",
      });
      invalidate();
      reset();
      onOpenChange(false);
    },
    form,
    resourceName: "Permit Requirement",
  });

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) reset();
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Waive Requirement</DialogTitle>
          <DialogDescription>
            {requirement ? describeRequirement(requirement) : "Waive this permit requirement"}
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-start gap-2 rounded-lg border border-yellow-600/30 bg-yellow-600/10 px-4 py-3">
          <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0 text-yellow-700 dark:text-yellow-400" />
          <p className="text-xs text-muted-foreground">
            Waiving does not make the movement legal. It records that your organization accepts the
            compliance risk and releases the dispatch block. Your reason is the audit trail if the
            load is stopped.
          </p>
        </div>

        <FormProvider {...form}>
          <FormGroup cols={1}>
            <FormControl cols="full">
              <TextareaField
                control={control}
                name="reason"
                label="Reason"
                description="At least 10 characters. Say what makes this movement acceptable."
                placeholder="Escort booked and route surveyed with the state"
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
        </FormProvider>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleSubmit((values) => mutation.mutate(values))}
            disabled={mutation.isPending}
          >
            {mutation.isPending ? "Waiving..." : "Waive Requirement"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
