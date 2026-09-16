"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { ChargeSplitEditor } from "@/components/billing/charge-split-editor";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import type { ChargeAllocation } from "@trenova/shared/types/shipment";
import { useEffect, useRef } from "react";
import { type Control, type FieldValues, useFormContext } from "react-hook-form";

export type ChargeSplitDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Path of the allocation array in the form, e.g. `freightAllocations`. */
  name: string;
  chargeAmount: number | null;
  percentOnly?: boolean;
  defaultPayer: { id: string; label: string } | null;
  title: string;
  description: string;
};

/**
 * Divides one charge among payers inside the surrounding form. Cancelling
 * restores the split that was there when the dialog opened, so a half-typed
 * change never lands on the form.
 */
export function ChargeSplitDialog({
  open,
  onOpenChange,
  name,
  chargeAmount,
  percentOnly = false,
  defaultPayer,
  title,
  description,
}: ChargeSplitDialogProps) {
  const t = useT();
  const { control, getValues, setValue, trigger } = useFormContext<FieldValues>();
  const snapshot = useRef<ChargeAllocation[]>([]);

  useEffect(() => {
    if (open) {
      snapshot.current = structuredClone((getValues(name) ?? []) as ChargeAllocation[]);
    }
  }, [open, name, getValues]);

  const handleCancel = () => {
    setValue(name, snapshot.current, { shouldDirty: true, shouldValidate: true });
    onOpenChange(false);
  };

  const handleSave = async () => {
    const valid = await trigger(name);
    if (!valid) return;
    onOpenChange(false);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) handleCancel();
      }}
    >
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <ChargeSplitEditor
          control={control as unknown as Control<FieldValues>}
          name={name}
          chargeAmount={chargeAmount}
          percentOnly={percentOnly}
          defaultPayer={defaultPayer}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleCancel}>
            {t("Cancel")}
          </Button>
          <Button type="button" onClick={() => void handleSave()}>
            {t("Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
