"use no memo";
import { EmptyState } from "@/components/empty-state";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { AccessorialCharge } from "@/types/accessorial-charge";
import type { Shipment } from "@/types/shipment";
import {
  BoxesIcon,
  PencilIcon,
  PlusIcon,
  ReceiptIcon,
  TrashIcon,
  TriangleAlertIcon,
  TruckIcon,
} from "lucide-react";
import { useState } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { AdditionalChargeDialog } from "./shipment-additional-charges-dialog";

export default function AdditionalChargesSection() {
  const {
    control,
    formState: { errors },
  } = useFormContext<Shipment>();
  const { fields, append, update, remove } = useFieldArray({
    control,
    name: "additionalCharges",
    keyName: "fieldId",
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const charges = useWatch({ control, name: "additionalCharges" });

  function handleAdd() {
    const newIndex = fields.length;
    append({
      accessorialChargeId: "",
      isSystemGenerated: false,
      method: "Flat",
      amount: 0,
      unit: 1,
    });
    setEditingIndex(newIndex);
    setIsEditing(false);
    setDialogOpen(true);
  }

  function handleEdit(index: number) {
    setEditingIndex(index);
    setIsEditing(true);
    setDialogOpen(true);
  }

  function handleDialogCancel() {
    if (editingIndex !== null && !isEditing) {
      remove(editingIndex);
    }
    setDialogOpen(false);
    setEditingIndex(null);
  }

  function handleDialogSave() {
    setDialogOpen(false);
    setEditingIndex(null);
  }

  return (
    <>
      <FormSection
        title="Additional Charges"
        description="Additional fees charged for services such as detention, fuel surcharge, and more."
        action={
          fields.length > 0 && (
            <Button type="button" variant="outline" size="xxs" onClick={handleAdd}>
              <PlusIcon className="size-3" />
              Add Charge
            </Button>
          )
        }
      >
        {fields.length > 0 ? (
          <div className="rounded-lg border">
            <div className="grid grid-cols-10 gap-2 border-b border-border px-4 py-2 text-2xs text-muted-foreground uppercase">
              <span className="col-span-4">Charge</span>
              <span className="col-span-2">Unit</span>
              <span className="col-span-2">Amount</span>
              <span className="col-span-2" />
            </div>
            <div className="divide-y">
              {fields.map((field, index) => {
                const charge = charges[index];
                const chargeObj = (charge as any)?.accessorialCharge as
                  | AccessorialCharge
                  | undefined;
                const displayName = chargeObj?.code ?? "—";
                const amt = Number(charge?.amount) || 0;

                const chargeErrors = errors.additionalCharges?.[index];
                const hasErrors = !!(chargeErrors && Object.keys(chargeErrors).length > 0);
                const errorMessages = hasErrors
                  ? Object.entries(chargeErrors as Record<string, { message?: string }>)
                      .filter(([key]) => key !== "ref" && key !== "root")
                      .map(([, err]) => err?.message ?? "Invalid")
                  : [];

                return (
                  <div
                    key={field.fieldId}
                    className={cn(
                      "grid grid-cols-10 items-center gap-2 px-4 py-2",
                      hasErrors && "bg-destructive/10 ring-1 ring-destructive ring-inset",
                    )}
                  >
                    <span className="col-span-4 truncate text-xs font-medium">{displayName}</span>
                    <span className="col-span-2 text-xs text-muted-foreground">
                      {charge?.unit ?? 1}
                    </span>
                    <span className="col-span-2 text-xs text-muted-foreground">
                      ${amt.toFixed(2)}
                    </span>
                    <div className="col-span-2 flex items-center justify-end gap-1">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-7"
                        onClick={() => handleEdit(index)}
                      >
                        <PencilIcon className="size-3.5 text-muted-foreground" />
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-7"
                        onClick={() => remove(index)}
                      >
                        <TrashIcon className="size-3.5 text-muted-foreground" />
                      </Button>
                      {hasErrors && (
                        <Tooltip>
                          <TooltipTrigger>
                            <TriangleAlertIcon className="size-3.5 cursor-help text-destructive" />
                          </TooltipTrigger>
                          <TooltipContent side="top" sideOffset={10}>
                            <div className="space-y-1">
                              {errorMessages.map((msg, idx) => (
                                <p key={idx} className="text-xs">
                                  {msg}
                                </p>
                              ))}
                            </div>
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        ) : (
          <EmptyState
            title="No Additional Charges"
            description="Shipment has no associated additional charges"
            icons={[ReceiptIcon, BoxesIcon, TruckIcon]}
            className="border-bg-sidebar-border max-h-[200px] rounded-lg border p-4"
            action={{
              label: "Add First Charge",
              onClick: handleAdd,
              icon: PlusIcon,
            }}
          />
        )}
      </FormSection>
      {editingIndex !== null && (
        <AdditionalChargeDialog
          open={dialogOpen}
          onCancel={handleDialogCancel}
          onSave={handleDialogSave}
          index={editingIndex}
          isEditing={isEditing}
          update={update}
        />
      )}
    </>
  );
}
