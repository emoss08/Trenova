import {
  AccessorialChargeAutocompleteField,
  CustomerAutocompleteField,
  LocationAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { DetentionPolicy } from "@trenova/shared/types/detention";
import { useFormContext, useWatch } from "react-hook-form";
import { DetentionTierEditor } from "./detention-tier-editor";

const STATUS_OPTIONS = [
  { label: "Draft", value: "Draft" },
  { label: "Active", value: "Active" },
  { label: "Inactive", value: "Inactive" },
];

const CLOCK_START_OPTIONS = [
  {
    label: "Later of arrival or appointment",
    value: "LaterOfArrivalOrAppointment",
  },
  { label: "Actual arrival", value: "Arrival" },
  { label: "Appointment, regardless of arrival", value: "Appointment" },
  {
    label: "Earlier of arrival or appointment",
    value: "EarlierOfArrivalOrAppointment",
  },
];

const LATE_ARRIVAL_OPTIONS = [
  { label: "No effect on entitlement", value: "NoEffect" },
  { label: "Forfeit detention entirely", value: "Forfeit" },
  { label: "Anchor the clock to the appointment", value: "ClockFromAppointment" },
  { label: "Subtract lateness from free time", value: "ReduceFreeTime" },
];

const ROUNDING_OPTIONS = [
  { label: "Round up", value: "Up" },
  { label: "Round down", value: "Down" },
  { label: "Round to nearest", value: "Nearest" },
  { label: "Bill exact minutes", value: "Exact" },
];

const RATE_SOURCE_OPTIONS = [
  { label: "Flat accessorial rate", value: "Accessorial" },
  { label: "Graduated tiers", value: "Tiers" },
];

const NOTIFICATION_OPTIONS = [
  { label: "No notice required", value: "None" },
  { label: "Advisory only", value: "Advisory" },
  { label: "Required to bill", value: "Required" },
];

const UNNOTIFIED_OPTIONS = [
  { label: "Bill anyway", value: "Bill" },
  { label: "Hold for review", value: "Flag" },
  { label: "Suppress the charge", value: "Suppress" },
];

const CLOCK_START_HELP: Record<string, string> = {
  LaterOfArrivalOrAppointment:
    "The most common contract term: an early driver does not begin accruing until the appointment.",
  Arrival: "The clock starts the moment the driver arrives, even if early.",
  Appointment:
    "An early driver accrues free time while waiting for the appointment.",
  EarlierOfArrivalOrAppointment:
    "Favors the carrier when a facility accepts early delivery.",
};

const LATE_ARRIVAL_HELP: Record<string, string> = {
  NoEffect: "Lateness is recorded but the carrier keeps full entitlement.",
  Forfeit:
    "A late arrival voids detention entirely. Common in shipper-favorable rate confirmations — verify before selecting.",
  ClockFromAppointment:
    "Entitlement survives, but lateness burns the carrier's own free time.",
  ReduceFreeTime: "The minutes of lateness are subtracted from the allowance.",
};

function Section({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}

/**
 * The policy builder.
 *
 * Every field that changes what a customer is charged carries the plain-language
 * consequence of the choice, because the terms here are contract language and
 * picking the wrong one is invisible until an invoice is rejected.
 */
export function DetentionPolicyForm() {
  const { control } = useFormContext<DetentionPolicy>();

  const isOrgDefault = useWatch({ control, name: "isOrgDefault" });
  const rateSource = useWatch({ control, name: "rateSource" });
  const roundingMode = useWatch({ control, name: "roundingMode" });
  const clockStartBasis = useWatch({ control, name: "clockStartBasis" });
  const lateArrivalRule = useWatch({ control, name: "lateArrivalRule" });
  const notificationRequirement = useWatch({
    control,
    name: "notificationRequirement",
  });
  const convertToLayover = useWatch({ control, name: "convertToLayoverAtMinutes" });

  return (
    <div className="flex flex-col gap-4">
      <Section
        title="Identity"
        description="How this policy is referenced across billing and audit"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField control={control} name="name" label="Name" />
          </FormControl>
          <FormControl>
            <InputField control={control} name="code" label="Code" />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              options={STATUS_OPTIONS}
              description="Only Active policies participate in resolution"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="priority"
              label="Priority"
              description="Overrides computed specificity. Leave at 0 unless you need an explicit override."
            />
          </FormControl>
          <FormControl className="col-span-2">
            <TextareaField
              control={control}
              name="description"
              label="Description"
            />
          </FormControl>
        </FormGroup>
      </Section>

      <Section
        title="Scope"
        description="Which freight this policy governs. An empty dimension is a wildcard, and the most specific matching policy wins."
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="isOrgDefault"
              label="Organization default"
              description="The fallback used when no narrower policy matches. A default cannot target a customer, facility, or type."
              position="left"
            />
          </FormControl>
        </FormGroup>

        {!isOrgDefault && (
          <FormGroup cols={2} className="mt-3">
            <FormControl>
              <CustomerAutocompleteField
                control={control}
                name="customerId"
                label="Customer"
                description="Specificity +16"
              />
            </FormControl>
            <FormControl>
              <LocationAutocompleteField
                control={control}
                name="locationId"
                label="Facility"
                description="Specificity +32. Facility-level terms outrank customer-wide ones."
              />
            </FormControl>
            <FormControl className="col-span-2">
              <SwitchField
                control={control}
                name="appointmentStopsOnly"
                label="Appointment stops only"
                description="Open stops will not accrue detention under this policy."
                position="left"
              />
            </FormControl>
          </FormGroup>
        )}
      </Section>

      <Section
        title="The clock"
        description="When detention starts running, and what happens when the driver is late"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="clockStartBasis"
              label="Clock starts at"
              options={CLOCK_START_OPTIONS}
              description={CLOCK_START_HELP[clockStartBasis ?? ""]}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="lateArrivalRule"
              label="If the driver arrives late"
              options={LATE_ARRIVAL_OPTIONS}
              description={LATE_ARRIVAL_HELP[lateArrivalRule ?? ""]}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="lateArrivalGraceMinutes"
              label="Late arrival grace"
              sideText="min"
              description="Arrivals inside this buffer are not treated as late."
            />
          </FormControl>
        </FormGroup>

        {lateArrivalRule === "Forfeit" && (
          <div className="mt-3 rounded-md border border-amber-500/40 bg-amber-500/10 p-3">
            <Badge className="border-none bg-amber-500/20 text-amber-700 dark:text-amber-400">
              Check the contract
            </Badge>
            <p className="mt-2 text-sm">
              Forfeit voids detention entirely on a late arrival, even by one
              minute. Only select this when the rate confirmation says so.
            </p>
          </div>
        )}
      </Section>

      <Section
        title="Free time"
        description="What the contract grants before charges begin, on both the customer and driver sides"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="billingFreeMinutes"
              label="Free time"
              sideText="min"
              description="Industry standard is 120 minutes."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="payFreeMinutes"
              label="Driver pay free time"
              sideText="min"
              description="Leave empty to match the customer allowance. A longer customer allowance than this means you pay for time you cannot bill."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="pickupFreeMinutes"
              label="Pickup override"
              sideText="min"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="deliveryFreeMinutes"
              label="Delivery override"
              sideText="min"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="minimumBillableMinutes"
              label="Minimum billable"
              sideText="min"
              description="Detention shorter than this bills nothing."
            />
          </FormControl>
        </FormGroup>
      </Section>

      <Section
        title="Rounding"
        description="How raw minutes collapse onto the billing increment. Rounding errors are a leading cause of rejected detention invoices."
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="roundingMode"
              label="Rounding"
              options={ROUNDING_OPTIONS}
            />
          </FormControl>
          {roundingMode !== "Exact" && (
            <FormControl>
              <NumberField
                control={control}
                name="billingIncrementMinutes"
                label="Increment"
                sideText="min"
                description="Typically 15, 30, or 60."
              />
            </FormControl>
          )}
        </FormGroup>
      </Section>

      <Section
        title="Rate"
        description="What the customer pays once free time is exhausted"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="rateSource"
              label="Rate source"
              options={RATE_SOURCE_OPTIONS}
            />
          </FormControl>
          <FormControl>
            <AccessorialChargeAutocompleteField
              control={control}
              name="accessorialChargeId"
              label="Accessorial charge"
              description="The billing code detention posts against."
            />
          </FormControl>
        </FormGroup>

        {rateSource === "Tiers" && (
          <div className="mt-4">
            <DetentionTierEditor />
          </div>
        )}
      </Section>

      <Section
        title="Ceilings"
        description="Caps the contract places on a single stop, day, or shipment"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="maxBillableMinutesPerStop"
              label="Max billable minutes"
              sideText="min"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerStop"
              label="Max charge per stop"
              sideText="$"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerDay"
              label="Max charge per day"
              sideText="$"
              description="Allocated across calendar days, so a stay spanning midnight is not charged two full daily maximums."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerShipment"
              label="Max charge per shipment"
              sideText="$"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="convertToLayoverAtMinutes"
              label="Convert to layover at"
              sideText="min"
              description="Detention stops accruing here and layover takes over. Industry convention is 1440 (24h)."
            />
          </FormControl>
          {convertToLayover ? (
            <FormControl>
              <AccessorialChargeAutocompleteField
                control={control}
                name="layoverAccessorialChargeId"
                label="Layover charge"
              />
            </FormControl>
          ) : null}
        </FormGroup>
      </Section>

      <Section
        title="Customer notice"
        description="Contracts that pay detention reliably require written notice at or before free-time expiry. Missing it is the most common reason a valid claim goes uncollected."
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="notificationRequirement"
              label="Notice requirement"
              options={NOTIFICATION_OPTIONS}
            />
          </FormControl>
          {notificationRequirement !== "None" && (
            <>
              <FormControl>
                <SelectField
                  control={control}
                  name="unnotifiedBehavior"
                  label="If the notice is missed"
                  options={UNNOTIFIED_OPTIONS}
                  description={
                    notificationRequirement === "Required"
                      ? "A required notice must Flag or Suppress; it cannot bill regardless."
                      : undefined
                  }
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="notificationLeadMinutes"
                  label="Warn before expiry"
                  sideText="min"
                  description="Fires while dispatch can still get the truck unloaded."
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="notificationDeadlineMinutes"
                  label="Grace after expiry"
                  sideText="min"
                  description="How long after free time a notice still satisfies the contract."
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="autoSendNotice"
                  label="Send the notice automatically"
                  position="left"
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="sendDepartureSummary"
                  label="Send a summary on departure"
                  description="Customers rarely dispute a number they were told twice while it was happening."
                  position="left"
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </Section>

      <Section
        title="Approval"
        description="Which charges clear automatically and which need a human"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="autoApproveUnderAmount"
              label="Auto-approve under"
              sideText="$"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="requireApprovalOverAmount"
              label="Require approval over"
              sideText="$"
            />
          </FormControl>
          <FormControl className="col-span-2">
            <TextareaField
              control={control}
              name="comments"
              label="Contract notes"
              description="Reference the clause this policy encodes."
            />
          </FormControl>
        </FormGroup>
      </Section>
    </div>
  );
}
