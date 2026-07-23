import { ControlledShipmentAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl, FormGroup } from "@/components/ui/form";
import { graphQLErrorMessage } from "@/lib/graphql";
import { attachOrderShipments } from "@/lib/graphql/order";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { useMutation } from "@tanstack/react-query";
import { XIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { useOrderInvalidation } from "./use-order-invalidation";

type AddLegDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
  customerId?: string;
};

type SelectedLeg = {
  id: string;
  label: string;
};

export function AddLegDialog({ open, onOpenChange, orderId, customerId }: AddLegDialogProps) {
  const invalidateOrders = useOrderInvalidation();
  const [pickerValue, setPickerValue] = useState("");
  const [selected, setSelected] = useState<SelectedLeg[]>([]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    setPickerValue("");
    setSelected([]);
  }, [onOpenChange]);

  const handleOptionChange = useCallback((option: GraphQLSelectOption | null) => {
    if (!option?.id) return;
    const id = option.id;
    setSelected((current) =>
      current.some((leg) => leg.id === id)
        ? current
        : [...current, { id, label: option.label || id }],
    );
    setPickerValue("");
  }, []);

  const { mutate, isPending } = useMutation({
    mutationFn: () =>
      attachOrderShipments(
        orderId,
        selected.map((leg) => leg.id),
      ),
    onSuccess: () => {
      invalidateOrders();
      toast.success(selected.length === 1 ? "Leg added" : `${selected.length} legs added`, {
        description: "The shipments have been attached to this order.",
      });
      handleClose();
    },
    onError: (error) => {
      toast.error("Failed to add legs", {
        description: graphQLErrorMessage(error, "The shipments could not be attached."),
      });
    },
  });

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>Add Legs</DialogTitle>
          <DialogDescription>
            Attach one or more shipments to this order as additional legs.
          </DialogDescription>
        </DialogHeader>
        <FormGroup cols={1} className="pb-4">
          <FormControl>
            <ControlledShipmentAutocompleteField
              value={pickerValue}
              onValueChange={setPickerValue}
              onOptionChange={handleOptionChange}
              disabled={!customerId}
              description={
                customerId
                  ? "Only attachable shipments for this order's customer are shown."
                  : "Loading the order's customer..."
              }
              extraSearchParams={
                customerId
                  ? { customerId, attachableOnly: "true", excludeOrderId: orderId }
                  : undefined
              }
            />
          </FormControl>
          {selected.length > 0 && (
            <ul className="flex flex-col gap-1">
              {selected.map((leg) => (
                <li
                  key={leg.id}
                  className="flex items-center justify-between rounded-md border px-2 py-1 text-sm"
                >
                  <span className="truncate font-mono">{leg.label}</span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() =>
                      setSelected((current) => current.filter((item) => item.id !== leg.id))
                    }
                    aria-label={`Remove ${leg.label}`}
                  >
                    <XIcon className="size-3" />
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </FormGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={selected.length === 0}
            isLoading={isPending}
            loadingText="Adding..."
            onClick={() => mutate()}
          >
            {selected.length > 1 ? `Add ${selected.length} Legs` : "Add Leg"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
