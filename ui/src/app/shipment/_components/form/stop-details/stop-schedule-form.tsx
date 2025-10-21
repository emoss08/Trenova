import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useFormContext } from "react-hook-form";
import { AutoCompleteDateTimeField } from "@/components/fields/datetime-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";

export function StopScheduleInformationForm({
  moveIdx,
  stopIdx,
}: {
  moveIdx: number;
  stopIdx: number;
}) {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <FormSection
      title="Schedule Information"
      description="Manage planned and actual arrival/departure times for this stop."
      className="mt-2 gap-1"
    >
      <div className="rounded-lg bg-muted p-4">
        <h4 className="text-sm font-medium text-foreground mb-3">
          Planned Times
        </h4>
        <FormGroup cols={2} className="gap-4">
          <FormControl>
            <AutoCompleteDateTimeField
              name={`moves.${moveIdx}.stops.${stopIdx}.plannedArrival`}
              control={control}
              rules={{ required: true }}
              label="Planned Arrival"
              placeholder="Select planned arrival"
              description="Indicates the scheduled arrival time for this stop."
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateTimeField
              name={`moves.${moveIdx}.stops.${stopIdx}.plannedDeparture`}
              control={control}
              rules={{ required: true }}
              label="Planned Departure"
              placeholder="Select planned departure"
              description="Indicates the scheduled departure time from this stop."
            />
          </FormControl>
        </FormGroup>
      </div>
      <div className="rounded-lg bg-muted p-4">
        <h4 className="text-sm font-medium text-foreground mb-3">
          Actual Times
        </h4>
        <FormGroup cols={2} className="gap-4">
          <FormControl>
            <AutoCompleteDateTimeField
              name={`moves.${moveIdx}.stops.${stopIdx}.actualArrival`}
              control={control}
              label="Actual Arrival"
              placeholder="Select actual arrival"
              description="Records the actual arrival time at this stop."
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateTimeField
              name={`moves.${moveIdx}.stops.${stopIdx}.actualDeparture`}
              control={control}
              label="Actual Departure"
              placeholder="Select actual departure"
              description="Records the actual departure time from this stop."
            />
          </FormControl>
        </FormGroup>
      </div>
    </FormSection>
  );
}
