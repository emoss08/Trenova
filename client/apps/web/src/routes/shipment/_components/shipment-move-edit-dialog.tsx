import { useT } from "@trenova/shared/i18n/use-t";
import { LocationAutocompleteField } from "@/components/autocomplete-fields";
import { CheckboxField } from "@/components/fields/checkbox-field";
import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import {
  moveStatusChoices,
  stopHandlingChoices,
  stopScheduleTypeChoices,
  stopStatusChoices,
  stopTypeChoices,
} from "@/lib/choices";
import type { Shipment, StopScheduleType, StopStatus } from "@trenova/shared/types/shipment";
import {
  AlertTriangleIcon,
  CalendarIcon,
  MarkerPin01Icon,
  PlusIcon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { useRef, useState } from "react";
import {
  type FieldPath,
  useFieldArray,
  useFormContext,
  useFormState,
  useWatch,
} from "react-hook-form";
import { type MoveProblem, moveProblems, problemText } from "./move/move-problems";

export type MoveDialogState =
  | { open: false }
  | {
      open: true;
      moveIndex: number;
      isNew: boolean;
      /** The form path of the field to open on, such as a stop's location or a field to fix. */
      focus?: string;
    };

// The fields the editor draws, each showing its own error. A problem on any other field (or on
// the move or a stop as a whole) is said in a list, since no field is there to show it.
const MOVE_FIELDS_DRAWN = new Set(["status", "distance", "loaded"]);
const STOP_FIELDS_DRAWN = new Set([
  "locationId",
  "type",
  "pieces",
  "weight",
  "scheduleType",
  "handling",
  "scheduledWindowStart",
  "scheduledWindowEnd",
  "actualArrival",
  "actualDeparture",
]);

/** The problems no field of the editor shows, said in a list of their own. */
function UndrawnProblems({ problems }: { problems: readonly MoveProblem[] }) {
  const t = useT();
  if (problems.length === 0) return null;
  return (
    <Alert variant="destructive" size="sm">
      <AlertTriangleIcon />
      <AlertDescription>
        <ul className="space-y-0.5">
          {problems.map((problem) => (
            <li key={`${problem.path}:${problem.message}`}>{problemText(problem, t)}</li>
          ))}
        </ul>
      </AlertDescription>
    </Alert>
  );
}

const stopStatusBadgeVariant: Record<StopStatus, "info" | "success" | "danger"> = {
  New: "info",
  InTransit: "info",
  Completed: "success",
  Canceled: "danger",
};

/**
 * Edits one move and its stops. It opens on the field it is asked to (a stop's location, or the
 * first field to fix), and it closes only once the move has nothing left to fix, taking the
 * person to the first problem when it does.
 */
export function MoveEditDialog({
  state,
  onClose,
  onCancel,
}: {
  state: MoveDialogState;
  onClose: () => void;
  onCancel: () => void;
}) {
  if (!state.open) return null;

  return (
    <MoveEditor
      moveIndex={state.moveIndex}
      isNew={state.isNew}
      focus={state.focus}
      onClose={onClose}
      onCancel={onCancel}
    />
  );
}

function MoveEditor({
  moveIndex,
  isNew,
  focus,
  onClose,
  onCancel,
}: {
  moveIndex: number;
  isNew: boolean;
  focus: string | undefined;
  onClose: () => void;
  onCancel: () => void;
}) {
  const t = useT();

  const { control, setFocus, trigger, getFieldState } = useFormContext<Shipment>();
  const { errors } = useFormState({ control, name: `moves.${moveIndex}` });
  const problems = moveProblems(moveIndex, errors.moves?.[moveIndex]);
  const undrawnMoveProblems = problems.filter(
    (problem) => problem.stopIndex === null && !MOVE_FIELDS_DRAWN.has(problem.field),
  );
  const contentRef = useRef<HTMLDivElement>(null);
  const [checking, setChecking] = useState(false);

  // Brings the field at a path into view and focus. The stop (or the move's own fields) it is on
  // takes focus first, so a problem on a field the editor does not draw still lands where it is
  // said; the field then takes focus from it when it can.
  function reveal(path: string): HTMLElement | null {
    const stopIndex = /^moves\.\d+\.stops\.(\d+)/.exec(path)?.[1];
    const region = contentRef.current?.querySelector<HTMLElement>(
      stopIndex === undefined ? "[data-move-fields]" : `[data-stop-index="${stopIndex}"]`,
    );
    region?.focus();
    setFocus(path as FieldPath<Shipment>);
    const active = document.activeElement;
    return active instanceof HTMLElement && contentRef.current?.contains(active) ? active : null;
  }

  async function handleSave() {
    setChecking(true);
    try {
      if (await trigger(`moves.${moveIndex}`)) {
        onClose();
        return;
      }
    } finally {
      setChecking(false);
    }
    const first = moveProblems(moveIndex, getFieldState(`moves.${moveIndex}`).error)[0];
    if (first) reveal(first.path);
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onCancel();
      }}
    >
      <DialogContent
        size="lg"
        className="gap-0 overflow-hidden p-0"
        initialFocus={focus ? () => reveal(focus) ?? true : undefined}
      >
        <DialogHeader className="border-border border-b px-4 py-3">
          <DialogTitle>{isNew ? t("Add move") : t("Edit move {0}", moveIndex + 1)}</DialogTitle>
          <DialogDescription>
            {isNew
              ? t("Define the move legs and stop sequence for this shipment.")
              : t("Update move details, timing, and stop sequence.")}
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className="max-h-[65vh]">
          <div ref={contentRef} className="space-y-4 p-4">
            <UndrawnProblems problems={undrawnMoveProblems} />
            <div data-move-fields="" tabIndex={-1} className="outline-none">
              <FormGroup cols={3} dense>
                <FormControl>
                  <SelectField
                    control={control}
                    name={`moves.${moveIndex}.status`}
                    rules={{ required: true }}
                    label={t("Status")}
                    description={t(
                      "Tracks where this move is in its lifecycle, from new through completion",
                    )}
                    isReadOnly
                    options={moveStatusChoices}
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name={`moves.${moveIndex}.distance`}
                    label={t("Distance (mi)")}
                    description={t(
                      "Total miles for this leg, used for rate calculations and driver pay",
                    )}
                    placeholder="0"
                  />
                </FormControl>
                <FormControl>
                  <CheckboxField
                    control={control}
                    name={`moves.${moveIndex}.loaded`}
                    label={t("Loaded")}
                    description={t(
                      "Indicates whether this move is carrying freight or is an empty repositioning leg",
                    )}
                  />
                </FormControl>
              </FormGroup>
            </div>

            <StopsList moveIndex={moveIndex} problems={problems} />
          </div>
        </ScrollArea>
        <DialogFooter className="m-0">
          <Button type="button" variant="outline" onClick={onCancel}>
            {t("Cancel")}
          </Button>
          <Button type="button" isLoading={checking} onClick={() => void handleSave()}>
            {isNew ? t("Add move") : t("Save changes")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function StopsList({
  moveIndex,
  problems,
}: {
  moveIndex: number;
  problems: readonly MoveProblem[];
}) {
  const t = useT();

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
      handling: "Live",
      locationId: "",
      sequence,
      scheduledWindowStart: 0,
      scheduledWindowEnd: null,
    });

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <span className="text-muted-foreground text-xs font-medium">
          {t("Stops ({0})", fields.length)}
        </span>
        <Button type="button" variant="outline" size="xxs" onClick={() => addStop(fields.length)}>
          <PlusIcon className="size-3" />
          {t("Add stop")}
        </Button>
      </div>

      <div className="space-y-2">
        {fields.map((field, stopIndex) => (
          <StopCard
            key={field.fieldId}
            moveIndex={moveIndex}
            stopIndex={stopIndex}
            totalStops={fields.length}
            problems={problems.filter((problem) => problem.stopIndex === stopIndex)}
            onRemove={() => remove(stopIndex)}
          />
        ))}

        {fields.length === 0 && (
          <div className="flex flex-col items-center justify-center rounded-md border border-dashed py-8 text-center">
            <MarkerPin01Icon className="text-muted-foreground/40 mb-2 size-4" />
            <p className="text-muted-foreground text-xs">{t("No stops configured")}</p>
            <p className="text-muted-foreground/60 mb-3 text-xs">
              {t("Add at least one pickup and delivery stop.")}
            </p>
            <Button type="button" variant="outline" size="xxs" onClick={() => addStop(0)}>
              <PlusIcon className="size-3" />
              {t("Add first stop")}
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
  problems,
  onRemove,
}: {
  moveIndex: number;
  stopIndex: number;
  totalStops: number;
  /** What keeps this stop from saving. */
  problems: readonly MoveProblem[];
  onRemove: () => void;
}) {
  const t = useT();

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
        ? t("Appt. window start")
        : t("Appt. time")
      : hasWindowEnd
        ? t("Scheduled start")
        : t("Scheduled time");
  const endLabel = scheduleType === "Appointment" ? t("Appt. window end") : t("Scheduled end");
  const startPlaceholder =
    scheduleType === "Appointment"
      ? hasWindowEnd
        ? t("Start time")
        : t("Appointment time")
      : hasWindowEnd
        ? t("Start time")
        : t("Scheduled time");
  const endPlaceholder = t("End time (optional)");
  const undrawn = problems.filter((problem) => !STOP_FIELDS_DRAWN.has(problem.field));

  return (
    <div
      data-stop-index={stopIndex}
      tabIndex={-1}
      className={cn("rounded-md border outline-none", problems.length > 0 && "border-danger")}
    >
      <div className="flex items-center justify-between border-b px-3 py-1.5">
        <div className="flex items-center gap-2">
          <MarkerPin01Icon className="text-muted-foreground size-3" />
          <span className="text-xs font-medium">
            {t("Stop {0}", stopIndex + 1)}
            <span className="text-muted-foreground"> / {totalStops}</span>
          </span>
          <Badge variant={stopStatusBadgeVariant[status]} className="text-2xs h-5">
            {statusLabel}
          </Badge>
          {problems.length > 0 && (
            <span className="text-danger text-2xs flex items-center gap-1 font-medium">
              <AlertTriangleIcon aria-hidden className="size-3" />
              {t("{0, plural, one {# problem to fix} other {# problems to fix}}", problems.length)}
            </span>
          )}
        </div>
        <Button type="button" variant="ghost" size="icon" className="size-6" onClick={onRemove}>
          <XCloseIcon className="text-muted-foreground size-3" />
        </Button>
      </div>

      <div className="space-y-3 p-3">
        <UndrawnProblems problems={undrawn} />
        <FormGroup cols={2} dense>
          <FormControl>
            <LocationAutocompleteField
              control={control}
              name={`moves.${moveIndex}.stops.${stopIndex}.locationId`}
              rules={{ required: true }}
              label={t("Location")}
              description={t("The facility, warehouse, or yard where the driver will stop")}
              placeholder={t("Search locations...")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name={`moves.${moveIndex}.stops.${stopIndex}.type`}
              rules={{ required: true }}
              label={t("Stop type")}
              description={t(
                "Defines the purpose of the stop — pickup, delivery, or a split operation",
              )}
              options={stopTypeChoices}
            />
          </FormControl>
        </FormGroup>

        <FormGroup cols={2} dense>
          <FormControl>
            <NumberField
              control={control}
              name={`moves.${moveIndex}.stops.${stopIndex}.pieces`}
              label={t("Pieces")}
              description={t(
                "Count of individual freight units (pallets, crates, etc.) handled at this stop",
              )}
              placeholder="0"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`moves.${moveIndex}.stops.${stopIndex}.weight`}
              label={t("Weight (lbs)")}
              description={t(
                "Combined weight of all freight being loaded or unloaded at this stop",
              )}
              placeholder="0"
            />
          </FormControl>
        </FormGroup>

        <div>
          <div className="mb-1.5 flex items-center gap-1">
            <CalendarIcon className="text-muted-foreground size-3" />
            <span className="text-2xs text-muted-foreground font-medium">{t("Scheduling")}</span>
          </div>
          <div className="space-y-1">
            <FormGroup cols={2} dense>
              <FormControl>
                <SelectField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.scheduleType`}
                  rules={{ required: true }}
                  label={t("Schedule type")}
                  description={t(
                    "How this stop is scheduled — an open arrival window or a fixed appointment time",
                  )}
                  options={stopScheduleTypeChoices}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.handling`}
                  rules={{ required: true }}
                  label={t("Live or drop")}
                  description={t(
                    "Live holds the truck while the facility loads or unloads; a drop leaves or takes a trailer",
                  )}
                  options={stopHandlingChoices}
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
                  description={t("The earliest time the driver is expected to arrive at this stop")}
                  placeholder={startPlaceholder}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.scheduledWindowEnd`}
                  label={endLabel}
                  description={t(
                    "The latest acceptable arrival time, leave blank for exact appointments",
                  )}
                  placeholder={endPlaceholder}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.actualArrival`}
                  label={t("Actual arrival")}
                  description={t("Recorded time the driver checked in at the facility")}
                  placeholder={t("Arrival time")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name={`moves.${moveIndex}.stops.${stopIndex}.actualDeparture`}
                  label={t("Actual departure")}
                  description={t(
                    "Recorded time the driver left the facility after loading or unloading",
                  )}
                  placeholder={t("Departure time")}
                />
              </FormControl>
            </FormGroup>
          </div>
        </div>
      </div>
    </div>
  );
}
