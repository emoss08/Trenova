import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

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
        title={t("Identity")}
        description={t("How this policy is referenced across billing and audit")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label={t("Name")}
              placeholder={t("Standard Dry Van Detention")}
              rules={{ required: true }}
              maxLength={100}
              description={t("Human-friendly name shown in lists, notices, and audit history.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label={t("Code")}
              placeholder={t("DET-STD")}
              rules={{ required: true }}
              maxLength={50}
              description={t("Short identifier quoted on notices and invoices.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              placeholder={t("Select status")}
              rules={{ required: true }}
              options={detentionPolicyStatusChoices}
              description={t("Only Active policies participate in resolution; Draft lets you build and backtest safely.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="priority"
              label={t("Priority")}
              placeholder="0"
              description={t("Overrides computed specificity when two policies tie. Leave at 0 unless you need an explicit override.")}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("Detention terms for standard dry van freight, per the 2026 master agreement")}
              description={t("Context for the next person who has to understand why these terms exist.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Scope")}
        description={t("Which freight this policy governs. An empty dimension is a wildcard, and the most specific matching policy wins: facility beats customer beats commodity beats type.")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="isOrgDefault"
              label={t("Organization default")}
              description={t("The fallback used when no narrower policy matches. A default cannot target a customer, facility, or type.")}
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
                  label={t("Customer")}
                  placeholder={t("Any customer")}
                  description={t("Limit this policy to one customer's freight. Specificity +16.")}
                  clearable
                />
              </FormControl>
              <FormControl>
                <LocationAutocompleteField
                  control={control}
                  name="locationId"
                  label={t("Facility")}
                  placeholder={t("Any facility")}
                  description={t("Limit to one facility. Specificity +32 — facility-level terms outrank customer-wide ones.")}
                  clearable
                />
              </FormControl>
              <FormControl>
                <ShipmentTypeMultiSelectField
                  control={control}
                  name="shipmentTypeIds"
                  label={t("Shipment Types")}
                  placeholder={t("Any shipment type")}
                  description={t("Only shipments of these types are governed. Specificity +4.")}
                />
              </FormControl>
              <FormControl>
                <ServiceTypeMultiSelectField
                  control={control}
                  name="serviceTypeIds"
                  label={t("Service Types")}
                  placeholder={t("Any service type")}
                  description={t("Only these service levels are governed. Specificity +2.")}
                />
              </FormControl>
              <FormControl cols="full">
                <CommodityMultiSelectField
                  control={control}
                  name="commodityIds"
                  label={t("Commodities")}
                  placeholder={t("Any commodity")}
                  description={t("Only shipments carrying these commodities are governed. Specificity +8.")}
                />
              </FormControl>
              <FormControl cols="full">
                <MultiCheckboxField
                  control={control}
                  name="stopTypes"
                  label={t("Stop Types")}
                  options={stopTypeChoices}
                  description={t("Leave all unchecked to govern every stop type. Specificity +1.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField
                  control={control}
                  name="effectiveStartDate"
                  label={t("Effective From")}
                  placeholder={t("No start date")}
                  description={t("The policy only governs stops arriving on or after this date.")}
                  clearable
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField
                  control={control}
                  name="effectiveEndDate"
                  label={t("Expires")}
                  placeholder={t("No expiration")}
                  description={t("The policy stops governing after this date — set it when a contract term ends.")}
                  clearable
                />
              </FormControl>
            </>
          )}

          <FormControl cols="full">
            <SwitchField
              control={control}
              name="appointmentStopsOnly"
              label={t("Appointment stops only")}
              description={t("Open (first-come) stops will not accrue detention under this policy.")}
              outlined
              position="left"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("The clock")}
        description={t("When detention starts running, and what happens when the driver is late")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="clockStartBasis"
              label={t("Clock starts at")}
              placeholder={t("Select clock basis")}
              rules={{ required: true }}
              options={detentionClockStartBasisChoices}
              description={CLOCK_START_HELP[clockStartBasis ?? ""]}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="lateArrivalRule"
              label={t("If the driver arrives late")}
              placeholder={t("Select late arrival rule")}
              rules={{ required: true }}
              options={detentionLateArrivalRuleChoices}
              description={LATE_ARRIVAL_HELP[lateArrivalRule ?? ""]}
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="lateArrivalGraceMinutes"
              label={t("Late arrival grace")}
              placeholder="0"
              sideText="min"
              description={t("Arrivals inside this buffer are not treated as late.")}
            />
          </FormControl>
        </FormGroup>

        {lateArrivalRule === "Forfeit" && (
          <div className="mt-3 rounded-md border border-amber-500/40 bg-amber-500/10 p-3">
            <Badge className="border-none bg-amber-500/20 text-amber-700 dark:text-amber-400">
              {t("Check the contract")}
            </Badge>
            <p className="mt-2 text-sm">
              {t("Forfeit voids detention entirely on a late arrival, even by one minute. Only select this when the rate confirmation says so.")}
            </p>
          </div>
        )}
      </FormSection>

      <FormSection
        title={t("Free time")}
        description={t("What the contract grants before charges begin, on both the customer and driver sides")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="billingFreeMinutes"
              label={t("Free time")}
              placeholder="120"
              sideText="min"
              rules={{ required: true }}
              description={t("Minutes the customer gets before charges begin. Industry standard is 120.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="payFreeMinutes"
              label={t("Driver pay free time")}
              placeholder={t("Match customer allowance")}
              sideText="min"
              description={t("Leave empty to match the customer allowance. A longer customer allowance than this means you pay for time you cannot bill.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="pickupFreeMinutes"
              label={t("Pickup override")}
              placeholder={t("Use base free time")}
              sideText="min"
              description={t("Different allowance for pickup stops, when the contract splits them.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="deliveryFreeMinutes"
              label={t("Delivery override")}
              placeholder={t("Use base free time")}
              sideText="min"
              description={t("Different allowance for delivery stops, when the contract splits them.")}
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="minimumBillableMinutes"
              label={t("Minimum billable")}
              placeholder="0"
              sideText="min"
              description={t("Detention shorter than this bills nothing — filters out trivially small charges.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Rounding")}
        description={t("How raw minutes collapse onto the billing increment. Rounding errors are a leading cause of rejected detention invoices.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="roundingMode"
              label={t("Rounding")}
              placeholder={t("Select rounding mode")}
              rules={{ required: true }}
              options={detentionRoundingModeChoices}
              description={t("How partial increments are billed once free time is exhausted.")}
            />
          </FormControl>
          {roundingMode !== "Exact" && (
            <FormControl>
              <NumberField
                control={control}
                name="billingIncrementMinutes"
                label={t("Increment")}
                placeholder="15"
                sideText="min"
                rules={{ required: true }}
                description={t("The billing unit minutes are rounded onto. Typically 15, 30, or 60.")}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>

      <FormSection title={t("Rate")} description={t("What the customer pays once free time is exhausted")}>
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="rateSource"
              label={t("Rate source")}
              placeholder={t("Select rate source")}
              rules={{ required: true }}
              options={detentionRateSourceChoices}
              description={t("A flat hourly accessorial rate, or a graduated ladder that escalates with dwell.")}
            />
          </FormControl>
          <FormControl>
            <AccessorialChargeAutocompleteField
              control={control}
              name="accessorialChargeId"
              label={t("Accessorial charge")}
              placeholder={t("Select accessorial charge")}
              rules={{ required: true }}
              description={t("The billing code detention posts against. Its rate applies when the source is Flat.")}
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
        title={t("Ceilings")}
        description={t("Caps the contract places on a single stop, day, or shipment")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="maxBillableMinutesPerStop"
              label={t("Max billable minutes")}
              placeholder={t("No cap")}
              sideText="min"
              description={t("Billable minutes per stop stop accruing here, regardless of dwell.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerStop"
              label={t("Max charge per stop")}
              placeholder={t("No cap")}
              sideText="$"
              description={t("Dollar ceiling for a single stop's detention.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerDay"
              label={t("Max charge per day")}
              placeholder={t("No cap")}
              sideText="$"
              description={t("Allocated across calendar days, so a stay spanning midnight is not charged two full daily maximums.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxChargePerShipment"
              label={t("Max charge per shipment")}
              placeholder={t("No cap")}
              sideText="$"
              description={t("Total detention across every stop on the shipment stops accruing here.")}
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="convertToLayoverAtMinutes"
              label={t("Convert to layover at")}
              placeholder={t("Never convert")}
              sideText="min"
              description={t("Detention stops accruing here and layover takes over. Industry convention is 1440 (24h).")}
            />
          </FormControl>
          {convertToLayover ? (
            <FormControl>
              <AccessorialChargeAutocompleteField
                control={control}
                name="layoverAccessorialChargeId"
                label={t("Layover charge")}
                placeholder={t("Select layover charge")}
                rules={{ required: true }}
                description={t("The billing code the stay converts to once the layover boundary is crossed.")}
              />
            </FormControl>
          ) : null}
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Customer notice")}
        description={t("Contracts that pay detention reliably require written notice at or before free-time expiry. Missing it is the most common reason a valid claim goes uncollected.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="notificationRequirement"
              label={t("Notice requirement")}
              placeholder={t("Select requirement")}
              rules={{ required: true }}
              options={detentionNotificationRequirementChoices}
              description={t("Whether the contract obligates a written notice before detention can bill.")}
            />
          </FormControl>
          {notificationRequirement !== "None" && (
            <>
              <FormControl>
                <SelectField
                  control={control}
                  name="unnotifiedBehavior"
                  label={t("If the notice is missed")}
                  placeholder={t("Select behavior")}
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
                  label={t("Warn before expiry")}
                  placeholder="30"
                  sideText="min"
                  description={t("Fires while dispatch can still get the truck unloaded.")}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="notificationDeadlineMinutes"
                  label={t("Grace after expiry")}
                  placeholder="0"
                  sideText="min"
                  description={t("How long after free time a notice still satisfies the contract.")}
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="autoSendNotice"
                  label={t("Send the notice automatically")}
                  description={t("The sweep emails the customer the moment the notice window opens, using their configured recipients.")}
                  outlined
                  position="left"
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="attachNoticePdf"
                  label={t("Attach the notice as a PDF")}
                  description={t("Emails a printable copy alongside the message and files it against the shipment, which is what a customer's claims desk asks for in a dispute.")}
                  outlined
                  position="left"
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name="sendDepartureSummary"
                  label={t("Send a summary on departure")}
                  description={t("Customers rarely dispute a number they were told twice while it was happening.")}
                  outlined
                  position="left"
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Approval")}
        description={t("Which charges clear automatically and which need a human")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="autoApproveUnderAmount"
              label={t("Auto-approve under")}
              placeholder={t("Never auto-approve")}
              sideText="$"
              description={t("Charges below this amount post without review.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="requireApprovalOverAmount"
              label={t("Require approval over")}
              placeholder={t("Never require")}
              sideText="$"
              description={t("Charges above this amount always wait for a human.")}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="comments"
              label={t("Contract notes")}
              placeholder={t("Section 4.2 of the 2026 master agreement — detention after 2 free hours at $75/hr")}
              description={t("Reference the clause this policy encodes, so a dispute can cite it.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
