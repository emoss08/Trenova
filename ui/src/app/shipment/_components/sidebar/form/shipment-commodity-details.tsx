import { AutocompleteField } from "@/components/fields/autocomplete";
import { InputField } from "@/components/fields/input-field";
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
import { EmptyState } from "@/components/ui/empty-state";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import { VirtualizedScrollArea } from "@/components/ui/scroll-area";
import { CommoditySchema } from "@/lib/schemas/commodity-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { TableSheetProps } from "@/types/data-table";
import { ShipmentCommodity } from "@/types/shipment";
import {
  faBoxesStacked,
  faPencil,
  faPlus,
  faTrailer,
  faTruckContainer,
} from "@fortawesome/pro-solid-svg-icons";
import { useVirtualizer } from "@tanstack/react-virtual";
import { CSSProperties, memo, useCallback, useRef, useState } from "react";
import {
  useFieldArray,
  UseFieldArrayUpdate,
  useFormContext,
} from "react-hook-form";

const ROW_HEIGHT = 38;
const OVERSCAN = 5;

function CommodityRow({
  shipmentCommodity,
  style,
  isLast,
  onEdit,
  index,
}: {
  shipmentCommodity: ShipmentCommodity;
  style: CSSProperties;
  isLast: boolean;
  onEdit: (index: number) => void;
  index: number;
}) {
  if (!shipmentCommodity.commodity) return null;

  return (
    <div
      className={cn(
        "grid grid-cols-12 gap-4 p-2 text-sm",
        !isLast && "border-b border-border",
      )}
      style={style}
    >
      <div className="col-span-5">
        <EntityRedirectLink
          entityId={shipmentCommodity.commodity.id}
          baseUrl="/shipments/configurations/commodities"
          modelOpen
          value={shipmentCommodity.commodity.name}
        >
          {shipmentCommodity.commodity.name}
        </EntityRedirectLink>
      </div>
      <div className="col-span-3 text-left">{shipmentCommodity.pieces}</div>
      <div className="col-span-3 text-left">{shipmentCommodity.weight}</div>
      <div className="col-span-1">
        <Button variant="ghost" size="xs" onClick={() => onEdit(index)}>
          <Icon icon={faPencil} className="size-4" />
        </Button>
      </div>
    </div>
  );
}

CommodityRow.displayName = "CommodityRow";

const TableHeader = memo(() => (
  <div className="sticky top-0 z-10 grid grid-cols-12 gap-4 p-2 text-sm text-muted-foreground bg-card border-b border-border rounded-t-lg">
    <div className="col-span-5">Commodity</div>
    <div className="col-span-3 text-left">Pieces</div>
    <div className="col-span-3 text-left">Weight</div>
    <div className="col-span-1" />
  </div>
));

TableHeader.displayName = "TableHeader";

