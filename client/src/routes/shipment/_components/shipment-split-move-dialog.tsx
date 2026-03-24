import { LocationAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { NumberField } from "@/components/fields/number-field";
import { Badge } from "@/components/ui/badge";
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
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatSplitDateTime } from "@/lib/date";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import type { ShipmentMove, SplitMovePayload, SplitMoveResponse, StopType } from "@/types/shipment";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { InfoIcon } from "lucide-react";
import { useCallback } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

type SplitMoveDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  moveId: string;
  currentMove: ShipmentMove;
  onSplit: (resp: SplitMoveResponse) => void;
};

type SplitMoveFormValues = {
  newDeliveryLocationId: string;
  splitPickupScheduledWindowStart: number;
  splitPickupScheduledWindowEnd: number | null;
  newDeliveryScheduledWindowStart: number;
  newDeliveryScheduledWindowEnd: number | null;
  pieces: number | null;
  weight: number | null;
};

const stopTypeLabels: Record<StopType, string> = {
  Pickup: "Pickup",
  Delivery: "Delivery",
  SplitPickup: "Split Pickup",
  SplitDelivery: "Split Delivery",
};

function MiniLocationDisplay({
  locationId,
  fallbackLabel,
}: {
  locationId: string;
  fallbackLabel?: string;
}) {
  const { data: location } = useQuery({
    queryKey: ["location", "selectOption", locationId],
    queryFn: () => apiService.locationService.getOption(locationId),
    enabled: !!locationId,
    staleTime: 5 * 60 * 1000,
  });

  if (!location) {
    return <span className="text-xs text-muted-foreground">{fallbackLabel ?? "Loading..."}</span>;
  }

  return (
    <div className="min-w-0">
      <p className="truncate text-xs font-medium">{location.name}</p>
      {location.addressLine1 && (
        <p className="truncate text-xs text-muted-foreground">{location.addressLine1}</p>
      )}
      <p className="truncate text-xs text-muted-foreground">
        {location.city}
        {location.state?.abbreviation && `, ${location.state.abbreviation}`} {location.postalCode}
      </p>
    </div>
  );
}

function MiniStopRow({
  locationId,
  stopType,
  time,
  showConnector,
  placeholder,
}: {
  locationId?: string;
  stopType: StopType;
  time?: { date: string; time: string } | null;
  showConnector?: boolean;
  placeholder?: string;
}) {
  return (
    <div className="relative flex items-start gap-3">
      <div className="flex flex-col items-center">
        <div className="flex size-5 shrink-0 items-center justify-center rounded-full bg-purple-500">
          <div className="size-2 rounded-full bg-white" />
        </div>
        {showConnector && <div className="mt-0.5 h-12 w-0.5 bg-purple-500" />}
      </div>

      <div className="flex min-w-0 flex-1 items-start justify-between gap-2 pb-1">
        <div className="min-w-0 flex-1">
          {locationId ? (
            <MiniLocationDisplay locationId={locationId} />
          ) : placeholder ? (
            <div className="flex h-10 items-center rounded border border-dashed px-2">
              <span className="text-xs text-muted-foreground">{placeholder}</span>
            </div>
          ) : null}
        </div>
        <div className="flex shrink-0 flex-col items-end gap-1">
          <Badge variant="secondary">{stopTypeLabels[stopType]}</Badge>
          {time ? (
            <span className="text-2xs text-muted-foreground">
              {time.date} {time.time}
            </span>
          ) : (
            <span className="text-2xs text-muted-foreground">--</span>
          )}
        </div>
      </div>
    </div>
  );
}

function CurrentMovePreview({ move }: { move: ShipmentMove }) {
  const user = useAuthStore((state) => state.user);
  const userTimezone = user?.timezone || "auto";
  const userTimeFormat = user?.timeFormat || "12-hour";
  const pickup = move.stops[0];
  const delivery = move.stops[1];
  if (!pickup || !delivery) return null;

  const pickupTime =
    pickup.scheduledWindowStart && pickup.scheduledWindowStart > 0
      ? formatSplitDateTime(pickup.scheduledWindowStart, userTimeFormat, userTimezone)
      : null;
  const deliveryTime =
    delivery.scheduledWindowStart && delivery.scheduledWindowStart > 0
      ? formatSplitDateTime(delivery.scheduledWindowStart, userTimeFormat, userTimezone)
      : null;

  return (
    <div>
      <h4 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
        Current Move
      </h4>
      <div className="rounded-lg border bg-muted/50 p-3">
        <MiniStopRow
          locationId={pickup.locationId}
          stopType={pickup.type}
          time={pickupTime}
          showConnector
        />
        <MiniStopRow
          locationId={delivery.locationId}
          stopType={delivery.type}
          time={deliveryTime}
        />
      </div>
    </div>
  );
}

