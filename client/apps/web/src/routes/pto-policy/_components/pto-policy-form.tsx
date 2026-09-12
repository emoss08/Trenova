import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { ptoTypeChoices } from "@/lib/choices";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  PTO_ACCRUAL_METHOD_LABELS,
  PTO_TERMINATION_ACTION_LABELS,
  PTO_YEAR_BASIS_LABELS,
  type PTOAccrualMethod,
  type PTOPolicyFormValues,
  type PTOTerminationAction,
  type PTOYearBasis,
} from "@trenova/shared/types/pto-policy";
import { InfoIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

const POLICY_STATUS_OPTIONS = [
  { value: "Active", label: "Active", color: "#15803d" },
  { value: "Inactive", label: "Inactive", color: "#b91c1c" },
  { value: "Draft", label: "Draft", color: "#6b7280" },
];

const YEAR_BASIS_OPTIONS = (Object.keys(PTO_YEAR_BASIS_LABELS) as PTOYearBasis[]).map((value) => ({
  value,
  label: PTO_YEAR_BASIS_LABELS[value],
}));

const ACCRUAL_METHOD_OPTIONS = (Object.keys(PTO_ACCRUAL_METHOD_LABELS) as PTOAccrualMethod[]).map(
  (value) => ({ value, label: PTO_ACCRUAL_METHOD_LABELS[value] }),
);

const TERMINATION_OPTIONS = (
  Object.keys(PTO_TERMINATION_ACTION_LABELS) as PTOTerminationAction[]
).map((value) => ({ value, label: PTO_TERMINATION_ACTION_LABELS[value] }));

function amountLabel(method: PTOAccrualMethod): string {
  switch (method) {
    case "Monthly":
      return "Days per month";
    case "FixedAnnualGrant":
      return "Days per year";
    case "PerPayPeriod":
      return "Days per pay period";
    case "None":
      return "Accrual amount";
    default:
      return "Accrual amount";
  }
}

export function PTOPolicyForm({
  isEdit,
  openAssignmentCount = 0,
}: {
  isEdit: boolean;
  openAssignmentCount?: number;
}) {
  const t = useT();

  const { control } = useFormContext<PTOPolicyFormValues>();
  const rulesArray = useFieldArray({ control, name: "rules" });
  const allowNegative = useWatch({ control, name: "allowNegative" });

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("General")}
          hint={t("Name and code identify the policy on worker records and reports.")}
        />
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label={t("Code")}
              placeholder={t("e.g. STD-DRIVER")}
              rules={{ required: true }}
              description={t("Short unique identifier. Uppercase letters, digits, and dashes.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label={t("Name")}
              placeholder={t("e.g. Standard Driver")}
              rules={{ required: true }}
              description={t("Shown on worker records, assignments and PTO reports.")}
            />
          </FormControl>
          <FormControl cols="full">
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              options={POLICY_STATUS_OPTIONS}
              rules={{ required: true }}
              placeholder={t("Select a status")}
              description={
                isEdit && openAssignmentCount > 0
                  ? `${openAssignmentCount} worker${openAssignmentCount === 1 ? " is" : "s are"} assigned; reassign them before deactivating.`
                  : "Only active policies can be assigned to workers."
              }
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="isDefault"
              label={t("Default for new hires")}
              description={t("New workers are enrolled in this policy from their hire date.")}
              position="left"
              outlined
            />
          </FormControl>
          <FormControl className="col-span-2">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("Who this policy covers and anything unusual about it")}
              maxLength={1000}
              description={t("Optional notes for whoever assigns or maintains the policy.")}
            />
          </FormControl>
        </FormGroup>
      </section>

      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("Year & counting")}
          hint={t("Controls when the policy year rolls over and how request days are counted.")}
        />
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="yearBasis"
              label={t("Policy year")}
              options={YEAR_BASIS_OPTIONS}
              rules={{ required: true }}
              placeholder={t("Select a year basis")}
              description={t(
                "Carryover caps, expiries, and annual grants apply at the start of this year.",
              )}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="waitingPeriodDays"
              label={t("Waiting period")}
              sideText="days"
              min={0}
              placeholder="90"
              description={t(
                "Days after hire before any accrual starts. Skipped periods are not back-filled.",
              )}
            />
          </FormControl>
          <FormControl className="col-span-2">
            <SwitchField
              control={control}
              name="countWeekends"
              label={t("Count weekends")}
              description={t(
                "On for drivers who work seven-day schedules. Off counts only Monday–Friday against a request and skips observed holidays.",
              )}
              position="left"
              outlined
            />
          </FormControl>
        </FormGroup>
      </section>

      <section className="flex flex-col gap-3">
        <div className="flex items-start justify-between gap-2">
          <SectionTitle
            title={t("Accrual rules")}
            hint={t(
              "One rule per PTO type. Types without a rule are still requestable but are not tracked against a balance.",
            )}
          />
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() =>
              rulesArray.append({
                ptoType: "Sick",
                accrualMethod: "FixedAnnualGrant",
                accrualAmountDays: "5",
                maxBalanceDays: null,
                carryoverCapDays: null,
                carryoverExpiryDays: 0,
                tiers: [],
                onTermination: "Forfeit",
              })
            }
          >
            <PlusIcon className="size-3.5" />
            {t("Add type")}
          </Button>
        </div>
        <Alert variant="default">
          <InfoIcon className="size-4" />
          <AlertTitle>{t("Rule changes apply going forward")}</AlertTitle>
          <AlertDescription>
            {t(
              "Accruals are posted once per period. Changing an amount, cap or tier affects periods that have not been posted yet; days already in a worker's ledger are not recalculated.",
            )}
          </AlertDescription>
        </Alert>
        <div className="flex flex-col gap-3">
          {rulesArray.fields.map((field, index) => (
            <RuleRow
              key={field.id}
              index={index}
              onRemove={rulesArray.fields.length > 1 ? () => rulesArray.remove(index) : undefined}
            />
          ))}
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("Enforcement")}
          hint={t(
            "Whether requests are blocked when a balance runs out, and whether dispatch must approve them.",
          )}
        />
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="requiresApproval"
              label={t("Requires approval")}
              description={t("Off auto-approves requests and books the days immediately.")}
              position="left"
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enforceBalance"
              label={t("Enforce balance")}
              description={t("Off keeps balances informational — requests are never blocked.")}
              position="left"
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowNegative"
              label={t("Allow negative balance")}
              description={t("Lets a request go below zero down to the floor set here.")}
              position="left"
              outlined
            />
          </FormControl>
          {allowNegative ? (
            <FormControl>
              <InputField
                control={control}
                name="negativeFloorDays"
                label={t("Negative floor")}
                placeholder="-3"
                description={t("Lowest balance a request may leave, e.g. -3.")}
              />
            </FormControl>
          ) : null}
        </FormGroup>
      </section>
    </div>
  );
}

