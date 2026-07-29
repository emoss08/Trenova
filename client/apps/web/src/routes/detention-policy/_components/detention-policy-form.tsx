import {
  AccessorialChargeAutocompleteField,
  CommodityMultiSelectField,
  CustomerAutocompleteField,
  LocationAutocompleteField,
  ServiceTypeMultiSelectField,
  ShipmentTypeMultiSelectField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  detentionClockStartBasisChoices,
  detentionLateArrivalRuleChoices,
  detentionNotificationRequirementChoices,
  detentionPolicyStatusChoices,
  detentionRateSourceChoices,
  detentionRoundingModeChoices,
  detentionUnnotifiedBehaviorChoices,
  stopTypeChoices,
} from "@/lib/choices";
import { Badge } from "@trenova/shared/components/ui/badge";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { DetentionPolicy } from "@trenova/shared/types/detention";
import { useFormContext, useWatch } from "react-hook-form";
import { DetentionTierEditor } from "./detention-tier-editor";

const CLOCK_START_HELP: Record<string, string> = {
  LaterOfArrivalOrAppointment:
    "The most common contract term: an early driver does not begin accruing until the appointment.",
  Arrival: "The clock starts the moment the driver arrives, even if early.",
  Appointment: "An early driver accrues free time while waiting for the appointment.",
  EarlierOfArrivalOrAppointment: "Favors the carrier when a facility accepts early delivery.",
};

