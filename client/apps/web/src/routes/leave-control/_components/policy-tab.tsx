import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { useT } from "@trenova/shared/i18n/use-t";
import { sourceLabels } from "@trenova/shared/i18n/labels";
import { MEASUREMENT_METHOD_HINTS, MEASUREMENT_METHOD_LABELS } from "@trenova/shared/lib/leave";
import {
  LEAVE_CONTROL_DECIMAL_SCALE,
  leaveMeasurementMethodSchema,
  type LeaveControl,
  type LeaveMeasurementMethod,
} from "@trenova/shared/types/worker-leave";
import { useFormContext, useWatch } from "react-hook-form";

const METHOD_OPTIONS = leaveMeasurementMethodSchema.options.map((value) => ({
  value,
  label: sourceLabels(MEASUREMENT_METHOD_LABELS)[value],
  description: sourceLabels(MEASUREMENT_METHOD_HINTS)[value],
}));

/**
 * The methods that let an employee take one year's leave at the end of a year and the next
 * year's at the start of the next, twenty-four weeks in a row (29 CFR 825.200(b)).
 */
const STACKING_METHODS: ReadonlySet<LeaveMeasurementMethod> = new Set([
  leaveMeasurementMethodSchema.enum.CalendarYear,
  leaveMeasurementMethodSchema.enum.HireAnniversary,
]);

/** The leave policy every employee is held to: the year, the entitlement and who qualifies. */
export function PolicyTab() {
  const t = useT();
  const { control } = useFormContext<LeaveControl>();
  const method = useWatch({ control, name: "measurementMethod" });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Leave year")}
        description={t(
          "The twelve months each employee's leave is counted in, the same for everyone.",
        )}
      >
        <SelectField
          control={control}
          name="measurementMethod"
          label={t("Measured as")}
          description={t(
            "Changing it is a change of policy that employees get sixty days' notice of.",
          )}
          options={METHOD_OPTIONS}
          placeholder={t("Choose a method")}
          warning={{
            show: STACKING_METHODS.has(method),
            message: t(
              "An employee can take two years' leave back to back across the turn of the year.",
            ),
          }}
          required
        />
      </SettingsSection>

      <SettingsSection
        title={t("Entitlement")}
        description={t("How much leave an eligible employee gets in each leave year.")}
      >
        <NumberField
          control={control}
          name="entitlementWeeks"
          valueType="string"
          label={t("Leave")}
          description={t("At least twelve weeks under the FMLA; more is allowed.")}
          placeholder="12"
          sideText={t("weeks")}
          decimalScale={LEAVE_CONTROL_DECIMAL_SCALE}
          min={12}
          max={52}
          required
        />
        <NumberField
          control={control}
          name="militaryCaregiverWeeks"
          valueType="string"
          label={t("Military caregiver leave")}
          description={t(
            "Leave to care for a seriously injured service member; at least twenty-six weeks.",
          )}
          placeholder="26"
          sideText={t("weeks")}
          decimalScale={LEAVE_CONTROL_DECIMAL_SCALE}
          min={26}
          max={52}
          required
        />
        <NumberField
          control={control}
          name="workweekHours"
          valueType="string"
          label={t("Workweek")}
          description={t(
            "What a week of leave is worth when leave is taken a few hours at a time.",
          )}
          placeholder="40"
          sideText={t("hours")}
          decimalScale={LEAVE_CONTROL_DECIMAL_SCALE}
          min={0}
          max={168}
          required
        />
      </SettingsSection>

      <SettingsSection
        title={t("Eligibility")}
        description={t("What an employee must have worked before they qualify for leave.")}
      >
        <NumberField
          control={control}
          name="eligibilityMonths"
          label={t("Service")}
          description={t("Counted from the hire date; twelve months under the FMLA.")}
          placeholder="12"
          sideText={t("months")}
          min={0}
          max={120}
          required
        />
        <NumberField
          control={control}
          name="eligibilityHours"
          label={t("Hours in the prior year")}
          description={t("Checked on each case by hand; 1,250 hours under the FMLA.")}
          placeholder="1250"
          sideText={t("hours")}
          min={0}
          max={8760}
          required
        />
      </SettingsSection>

      <SettingsSection
        title={t("Certification")}
        description={t(
          "How long an employee has to return a medical certification once it is requested.",
        )}
      >
        <NumberField
          control={control}
          name="certificationDueDays"
          label={t("Time to return it")}
          description={t("At least fifteen calendar days under the FMLA.")}
          placeholder="15"
          sideText={t("days")}
          min={15}
          required
        />
      </SettingsSection>
    </SettingsSections>
  );
}