function RuleRow({ index, onRemove }: { index: number; onRemove?: () => void }) {
  const t = useT();

  const { control } = useFormContext<PTOPolicyFormValues>();
  const method = useWatch({ control, name: `rules.${index}.accrualMethod` });
  const tiersArray = useFieldArray({ control, name: `rules.${index}.tiers` });
  const accrues = method !== "None";

  return (
    <div className="bg-muted/30 rounded-lg border p-3">
      <div className="mb-2 flex items-center justify-between">
        <p className="text-muted-foreground text-[11px] font-medium uppercase">
          {t("Rule {0}", index + 1)}
        </p>
        {onRemove ? (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="size-7"
            aria-label={`Remove rule ${index + 1}`}
            onClick={onRemove}
          >
            <Trash2Icon className="size-3.5" />
          </Button>
        ) : null}
      </div>
      <FormGroup cols={3}>
        <FormControl>
          <SelectField
            control={control}
            name={`rules.${index}.ptoType`}
            label={t("PTO type")}
            options={ptoTypeChoices}
            rules={{ required: true }}
            placeholder={t("Select a PTO type")}
            description={t("The kind of time off this rule accrues and tracks a balance for.")}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name={`rules.${index}.accrualMethod`}
            label={t("Accrual")}
            options={ACCRUAL_METHOD_OPTIONS}
            rules={{ required: true }}
            placeholder={t("Select an accrual method")}
            description={
              method === "PerPayPeriod"
                ? "Follows the settlement pay period calendar."
                : "How often days are added to the balance."
            }
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={`rules.${index}.accrualAmountDays`}
            label={amountLabel(method)}
            placeholder="0.83"
            disabled={!accrues}
            description={t("Days a worker earns per period; balances are kept in days.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={`rules.${index}.maxBalanceDays`}
            label={t("Max balance")}
            placeholder={t("Unlimited")}
            description={t("Accruals stop at this balance.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={`rules.${index}.carryoverCapDays`}
            label={t("Carryover cap")}
            placeholder={t("Unlimited")}
            description={t("Days kept at year start. 0 = use it or lose it.")}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name={`rules.${index}.carryoverExpiryDays`}
            label={t("Carryover expires")}
            sideText="days"
            min={0}
            placeholder="90"
            description={t("Carried days expire this many days into the new year. 0 = never.")}
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name={`rules.${index}.onTermination`}
            label={t("When employment ends")}
            options={TERMINATION_OPTIONS}
            rules={{ required: true }}
            placeholder={t("Select an action")}
            description={t(
              "Paid-out balances count toward the PTO liability report; forfeited ones are written off on the termination date.",
            )}
          />
        </FormControl>
      </FormGroup>

      <div className="mt-3 border-t pt-3">
        <div className="flex items-start justify-between gap-2">
          <div>
            <p className="text-xs font-medium">{t("Tenure tiers")}</p>
            <p className="text-muted-foreground text-[11px]">
              {accrues
                ? t(
                    "Raise the accrual once a worker has served long enough. The highest tier they qualify for wins.",
                  )
                : t("Tiers need an accruing rule.")}
            </p>
          </div>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            disabled={!accrues}
            onClick={() => {
              const last = tiersArray.fields[tiersArray.fields.length - 1];
              tiersArray.append({
                minMonths: last ? last.minMonths + 12 : 12,
                accrualAmountDays: "",
                maxBalanceDays: null,
              });
            }}
          >
            <PlusIcon className="size-3.5" />
            {t("Add tier")}
          </Button>
        </div>
        {tiersArray.fields.length > 0 ? (
          <div className="mt-2 flex flex-col gap-2">
            <p className="text-muted-foreground text-[11px]">
              {t(
                "After: months of service before the tier applies. Amount: replaces the base accrual from then on. Max balance: overrides the rule's cap; leave empty to inherit it.",
              )}
            </p>
            {tiersArray.fields.map((tier, tierIndex) => (
              <div
                key={tier.id}
                className="bg-background/60 grid grid-cols-[1fr_1fr_1fr_auto] items-end gap-2 rounded-md border px-2 py-2"
              >
                <NumberField
                  control={control}
                  name={`rules.${index}.tiers.${tierIndex}.minMonths`}
                  label={t("After")}
                  sideText="months"
                  min={1}
                  placeholder="12"
                />
                <InputField
                  control={control}
                  name={`rules.${index}.tiers.${tierIndex}.accrualAmountDays`}
                  label={amountLabel(method)}
                  placeholder="1.25"
                />
                <InputField
                  control={control}
                  name={`rules.${index}.tiers.${tierIndex}.maxBalanceDays`}
                  label={t("Max balance")}
                  placeholder={t("Inherit")}
                />
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  className="mb-0.5 size-7"
                  aria-label={`Remove tier ${tierIndex + 1} from rule ${index + 1}`}
                  onClick={() => tiersArray.remove(tierIndex)}
                >
                  <Trash2Icon className="size-3.5" />
                </Button>
              </div>
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function SectionTitle({ title, hint }: { title: string; hint: string }) {
  return (
    <div>
      <h3 className="text-sm font-semibold">{title}</h3>
      <p className="text-muted-foreground text-xs">{hint}</p>
    </div>
  );
}
