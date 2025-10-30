import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { useQueryStates } from "nuqs";
import { ShipmentEditForm } from "./shipment-edit-form";

export function ShipmentEditSheet({
  currentRecord,
}: EditTableSheetProps<ShipmentSchema>) {
  const [, setSearchParams] = useQueryStates(searchParamsParser, {
    history: "push",
    throttleMs: 50,
  });
  console.info("shipment edit sheet render");

  return (
    <Sheet
      open={!!currentRecord?.id}
      onOpenChange={(open) => {
        if (!open) {
          setSearchParams({ modalType: null, entityId: null });
        }
      }}
    >
      <SheetContent
        className="w-[500px] sm:max-w-[540px] p-0"
        withClose={false}
      >
        <VisuallyHidden>
          <SheetHeader>
            <SheetTitle>Shipment Details</SheetTitle>
          </SheetHeader>
          <SheetDescription>{currentRecord?.bol}</SheetDescription>
        </VisuallyHidden>
        <SheetBody className="p-0">
          <ShipmentEditForm
            selectedShipment={currentRecord}
            isLoading={false}
          />
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}
