import { CommodityAutocompleteField } from "@/components/autocomplete-fields";
import { EmptyState } from "@/components/empty-state";
import { NumberField } from "@/components/fields/number-field";
import { EntityRedirectLink } from "@/components/link";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { queries } from "@/lib/queries";
import { cn, findDuplicateIds, pluralize, truncateText } from "@/lib/utils";
import { ApiRequestError } from "@/lib/api";
import { apiService } from "@/services/api";
import type { Commodity } from "@/types/commodity";
import type { Shipment } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import {
  AlertCircleIcon,
  BiohazardIcon,
  BoxesIcon,
  CaravanIcon,
  PencilIcon,
  PlusIcon,
  TrashIcon,
  TriangleAlertIcon,
  TruckIcon,
} from "lucide-react";
import { useState } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

function CommodityDialog({
  open,
  onCancel,
  onSave,
  index,
  isEditing,
  maxShipmentWeightLimit,
  checkHazmatSegregation,
  update,
}: {
  open: boolean;
  onCancel: () => void;
  onSave: () => void;
  index: number;
  isEditing: boolean;
  maxShipmentWeightLimit?: number;
  checkHazmatSegregation?: boolean;
  update: (index: number, value: any) => void;
}) {
  const { control, setValue, getValues, setError, clearErrors } =
    useFormContext<Shipment>();
  const [saving, setSaving] = useState(false);

  function handleCommoditySelected(option: Commodity | null) {
    setValue(`commodities.${index}.commodity`, option ?? undefined);
  }

  async function handleSave() {
    const values = getValues(`commodities.${index}`);
    const commodities = getValues("commodities") ?? [];
    const nextWeight = typeof values.weight === "number" ? values.weight : 0;
    const totalWeightExcludingCurrent = commodities.reduce(
      (sum, commodity, commodityIndex) => {
        if (commodityIndex === index) {
          return sum;
        }
        return (
          sum + (typeof commodity?.weight === "number" ? commodity.weight : 0)
        );
      },
      0,
    );

    if (
      typeof maxShipmentWeightLimit === "number" &&
      totalWeightExcludingCurrent + nextWeight > maxShipmentWeightLimit
    ) {
      setError(`commodities.${index}.weight`, {
        type: "manual",
        message: `Total commodity weight cannot exceed ${maxShipmentWeightLimit.toLocaleString()} lbs`,
      });
      return;
    }

    clearErrors(`commodities.${index}.weight`);

    if (checkHazmatSegregation) {
      const allCommodityIds = commodities
        .map((c, i) => (i === index ? values.commodityId : c?.commodityId))
        .filter((id): id is string => !!id);

      if (allCommodityIds.length >= 2) {
        setSaving(true);
        try {
          await apiService.shipmentService.checkHazmatSegregation(
            allCommodityIds,
          );
          for (let i = 0; i < commodities.length; i++) {
            clearErrors(`commodities.${i}.commodityId`);
          }
        } catch (err) {
          if (err instanceof ApiRequestError && err.isValidationError()) {
            for (const fieldError of err.getFieldErrors()) {
              setError(fieldError.field as any, {
                type: "manual",
                message: fieldError.message,
              });
            }
            update(index, values);
            setSaving(false);
            return;
          }
        } finally {
          setSaving(false);
        }
      }
    }

    update(index, values);
    onSave();
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) onCancel();
      }}
    >
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? "Edit Commodity" : "Add Commodity"}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Update the commodity details"
              : "Select a commodity and specify quantity and weight"}
          </DialogDescription>
        </DialogHeader>
        <FormGroup cols={1}>
          <FormControl>
            <CommodityAutocompleteField
              control={control}
              name={`commodities.${index}.commodityId`}
              label="Commodity"
              placeholder="Select commodity"
              onOptionChange={handleCommoditySelected}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`commodities.${index}.pieces`}
              label="Pieces"
              placeholder="1"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`commodities.${index}.weight`}
              label="Weight (lbs)"
              placeholder="0"
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            disabled={saving}
          >
            Cancel
          </Button>
          <Button type="button" onClick={handleSave} disabled={saving}>
            {saving ? "Checking..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default function CommoditiesSection() {
  const {
    control,
    formState: { errors },
  } = useFormContext<Shipment>();
  const { fields, append, update, remove } = useFieldArray({
    control,
    name: "commodities",
    keyName: "fieldId",
  });
  const { data: shipmentUIPolicy } = useQuery({
    ...queries.shipment.uiPolicy(),
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const commodities = useWatch({ control, name: "commodities" }) ?? [];

  const duplicateCommodityIds = findDuplicateIds(
    commodities,
    (c) => c?.commodityId,
  );

  const totalPieces = commodities.reduce(
    (sum, c) => sum + (typeof c.pieces === "number" ? c.pieces : 0),
    0,
  );
  const totalWeight = commodities.reduce(
    (sum, c) => sum + (typeof c.weight === "number" ? c.weight : 0),
    0,
  );

  function handleAdd() {
    const newIndex = fields.length;
    append({
      commodityId: "",
      pieces: 1,
      weight: 0,
    } as any);
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
        title="Commodities"
        titleCount={commodities.length}
        description="Cargo items, weights, and hazardous material compliance"
        className="border-t border-border pt-4"
        action={
          fields.length > 0 && (
            <Button
              type="button"
              variant="outline"
              size="xxs"
              onClick={handleAdd}
            >
              <PlusIcon className="size-3" />
              Add Commodity
            </Button>
          )
        }
      >
        {fields.length > 0 ? (
          <div className="rounded-lg border">
            <div className="grid grid-cols-10 gap-2 border-b border-border px-4 py-2 text-2xs text-muted-foreground uppercase">
              <span className="col-span-4">Commodity</span>
              <span className="col-span-2">Pieces</span>
              <span className="col-span-2">Weight</span>
              <span className="col-span-2" />
            </div>
            <div className="divide-y">
              {fields.map((field, index) => {
                const item = commodities[index];
                const commodityObj = item?.commodity;
                const displayName = commodityObj?.name ?? "—";
                const hasHazmat = !!commodityObj?.hazardousMaterialId;
                const stackable = commodityObj?.stackable;
                const fragile = commodityObj?.fragile;
                const isDuplicate =
                  !!item?.commodityId &&
                  duplicateCommodityIds.has(item.commodityId);

                const commodityErrors = errors.commodities?.[index];
                const hasErrors = !!(
                  commodityErrors && Object.keys(commodityErrors).length > 0
                );
                const errorMessages = hasErrors
                  ? Object.entries(
                      commodityErrors as Record<string, { message?: string }>,
                    )
                      .filter(([key]) => key !== "ref" && key !== "root")
                      .map(([, err]) => err?.message ?? "Invalid")
                  : [];

                return (
                  <div
                    key={field.fieldId}
                    className={cn(
                      "grid grid-cols-10 items-center gap-2 px-4 py-2",
                      hasErrors &&
                        "bg-destructive/10 ring-1 ring-destructive ring-inset",
                      !hasErrors &&
                        isDuplicate &&
                        "bg-warning/20 ring-1 ring-warning ring-inset",
                    )}
                  >
                    <div className="col-span-4 flex items-center gap-1.5">
                      <EntityRedirectLink
                        entityId={commodityObj?.id}
                        baseUrl="/shipment-management/configuration-files/commodities"
                        panelOpen
                      >
                        <span className="text-xs font-medium">
                          {truncateText(displayName, 15)}
                        </span>
                      </EntityRedirectLink>
                      <div className="flex items-center gap-1">
                        {hasHazmat && (
                          <Tooltip>
                            <TooltipTrigger>
                              <BiohazardIcon className="size-3.5 cursor-help text-warning" />
                            </TooltipTrigger>
                            <TooltipContent side="top" sideOffset={10}>
                              Commodity is classified as hazardous material.
                            </TooltipContent>
                          </Tooltip>
                        )}
                        {stackable && (
                          <Tooltip>
                            <TooltipTrigger>
                              <BoxesIcon className="size-3.5 cursor-help text-success" />
                            </TooltipTrigger>
                            <TooltipContent side="top" sideOffset={10}>
                              Commodity is marked as stackable.
                            </TooltipContent>
                          </Tooltip>
                        )}
                        {fragile && (
                          <Tooltip>
                            <TooltipTrigger>
                              <AlertCircleIcon className="size-3.5 cursor-help text-destructive" />
                            </TooltipTrigger>
                            <TooltipContent side="top" sideOffset={10}>
                              Commodity is marked as fragile.
                            </TooltipContent>
                          </Tooltip>
                        )}
                      </div>
                    </div>
                    <span className="col-span-2 text-xs text-muted-foreground">
                      {truncateText(item?.pieces?.toLocaleString() ?? 0, 10)}
                    </span>
                    <span className="col-span-2 text-xs text-muted-foreground">
                      {truncateText(item?.weight?.toLocaleString() ?? 0, 8)} lbs
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
            <div className="flex flex-row items-center justify-end gap-3 rounded-b-lg border-t border-border bg-muted px-4 py-2">
              <span className="text-xs text-muted-foreground">
                {truncateText(totalPieces.toLocaleString(), 10)} total{" "}
                {pluralize("piece", totalPieces)}
              </span>
              <div className="flex flex-row items-center gap-0.5">
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <span className="cursor-help text-xs font-medium">
                          {truncateText(totalWeight.toLocaleString(), 10)} lbs
                        </span>
                      }
                    />
                    <TooltipContent side="top" sideOffset={10}>
                      Total weight of all commodities in the shipment (lbs).
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                {typeof shipmentUIPolicy?.maxShipmentWeightLimit ===
                  "number" && (
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <span className="cursor-help text-xs text-muted-foreground">
                            /{" "}
                            {shipmentUIPolicy.maxShipmentWeightLimit.toLocaleString()}{" "}
                            lbs
                          </span>
                        }
                      />
                      <TooltipContent side="top" sideOffset={10}>
                        Maximum shipment weight limit configured by
                        organization.
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                )}
              </div>
            </div>
          </div>
        ) : (
          <EmptyState
            className="border-bg-sidebar-border max-h-[200px] rounded-lg border p-4"
            title="No Commodities"
            description="Shipment has no associated commodities"
            icons={[CaravanIcon, BoxesIcon, TruckIcon]}
            action={{
              label: "Add First Commodity",
              onClick: handleAdd,
              icon: PlusIcon,
            }}
          />
        )}
        {duplicateCommodityIds.size > 0 && (
          <p className="flex items-center gap-1 text-xs text-warning">
            <TriangleAlertIcon className="size-3.5" />
            Duplicate commodities detected in this shipment.
          </p>
        )}
      </FormSection>
      {editingIndex !== null && (
        <CommodityDialog
          open={dialogOpen}
          onCancel={handleDialogCancel}
          onSave={handleDialogSave}
          index={editingIndex}
          isEditing={isEditing}
          maxShipmentWeightLimit={shipmentUIPolicy?.maxShipmentWeightLimit}
          checkHazmatSegregation={shipmentUIPolicy?.checkHazmatSegregation}
          update={update}
        />
      )}
    </>
  );
}
