import { AccessorialChargeAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
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
import { accessorialChargeMethodChoices } from "@/lib/choices";
import {
  accessorialChargeMethodSchema,
  type AccessorialCharge,
} from "@/types/accessorial-charge";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useRef } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const chargeFormSchema = z.object({
  accessorialChargeId: z.string().min(1, "Required"),
  method: accessorialChargeMethodSchema,
  amount: z.number().min(0, "Must be positive"),
  unit: z.number().int().min(1, "Must be at least 1").default(1),
});

type ChargeFormValues = z.infer<typeof chargeFormSchema>;

export type ChargeDialogResult = ChargeFormValues & {
  id?: string;
  accessorialCharge?: AccessorialCharge | null;
};

export function BillingQueueChargeDialog({
  open,
  onOpenChange,
  onSave,
  defaultValues,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (values: ChargeDialogResult) => void;
  defaultValues?: Partial<ChargeDialogResult>;
}) {
  const isEditing = !!defaultValues?.accessorialChargeId;
  const accessorialRef = useRef<AccessorialCharge | null>(null);

  const form = useForm({
    resolver: zodResolver(chargeFormSchema),
    defaultValues: {
      accessorialChargeId: defaultValues?.accessorialChargeId ?? "",
      method: defaultValues?.method ?? "Flat",
      amount: defaultValues?.amount ?? 0,
      unit: defaultValues?.unit ?? 1,
    },
  });

  const { control, handleSubmit, setValue, reset } = form;

  const handleChargeSelected = useCallback(
    (option: AccessorialCharge | null) => {
      if (option) {
        accessorialRef.current = option;
        setValue("method", option.method, { shouldDirty: true });
        setValue("amount", Number(option.amount), { shouldDirty: true });
      }
    },
    [setValue],
  );

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
    accessorialRef.current = null;
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    (values: ChargeFormValues) => {
      onSave({
        ...values,
        id: defaultValues?.id,
        accessorialCharge:
          accessorialRef.current ?? defaultValues?.accessorialCharge,
      });
      handleClose();
    },
    [onSave, defaultValues, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit Charge" : "Add Charge"}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Update the accessorial charge details"
              : "Select an accessorial charge and configure billing details"}
          </DialogDescription>
        </DialogHeader>
        <Form onSubmit={handleSubmit(onSubmit)}>
          <div className="pb-4">
            <FormGroup cols={2}>
              <FormControl className="col-span-2">
                <AccessorialChargeAutocompleteField
                  control={control}
                  name="accessorialChargeId"
                  label="Accessorial Charge"
                  clearable
                  rules={{ required: true }}
                  placeholder="Select Accessorial Charge"
                  onOptionChange={handleChargeSelected}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="unit"
                  label="Unit"
                  rules={{ required: true, min: 1 }}
                  placeholder="Unit"
                  sideText="unit(s)"
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="method"
                  label="Method"
                  options={accessorialChargeMethodChoices}
                  rules={{ required: true }}
                  placeholder="Select Method"
                />
              </FormControl>
              <FormControl className="col-span-2">
                <NumberField
                  control={control}
                  name="amount"
                  label="Amount"
                  decimalScale={2}
                  rules={{ required: true, min: 0 }}
                  placeholder="Amount"
                  sideText="USD"
                />
              </FormControl>
            </FormGroup>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit">Save</Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