export function ShipmentCommodityDetails({
  className,
}: {
  className?: string;
}) {
  const [commodityDialogOpen, setCommodityDialogOpen] =
    useState<boolean>(false);
  const parentRef = useRef<HTMLDivElement>(null);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const { control } = useFormContext<ShipmentSchema>();
  const { fields: commodities, update } = useFieldArray({
    control,
    name: "commodities",
  });

  const handleAddCommodity = () => {
    setCommodityDialogOpen(true);
  };

  const handleEdit = (index: number) => {
    setEditingIndex(index);
    setCommodityDialogOpen(true);
  };

  const handleDialogClose = () => {
    setCommodityDialogOpen(false);
    setEditingIndex(null);
  };

  const virtualizer = useVirtualizer({
    count: commodities?.length ?? 0,
    getScrollElement: () => parentRef.current,
    estimateSize: useCallback(() => ROW_HEIGHT, []),
    overscan: OVERSCAN,
    enabled: !!commodities?.length,
  });

  if (!commodities?.length) {
    return (
      <div className="flex flex-col gap-2 border-t border-bg-sidebar-border py-4">
        <EmptyState
          className="max-h-[200px] p-4"
          title="No Commodities"
          description="Shipment has no associated commodities"
          icons={[faTrailer, faBoxesStacked, faTruckContainer]}
        />
      </div>
    );
  }

  return (
    <>
      <div
        className={cn(
          "flex flex-col gap-2 border-t border-bg-sidebar-border py-4",
          className,
        )}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1">
            <h3 className="text-sm font-medium">Commodities</h3>
            <span className="text-2xs text-muted-foreground">
              ({commodities?.length ?? 0})
            </span>
          </div>
          <Button variant="outline" size="xs" onClick={handleAddCommodity}>
            <Icon icon={faPlus} className="size-4" />
            Add Commodity
          </Button>
        </div>

        <div className="rounded-lg border border-bg-sidebar-border bg-card">
          <TableHeader />
          <VirtualizedScrollArea
            ref={parentRef}
            className="flex max-h-40 flex-col"
          >
            <div
              className="relative w-full rounded-b-lg"
              style={{ height: `${virtualizer.getTotalSize()}px` }}
            >
              {virtualizer.getVirtualItems().map((virtualRow) => {
                const shipmentCommodity = commodities[virtualRow.index];

                return (
                  <CommodityRow
                    key={shipmentCommodity.id}
                    shipmentCommodity={shipmentCommodity as ShipmentCommodity}
                    isLast={virtualRow.index === commodities.length - 1}
                    onEdit={handleEdit}
                    index={virtualRow.index}
                    style={{
                      position: "absolute",
                      top: 0,
                      left: 0,
                      width: "100%",
                      transform: `translateY(${virtualRow.start}px)`,
                    }}
                  />
                );
              })}
            </div>
          </VirtualizedScrollArea>
        </div>
      </div>
      {commodityDialogOpen && (
        <CommodityDialog
          open={commodityDialogOpen}
          onOpenChange={handleDialogClose}
          isEditing={editingIndex !== null}
          update={update}
          index={editingIndex ?? commodities.length}
        />
      )}
    </>
  );
}

interface CommodityDialogProps extends TableSheetProps {
  index: number;
  isEditing: boolean;
  update?: UseFieldArrayUpdate<ShipmentSchema, "commodities">;
  initialData?: ShipmentCommodity;
}
function CommodityDialog({
  open,
  onOpenChange,
  isEditing,
  update,
  index,
}: CommodityDialogProps) {
  const { control, setValue, getValues } = useFormContext<ShipmentSchema>();

  // const commodity = watch(`commodities.${index}.commodity`);

  const handleSave = () => {
    if (!isEditing) {
      const formValues = getValues();

      const commodity = formValues.commodities?.[index];

      if (commodity?.commodityId && commodity?.commodity) {
        console.log("appending new commodity", commodity);
        update?.(index, {
          commodityId: commodity.commodityId,
          commodity: commodity.commodity,
          pieces: commodity.pieces || 1,
          weight: commodity.weight || 0,
          shipmentId: "",
        });
      }
    }
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit" : "Add"} Commodity</DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Edit the existing commodity"
              : "Add a new commodity to the existing shipment."}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <FormGroup>
            <FormControl>
              <AutocompleteField<CommoditySchema, ShipmentSchema>
                name={`commodities.${index}.commodityId`}
                control={control}
                link="/commodities/"
                label="Commodity"
                clearable
                rules={{ required: true }}
                placeholder="Select Commodity"
                description="Select the commodity to be added to the shipment."
                getOptionValue={(option) => option.id || ""}
                getDisplayValue={(option) => `${option.name}`}
                renderOption={(option) => `${option.name}`}
                onOptionChange={(option) => {
                  if (option) {
                    setValue(
                      `commodities.${index}.commodityId`,
                      option.id || "",
                    );
                    setValue(`commodities.${index}.commodity`, option);
                  }
                }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name={`commodities.${index}.pieces`}
                label="Pieces"
                type="number"
                rules={{ required: true, min: 1 }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name={`commodities.${index}.weight`}
                label="Weight"
                type="number"
                rules={{ required: true, min: 0 }}
              />
            </FormControl>
          </FormGroup>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