function AfterSplitPreview({
  move,
  formValues,
}: {
  move: ShipmentMove;
  formValues: SplitMoveFormValues;
}) {
  const user = useAuthStore((state) => state.user);
  const userTimezone = user?.timezone || "auto";
  const userTimeFormat = user?.timeFormat || "12-hour";
  const pickup = move.stops[0];
  const delivery = move.stops[1];
  if (!pickup || !delivery) return null;

  const pickupTime =
    pickup.scheduledWindowStart && pickup.scheduledWindowStart > 0
      ? formatSplitDateTime(pickup.scheduledWindowStart, userTimeFormat, userTimezone)
      : null;

  const splitPickupTime =
    formValues.splitPickupScheduledWindowStart > 0
      ? formatSplitDateTime(formValues.splitPickupScheduledWindowStart, userTimeFormat, userTimezone)
      : null;

  const newDeliveryTime =
    formValues.newDeliveryScheduledWindowStart > 0
      ? formatSplitDateTime(formValues.newDeliveryScheduledWindowStart, userTimeFormat, userTimezone)
      : null;

  const hasAssignment = !!move.assignment?.id;

  return (
    <div>
      <h4 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
        After Split
      </h4>
      <div className="grid grid-cols-2 gap-3">
        <div className="rounded-lg border bg-muted/50 p-3">
          <div className="mb-2 flex items-center gap-2">
            <Badge variant="secondary">Original</Badge>
            {hasAssignment && (
              <span className="text-2xs text-muted-foreground">keeps assignment</span>
            )}
          </div>
          <MiniStopRow
            locationId={pickup.locationId}
            stopType="Pickup"
            time={pickupTime}
            showConnector
          />
          <MiniStopRow locationId={delivery.locationId} stopType="SplitDelivery" />
        </div>

        <div className="rounded-lg border bg-muted/50 p-3">
          <div className="mb-2 flex items-center gap-2">
            <Badge variant="info">New</Badge>
            <span className="text-2xs text-muted-foreground">unassigned</span>
          </div>
          <MiniStopRow
            locationId={delivery.locationId}
            stopType="SplitPickup"
            time={splitPickupTime}
            showConnector
          />
          <MiniStopRow
            locationId={formValues.newDeliveryLocationId || undefined}
            stopType="Delivery"
            time={newDeliveryTime}
            placeholder="Select a destination..."
          />
        </div>
      </div>
    </div>
  );
}

