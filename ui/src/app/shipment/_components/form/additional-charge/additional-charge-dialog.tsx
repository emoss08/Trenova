import { SelectField } from "@/components/fields/select-field";
import { AccessorialChargeAutocompleteField } from "@/components/ui/autocomplete-fields";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { accessorialChargeMethodChoices } from "@/lib/choices";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { type TableSheetProps } from "@/types/data-table";
import { useCallback, useEffect } from "react";
import {
  type UseFieldArrayRemove,
  type UseFieldArrayUpdate,
  useFormContext,
} from "react-hook-form";

interface AdditionalChargeDialogProps extends TableSheetProps {
  index: number;
  isEditing: boolean;
  update: UseFieldArrayUpdate<ShipmentSchema, "additionalCharges">;
  remove: UseFieldArrayRemove;
}

export function AdditionalChargeDialog({
  open,
  onOpenChange,
  isEditing,
  update,
  index,
  remove,
}: AdditionalChargeDialogProps) {
  const { getValues, reset } = useFormContext<ShipmentSchema>();

  const handleSave = useCallback(() => {
    const formValues = getValues();
    const additionalCharge = formValues.additionalCharges?.[index];

    if (additionalCharge?.accessorialChargeId) {
      const updatedAdditionalCharge = {
        accessorialChargeId: additionalCharge.accessorialChargeId,
        accessorialCharge: additionalCharge.accessorialCharge,
        unit: additionalCharge.unit,
        method: additionalCharge.method,
        amount: additionalCharge.amount,
        // Preserve the existing ID if editing, otherwise it will be handled by the backend
        id: isEditing ? additionalCharge.id : undefined,
        shipmentId: formValues?.id || "",
      };

      update?.(index, updatedAdditionalCharge);
    }

    onOpenChange(false);
  }, [onOpenChange, getValues, index, isEditing, update]);

  const handleClose = useCallback(() => {
    onOpenChange(false);

    if (!isEditing) {
      remove(index);
    } else {
      const originalValues = getValues();
      const additionalCharges = originalValues?.additionalCharges || [];

      reset(
        {
          additionalCharges: [
            ...additionalCharges.slice(0, index),
            additionalCharges[index],
            ...additionalCharges.slice(index + 1),
          ],
        },
        {
          keepValues: true,
        },
      );
    }
  }, [onOpenChange, remove, index, isEditing, reset, getValues]);

  // Handle keyboard shortcut (Ctrl+Enter) to save
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === "Enter" && open) {
        handleSave();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, handleSave]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? "Edit" : "Add"} Additional Charge
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Edit the existing additional charge"
              : "Add a new additional charge to the existing shipment."}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <AdditionalChargeForm index={index} />
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button onClick={handleSave}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AdditionalChargeForm({ index }: { index: number }) {
  const { control, setValue } = useFormContext<ShipmentSchema>();

  // const additionalCharge = watch(`additionalCharges.${index}`);

  // // Add a ref to track previous accessorialChargeId to detect changes
  // const prevAccessorialChargeIdRef = useRef<string | undefined>(
  //   additionalCharge?.accessorialChargeId,
  // );

  // useEffect(() => {
  //   // Only set default values when the accessorialChargeId changes or when it's first selected
  //   const currentAccessorialChargeId = additionalCharge?.accessorialChargeId;
  //   const accessorialCharge = additionalCharge?.accessorialCharge;

  //   if (
  //     currentAccessorialChargeId &&
  //     accessorialCharge &&
  //     currentAccessorialChargeId !== prevAccessorialChargeIdRef.current
  //   ) {
  //     // Only set default values from the accessorial charge when it's newly selected
  //     setValue(`additionalCharges.${index}.unit`, accessorialCharge.unit);
  //     setValue(`additionalCharges.${index}.method`, accessorialCharge.method);
  //     setValue(`additionalCharges.${index}.amount`, accessorialCharge.amount);

  //     // Update the ref to the current ID
  //     prevAccessorialChargeIdRef.current = currentAccessorialChargeId;
  //   }
  // }, [
  //   additionalCharge?.accessorialChargeId,
  //   additionalCharge?.accessorialCharge,
  //   setValue,
  //   index,
  // ]);

  return (
    <AdditionalChargeFormInner>
      <FormControl cols="full">
        <AccessorialChargeAutocompleteField<ShipmentSchema>
          name={`additionalCharges.${index}.accessorialChargeId`}
          control={control}
          label="Accessorial Charge"
          clearable
          rules={{ required: true }}
          placeholder="Select Accessorial Charge"
          description="Accessorial charges are additional fees charged for services such as detention, fuel surcharge, and more."
          onOptionChange={(option) => {
            if (option) {
              setValue(
                `additionalCharges.${index}.accessorialChargeId`,
                option.id || "",
              );
              setValue(`additionalCharges.${index}.accessorialCharge`, option);
            }
          }}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name={`additionalCharges.${index}.unit`}
          label="Unit"
          rules={{ required: true, min: 1 }}
          placeholder="Unit"
          description="Quantity of units this charge applies to (number of pallets, hours of detention, etc.)"
          sideText="unit(s)"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name={`additionalCharges.${index}.method`}
          label="Method"
          options={accessorialChargeMethodChoices}
          rules={{ required: true }}
          placeholder="Select Method"
          description="Calculation method for this charge (flat rate, per mile, percentage of linehaul, etc.)"
        />
      </FormControl>
      <FormControl cols="full">
        <NumberField
          formatOptions={{
            style: "currency",
            currency: "USD",
          }}
          control={control}
          name={`additionalCharges.${index}.amount`}
          label="Amount"
          rules={{ required: true, min: 1 }}
          placeholder="Amount"
          description="Dollar value per unit for this accessorial service, used to calculate total charges for billing and settlement"
        />
      </FormControl>
    </AdditionalChargeFormInner>
  );
}

function AdditionalChargeFormInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <FormGroup cols={2}>{children}</FormGroup>;
}