const LATE_ARRIVAL_HELP: Record<string, string> = {
  NoEffect: "Lateness is recorded but the carrier keeps full entitlement.",
  Forfeit:
    "A late arrival voids detention entirely. Common in shipper-favorable rate confirmations — verify before selecting.",
  ClockFromAppointment: "Entitlement survives, but lateness burns the carrier's own free time.",
  ReduceFreeTime: "The minutes of lateness are subtracted from the allowance.",
};

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
  const notificationRequirement = useWatch({ control, name: "notificationRequirement" });
  const convertToLayover = useWatch({ control, name: "convertToLayoverAtMinutes" });

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title="Identity"
        description="How this policy is referenced across billing and audit"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Name"
              placeholder="Standard Dry Van Detention"
              rules={{ required: true }}
              maxLength={100}
              description="Human-friendly name shown in lists, notices, and audit history."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label="Code"
              placeholder="DET-STD"
              rules={{ required: true }}
              maxLength={50}
              description="Short identifier quoted on notices and invoices."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              placeholder="Select status"
              rules={{ required: true }}
              options={detentionPolicyStatusChoices}
              description="Only Active policies participate in resolution; Draft lets you build and backtest safely."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="priority"
              label="Priority"
              placeholder="0"
              description="Overrides computed specificity when two policies tie. Leave at 0 unless you need an explicit override."
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label="Description"
              placeholder="Detention terms for standard dry van freight, per the 2026 master agreement"
              description="Context for the next person who has to understand why these terms exist."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Scope"
        description="Which freight this policy governs. An empty dimension is a wildcard, and the most specific matching policy wins: facility beats customer beats commodity beats type."
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="isOrgDefault"
              label="Organization default"
              description="The fallback used when no narrower policy matches. A default cannot target a customer, facility, or type."
              outlined
              position="left"
            />
          </FormControl>

          {!isOrgDefault && (
            <>
              <FormControl>
                <CustomerAutocompleteField
                  control={control}
                  name="customerId"
                  label="Customer"
                  placeholder="Any customer"
                  description="Limit this policy to one customer's freight. Specificity +16."
                  clearable
                />
              </FormControl>
              <FormControl>
                <LocationAutocompleteField
                  control={control}
                  name="locationId"
                  label="Facility"
                  placeholder="Any facility"
                  description="Limit to one facility. Specificity +32 — facility-level terms outrank customer-wide ones."
                  clearable
                />
              </FormControl>
              <FormControl>
                <ShipmentTypeMultiSelectField
                  control={control}
                  name="shipmentTypeIds"
                  label="Shipment Types"
                  placeholder="Any shipment type"
                  description="Only shipments of these types are governed. Specificity +4."
                />
              </FormControl>
              <FormControl>
                <ServiceTypeMultiSelectField
                  control={control}
                  name="serviceTypeIds"
                  label="Service Types"
                  placeholder="Any service type"
                  description="Only these service levels are governed. Specificity +2."
                />
              </FormControl>
              <FormControl cols="full">
                <CommodityMultiSelectField
                  control={control}
                  name="commodityIds"
                  label="Commodities"
                  placeholder="Any commodity"
                  description="Only shipments carrying these commodities are governed. Specificity +8."
                />
              </FormControl>
              <FormControl cols="full">
                <MultiCheckboxField
                  control={control}
                  name="stopTypes"
                  label="Stop Types"
                  options={stopTypeChoices}
                  description="Leave all unchecked to govern every stop type. Specificity +1."
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField
                  control={control}
                  name="effectiveStartDate"
                  label="Effective From"
                  placeholder="No start date"
                  description="The policy only governs stops arriving on or after this date."
                  clearable
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField
                  control={control}
                  name="effectiveEndDate"
                  label="Expires"
                  placeholder="No expiration"
                  description="The policy stops governing after this date — set it when a contract term ends."
                  clearable
                />
              </FormControl>
            </>
          )}

          <FormControl cols="full">
            <SwitchField
              control={control}
              name="appointmentStopsOnly"
              label="Appointment stops only"
              description="Open (first-come) stops will not accrue detention under this policy."
              outlined
              position="left"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="The clock"
        description="When detention starts running, and what happens when the driver is late"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="clockStartBasis"
              label="Clock starts at"
              placeholder="Select clock basis"
              rules={{ required: true }}
              options={detentionClockStartBasisChoices}
              description={CLOCK_START_HELP[clockStartBasis ?? ""]}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="lateArrivalRule"
              label="If the driver arrives late"
              placeholder="Select late arrival rule"
              rules={{ required: true }}
              options={detentionLateArrivalRuleChoices}
              description={LATE_ARRIVAL_HELP[lateArrivalRule ?? ""]}
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="lateArrivalGraceMinutes"
              label="Late arrival grace"
              placeholder="0"
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
              Forfeit voids detention entirely on a late arrival, even by one minute. Only select
              this when the rate confirmation says so.
            </p>
          </div>
        )}
      </FormSection>

      <FormSection
        title="Free time"
        description="What the contract grants before charges begin, on both the customer and driver sides"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="billingFreeMinutes"
              label="Free time"
              placeholder="120"
              sideText="min"
              rules={{ required: true }}
              description="Minutes the customer gets before charges begin. Industry standard is 120."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="payFreeMinutes"
              label="Driver pay free time"
              placeholder="Match customer allowance"
              sideText="min"
              description="Leave empty to match the customer allowance. A longer customer allowance than this means you pay for time you cannot bill."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="pickupFreeMinutes"
              label="Pickup override"
              placeholder="Use base free time"
              sideText="min"
              description="Different allowance for pickup stops, when the contract splits them."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="deliveryFreeMinutes"
              label="Delivery override"
              placeholder="Use base free time"
              sideText="min"
              description="Different allowance for delivery stops, when the contract splits them."
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="minimumBillableMinutes"
              label="Minimum billable"
              placeholder="0"
              sideText="min"
              description="Detention shorter than this bills nothing — filters out trivially small charges."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Rounding"
        description="How raw minutes collapse onto the billing increment. Rounding errors are a leading cause of rejected detention invoices."
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="roundingMode"
              label="Rounding"
              placeholder="Select rounding mode"
              rules={{ required: true }}
              options={detentionRoundingModeChoices}
              description="How partial increments are billed once free time is exhausted."
            />
          </FormControl>
          {roundingMode !== "Exact" && (
            <FormControl>
              <NumberField
                control={control}
                name="billingIncrementMinutes"
                label="Increment"
                placeholder="15"
                sideText="min"
                rules={{ required: true }}
                description="The billing unit minutes are rounded onto. Typically 15, 30, or 60."
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>

      <FormSection title="Rate" description="What the customer pays once free time is exhausted">
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="rateSource"
              label="Rate source"
              placeholder="Select rate source"
              rules={{ required: true }}
              options={detentionRateSourceChoices}
              description="A flat hourly accessorial rate, or a graduated ladder that escalates with dwell."
            />
          </FormControl>
          <FormControl>
            <AccessorialChargeAutocompleteField
              control={control}
              name="accessorialChargeId"
              label="Accessorial charge"
              placeholder="Select accessorial charge"
              rules={{ required: true }}
              description="The billing code detention posts against. Its rate applies when the source is Flat."
            />
          </FormControl>
        </FormGroup>

        {rateSource === "Tiers" && (
          <div className="mt-4">
            <DetentionTierEditor />
          </div>
        )}
      </FormSection>

      <FormSection
        title="Ceilings"
        description="Caps the contract places on a single stop, day, or shipment"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="maxBillableMinutesPerStop"
              label="Max billable minutes"
              placeholder="No cap"
              sideText="min"
              description="Billable minutes per stop stop accruing here, regardless of dwell."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerStop"
              label="Max charge per stop"
              placeholder="No cap"
              sideText="$"
              description="Dollar ceiling for a single stop's detention."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerDay"
              label="Max charge per day"
              placeholder="No cap"
              sideText="$"
              description="Allocated across calendar days, so a stay spanning midnight is not charged two full daily maximums."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerShipment"
              label="Max charge per shipment"
              placeholder="No cap"
              sideText="$"
              description="Total detention across every stop on the shipment stops accruing here."
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="convertToLayoverAtMinutes"
              label="Convert to layover at"
              placeholder="Never convert"
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
                placeholder="Select layover charge"
                rules={{ required: true }}
                description="The billing code the stay converts to once the layover boundary is crossed."
              />
            </FormControl>
          ) : null}
        </FormGroup>
      </FormSection>

      <FormSection
        title="Customer notice"
        description="Contracts that pay detention reliably require written notice at or before free-time expiry. Missing it is the most common reason a valid claim goes uncollected."
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="notificationRequirement"
              label="Notice requirement"
              placeholder="Select requirement"
              rules={{ required: true }}
              options={detentionNotificationRequirementChoices}
              description="Whether the contract obligates a written notice before detention can bill."
            />
          </FormControl>
          {notificationRequirement !== "None" && (
            <>
              <FormControl>
                <SelectField
                  control={control}
                  name="unnotifiedBehavior"
                  label="If the notice is missed"
                  placeholder="Select behavior"
                  rules={{ required: true }}
                  options={detentionUnnotifiedBehaviorChoices}
                  description={
                    notificationRequirement === "Required"
                      ? "A required notice must Flag or Suppress; it cannot bill regardless."
                      : "What happens to the charge when the notice never went out."
                  }
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="notificationLeadMinutes"
                  label="Warn before expiry"
                  placeholder="30"
                  sideText="min"
                  description="Fires while dispatch can still get the truck unloaded."
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="notificationDeadlineMinutes"
                  label="Grace after expiry"
                  placeholder="0"
                  sideText="min"
                  description="How long after free time a notice still satisfies the contract."
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="autoSendNotice"
                  label="Send the notice automatically"
                  description="The sweep emails the customer the moment the notice window opens, using their configured recipients."
                  outlined
                  position="left"
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="sendDepartureSummary"
                  label="Send a summary on departure"
                  description="Customers rarely dispute a number they were told twice while it was happening."
                  outlined
                  position="left"
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </FormSection>

      <FormSection
        title="Approval"
        description="Which charges clear automatically and which need a human"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="autoApproveUnderAmount"
              label="Auto-approve under"
              placeholder="Never auto-approve"
              sideText="$"
              description="Charges below this amount post without review."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="requireApprovalOverAmount"
              label="Require approval over"
              placeholder="Never require"
              sideText="$"
              description="Charges above this amount always wait for a human."
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="comments"
              label="Contract notes"
              placeholder="Section 4.2 of the 2026 master agreement — detention after 2 free hours at $75/hr"
              description="Reference the clause this policy encodes, so a dispute can cite it."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