export function SplitMoveDialog({
  open,
  onOpenChange,
  moveId,
  currentMove,
  onSplit,
}: SplitMoveDialogProps) {
  const queryClient = useQueryClient();

  const origDelivery = currentMove.stops[1];
  const origDeliveryArrival = origDelivery?.scheduledWindowStart ?? 0;
  const origDeliveryDeparture = origDelivery?.scheduledWindowEnd ?? null;

  const form = useForm({
    defaultValues: {
      newDeliveryLocationId: "",
      splitPickupScheduledWindowStart: origDeliveryArrival,
      splitPickupScheduledWindowEnd: origDeliveryDeparture ?? origDeliveryArrival,
      newDeliveryScheduledWindowStart: 0,
      newDeliveryScheduledWindowEnd: null,
      pieces: null,
      weight: null,
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    setError,
    formState: { isSubmitting },
  } = form;

  const formValues = useWatch({ control }) as SplitMoveFormValues;

  const { mutateAsync } = useApiMutation<
    SplitMoveResponse,
    SplitMovePayload,
    unknown,
    SplitMoveFormValues
  >({
    mutationFn: (payload: SplitMovePayload) =>
      apiService.assignmentService.splitMove(moveId, payload),
    resourceName: "Split Move",
    setFormError: setError,
    onSuccess: (data: SplitMoveResponse) => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      onSplit(data);
      toast.success("Move split successfully");
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: SplitMoveFormValues) => {
      const payload: SplitMovePayload = {
        newDeliveryLocationId: values.newDeliveryLocationId,
        splitPickupTimes: {
          scheduledWindowStart: values.splitPickupScheduledWindowStart,
          scheduledWindowEnd: values.splitPickupScheduledWindowEnd,
        },
        newDeliveryTimes: {
          scheduledWindowStart: values.newDeliveryScheduledWindowStart,
          scheduledWindowEnd: values.newDeliveryScheduledWindowEnd,
        },
        ...(values.pieces != null && { pieces: values.pieces }),
        ...(values.weight != null && { weight: values.weight }),
      };
      await mutateAsync(payload);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  const hasAssignment = !!currentMove.assignment?.id;

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="gap-2 overflow-hidden p-0 sm:max-w-3xl">
        <DialogHeader className="gap-0 border-b border-border p-4">
          <DialogTitle>Split Move</DialogTitle>
          <DialogDescription>
            The original delivery becomes the handoff point. A new move continues from there to a
            new destination.
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <ScrollArea className="flex max-h-[calc(100vh-14rem)] flex-col px-4 [&_[data-slot=scroll-area-viewport]>div]:block!">
            <div className="space-y-5 px-1 pb-4">
              {hasAssignment && (
                <div className="flex shrink-0 items-start gap-2 rounded-lg border border-blue-200 bg-blue-50 p-3 dark:border-blue-900 dark:bg-blue-950/50">
                  <InfoIcon className="mt-0.5 size-4 shrink-0 text-blue-600 dark:text-blue-400" />
                  <p className="text-xs text-blue-700 dark:text-blue-300">
                    The current assignment will remain on the original move. The new move will be
                    unassigned.
                  </p>
                </div>
              )}
              <CurrentMovePreview move={currentMove} />

              <Separator />

              <div>
                <h4 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                  New Destination
                </h4>
                <FormGroup cols={1}>
                  <FormControl>
                    <LocationAutocompleteField
                      control={control}
                      name="newDeliveryLocationId"
                      rules={{ required: true }}
                      label="Delivery Location"
                      placeholder="Select destination for the new move"
                    />
                  </FormControl>
                </FormGroup>
              </div>
              <AfterSplitPreview move={currentMove} formValues={formValues} />
              <Separator />
              <Section
                label="Handoff Pickup Times"
                description="Pre-filled from the original delivery. The new carrier picks up at the same location, so these times should match or follow the original delivery."
              >
                <FormGroup cols={2}>
                  <FormControl>
                    <AutoCompleteDateTimeField
                      control={control}
                      name="splitPickupScheduledWindowStart"
                      rules={{ required: true }}
                      label="Scheduled Window Start"
                      placeholder="Scheduled window start"
                    />
                  </FormControl>
                  <FormControl>
                    <AutoCompleteDateTimeField
                      control={control}
                      name="splitPickupScheduledWindowEnd"
                      label="Scheduled Window End"
                      placeholder="Optional scheduled window end"
                    />
                  </FormControl>
                </FormGroup>
              </Section>
              <Section
                label="New Delivery Times"
                description="When the new move arrives at the final destination. These should be after the handoff pickup departure above."
              >
                <FormGroup cols={2}>
                  <FormControl>
                    <AutoCompleteDateTimeField
                      control={control}
                      name="newDeliveryScheduledWindowStart"
                      rules={{ required: true }}
                      label="Scheduled Window Start"
                      placeholder="Scheduled window start"
                    />
                  </FormControl>
                  <FormControl>
                    <AutoCompleteDateTimeField
                      control={control}
                      name="newDeliveryScheduledWindowEnd"
                      label="Scheduled Window End"
                      placeholder="Optional scheduled window end"
                    />
                  </FormControl>
                </FormGroup>
              </Section>
              <Section label="Cargo (Optional)">
                <FormGroup cols={2}>
                  <FormControl>
                    <NumberField
                      control={control}
                      name="pieces"
                      label="Pieces"
                      placeholder="Optional"
                    />
                  </FormControl>
                  <FormControl>
                    <NumberField
                      control={control}
                      name="weight"
                      label="Weight"
                      placeholder="Optional"
                    />
                  </FormControl>
                </FormGroup>
              </Section>
            </div>
          </ScrollArea>
          <DialogFooter className="m-0">
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText="Splitting...">
              Split Move
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

function Section({
  label,
  description,
  children,
}: {
  label: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <h4 className="text-xs font-semibold tracking-wide text-foreground uppercase">{label}</h4>
      {description && <p className="mb-2 text-2xs text-muted-foreground">{description}</p>}
      {children}
    </div>
  );
}
