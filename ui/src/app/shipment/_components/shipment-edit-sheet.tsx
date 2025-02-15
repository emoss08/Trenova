import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { shipmentSchema, ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { Shipment } from "@/types/shipment";
import { yupResolver } from "@hookform/resolvers/yup";
import { useEffect, useRef, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useShipmentDetails } from "../queries/shipment";
import { ShipmentForm } from "./sidebar/form/shipment-form";

export function ShipmentEditSheet({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<Shipment>) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const isMountedRef = useRef(false);
  const [dimensions, setDimensions] = useState({
    contentHeight: 0,
    viewportHeight: 0,
  });

  const shipmentDetails = useShipmentDetails({
    shipmentId: currentRecord?.id ?? "",
    enabled: !!currentRecord?.id,
  });

  const isDetailsLoading = shipmentDetails.isLoading;

  const form = useForm<ShipmentSchema>({
    resolver: yupResolver(shipmentSchema),
    defaultValues: {},
  });

  const {
    setError,
    formState: { isDirty, isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (shipmentDetails.data && !isDetailsLoading) {
      reset(shipmentDetails.data);
    }
  }, [shipmentDetails.data, isDetailsLoading, reset]);

  useEffect(() => {
    if (!open) return;

    const updateDimensions = () => {
      if (sheetRef.current) {
        const contentHeight = sheetRef.current.getBoundingClientRect().height;
        const viewportHeight = window.innerHeight;

        // Only update if we have valid measurements
        if (contentHeight > 0 && viewportHeight > 0) {
          setDimensions({
            contentHeight,
            viewportHeight,
          });
        }
      }
    };

    // Initial setup with a small delay to ensure proper rendering
    const initialTimer = setTimeout(() => {
      isMountedRef.current = true;
      updateDimensions();
    }, 100);

    // Create a ResizeObserver for the sheet content
    const resizeObserver = new ResizeObserver(() => {
      if (isMountedRef.current) {
        updateDimensions();
      }
    });

    if (sheetRef.current) {
      resizeObserver.observe(sheetRef.current);
    }

    // Handle window resize
    const handleResize = () => {
      if (isMountedRef.current) {
        updateDimensions();
      }
    };
    window.addEventListener("resize", handleResize);

    return () => {
      clearTimeout(initialTimer);
      resizeObserver.disconnect();
      window.removeEventListener("resize", handleResize);
      isMountedRef.current = false;
    };
  }, [open]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        className="w-[1000px] sm:max-w-[500px] p-0"
        withClose={false}
        ref={sheetRef}
      >
        <VisuallyHidden>
          <SheetHeader>
            <SheetTitle>Shipment Details</SheetTitle>
          </SheetHeader>
          <SheetDescription>Test</SheetDescription>
        </VisuallyHidden>

        <FormProvider {...form}>
          <Form>
            <SheetBody className="p-0">
              {shipmentDetails.data && (
                <ShipmentForm
                  dimensions={dimensions}
                  selectedShipment={shipmentDetails.data}
                  isLoading={isDetailsLoading}
                  onBack={() => onOpenChange(false)}
                />
              )}
            </SheetBody>
            <SheetFooter className="p-3">
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button>Save Changes</Button>
            </SheetFooter>
          </Form>
        </FormProvider>
      </SheetContent>
    </Sheet>
  );
}
