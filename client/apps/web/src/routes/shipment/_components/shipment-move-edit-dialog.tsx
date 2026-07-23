import { LocationAutocompleteField } from "@/components/autocomplete-fields";
import { CheckboxField } from "@/components/fields/checkbox-field";
import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
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
import { FormControl, FormGroup } from "@/components/ui/form";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  moveStatusChoices,
  stopScheduleTypeChoices,
  stopStatusChoices,
  stopTypeChoices,
} from "@/lib/choices";
import type { Shipment, StopScheduleType, StopStatus } from "@/types/shipment";
import { CalendarIcon, MapPinIcon, PlusIcon, XIcon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

type MoveDialogState = { open: false } | { open: true; moveIndex: number; isNew: boolean };

const stopStatusBadgeVariant: Record<StopStatus, "info" | "teal" | "active" | "inactive"> = {
  New: "info",
  InTransit: "teal",
  Completed: "active",
  Canceled: "inactive",
};

export function MoveEditDialog({
  state,
  onClose,
  onCancel,
}: {
  state: MoveDialogState;
  onClose: () => void;
  onCancel: () => void;
}) {
  const { control } = useFormContext<Shipment>();

  if (!state.open) return null;

  const { moveIndex, isNew } = state;

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onCancel();
      }}
    >
      <DialogContent className="gap-0 overflow-hidden p-0 sm:max-w-2xl">
        <DialogHeader className="gap-0 p-4">
          <DialogTitle>{isNew ? "Add Move" : `Edit Move ${moveIndex + 1}`}</DialogTitle>
          <DialogDescription>
            {isNew
              ? "Define the move legs and stop sequence for this shipment."
              : "Update move details, timing, and stop sequence."}
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className="max-h-[65vh] px-4 pb-4">
          <div className="space-y-4">
            <FormGroup cols={3} dense>
              <FormControl>
                <SelectField
                  control={control}
                  name={`moves.${moveIndex}.status`}
                  rules={{ required: true }}
                  label="Status"
                  description="Tracks where this move is in its lifecycle, from new through completion"
                  isReadOnly
                  options={moveStatusChoices}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name={`moves.${moveIndex}.distance`}
                  label="Distance (mi)"
                  description="Total miles for this leg, used for rate calculations and driver pay"
                  placeholder="0"
                />
              </FormControl>
              <FormControl>
                <CheckboxField
                  control={control}
                  name={`moves.${moveIndex}.loaded`}
                  label="Loaded"
                  description="Indicates whether this move is carrying freight or is an empty repositioning leg"
                />
              </FormControl>
            </FormGroup>

            <StopsList moveIndex={moveIndex} />
          </div>
        </ScrollArea>
        <DialogFooter className="m-0">
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="button" onClick={onClose}>
            {isNew ? "Add Move" : "Save Changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function StopsList({ moveIndex }: { moveIndex: number }) {
  const { control } = useFormContext<Shipment>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: `moves.${moveIndex}.stops`,
    keyName: "fieldId",
  });

  const addStop = (sequence: number) =>
    append({
      status: "New",
      type: "Pickup",
      scheduleType: "Open",
      locationId: "",
      sequence,
      scheduledWindowStart: 0,
      scheduledWindowEnd: null,
    });

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <span className="text-xs font-medium text-muted-foreground">Stops ({fields.length})</span>
        <Button type="button" variant="outline" size="xxs" onClick={() => addStop(fields.length)}>
          <PlusIcon className="size-3" />
          Add Stop
        </Button>
      </div>

      <div className="space-y-2">
        {fields.map((field, stopIndex) => (
          <StopCard
            key={field.fieldId}
            moveIndex={moveIndex}
            stopIndex={stopIndex}
            totalStops={fields.length}
            onRemove={() => remove(stopIndex)}
          />
        ))}

        {fields.length === 0 && (
          <div className="flex flex-col items-center justify-center rounded-md border border-dashed py-8 text-center">
            <MapPinIcon className="mb-2 size-4 text-muted-foreground/40" />
            <p className="text-xs text-muted-foreground">No stops configured</p>
            <p className="mb-3 text-xs text-muted-foreground/60">
              Add at least one pickup and delivery stop.
            </p>
            <Button type="button" variant="outline" size="xxs" onClick={() => addStop(0)}>
              <PlusIcon className="size-3" />
              Add First Stop
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

function StopCard({
  moveIndex,
  stopIndex,
  totalStops,
  onRemove,
}: {
  moveIndex: number;
  stopIndex: number;
  totalStops: number;
  onRemove: () => void;
}) {
  const { control } = useFormContext<Shipment>();
  const status = useWatch({
    control,
    name: `moves.${moveIndex}.stops.${stopIndex}.status`,
  }) as StopStatus;
  const scheduleType = useWatch({
    control,
    name: `moves.${moveIndex}.stops.${stopIndex}.scheduleType`,
  }) as StopScheduleType;
  const scheduledWindowEnd = useWatch({
    control,
    name: `moves.${moveIndex}.stops.${stopIndex}.scheduledWindowEnd`,
  }) as number | null | undefined;

  const hasWindowEnd = !!scheduledWindowEnd && scheduledWindowEnd > 0;

  const statusLabel = stopStatusChoices.find((c) => c.value === status)?.label ?? status;

  const startLabel =
    scheduleType === "Appointment"
      ? hasWindowEnd
        ? "Appt. Window Start"
        : "Appt. Time"
      : hasWindowEnd
        ? "Scheduled Start"
        : "Scheduled Time";
  const endLabel = scheduleType === "Appointment" ? "Appt. Window End" : "Scheduled End";
  const startPlaceholder =
    scheduleType === "Appointment"
      ? hasWindowEnd
        ? "Start time"
        : "Appointment time"
      : hasWindowEnd
        ? "Start time"
        : "Scheduled time";
  const endPlaceholder = "End time (optional)";

  return (
    <div className="rounded-md border">
      <div className="flex items-center justify-between border-b px-3 py-1.5">
        <div className="flex items-center gap-2">
          <MapPinIcon className="size-3 text-muted-foreground" />
          <span className="text-xs font-medium">
            Stop {stopIndex + 1}
            <span className="text-muted-foreground"> / {totalStops}</span>
          </span>
          <Badge variant={stopStatusBadgeVariant[status]} className="h-5 text-2xs">
            {statusLabel}
          </Badge>
        </div>
        <Button type="button" variant="ghost" size="icon" className="size-6" onClick={onRemove}>
          <XIcon className="size-3 text-muted-foreground" />
        </Button>
      </div>

      <div className="space-y-3 p-3">
        <FormGroup cols={2} dense>
          <FormControl>
            <LocationAutocompleteField
              control={control}
              name={`moves.${moveIndex}.stops.${stopIndex}.locationId`}
              rules={{ required: true }}
              label="Location"
              description="The facility, warehouse, or yard where the driver will stop"
              placeholder="Search locations..."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name={`moves.${moveIndex}.stops.${stopIndex}.type`}
              rules={{ required: true }}
              label="Stop Type"
              description="Defines the purpose of the stop — pickup, delivery, or a split operation"
              options={stopTypeChoices}
            />
          </FormControl>
        </FormGroup>

        <FormGroup cols={2} dense>
          <FormControl>
            <NumberField
              control={control}
              name={`moves.${moveIndex}.stops.${stopIndex}.pieces`}
              label="Pieces"
              description="Count of individual freight units (pallets, crates, etc.) handled at this stop"
              placeholder="0"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`moves.${moveIndex}.stops.${stopIndex}.weight`}
              label="Weight (lbs)"
              description="Combined weight of all freight being loaded or unloaded at this stop"
              placeholder="0"
            />
          </FormControl>
        </FormGroup>

        <div>
          <div className="mb-1.5 flex items-center gap-1">
            <CalendarIcon className="size-3 text-muted-foreground" />
            <span className="text-2xs font-medium text-muted-foreground">Scheduling</span>
          </div>
          <div className="space-y-1">
            <FormGroup cols={1} dense>
              <FormControl>
                <SelectField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.scheduleType`}
                  rules={{ required: true }}
                  label="Schedule Type"
                  description="How this stop is scheduled — an open arrival window or a fixed appointment time"
                  options={stopScheduleTypeChoices}
                />
              </FormControl>
            </FormGroup>
            <FormGroup cols={2} dense>
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.scheduledWindowStart`}
                  rules={{ required: true }}
                  label={startLabel}
                  description="The earliest time the driver is expected to arrive at this stop"
                  placeholder={startPlaceholder}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.scheduledWindowEnd`}
                  label={endLabel}
                  description="The latest acceptable arrival time, leave blank for exact appointments"
                  placeholder={endPlaceholder}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.actualArrival`}
                  label="Actual Arrival"
                  description="Recorded time the driver checked in at the facility"
                  placeholder="Arrival time"
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.actualDeparture`}
                  label="Actual Departure"
                  description="Recorded time the driver left the facility after loading or unloading"
                  placeholder="Departure time"
                />
              </FormControl>
            </FormGroup>
          </div>
        </div>
      </div>
    </div>
  );
}
