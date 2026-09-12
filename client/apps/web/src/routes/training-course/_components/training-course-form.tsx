import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { driverTypeChoices, statusChoices } from "@/lib/choices";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  TRAINING_CATEGORY_LABELS,
  TRAINING_DELIVERY_HINTS,
  TRAINING_DELIVERY_LABELS,
  TRAINING_DELIVERY_SELF_SERVE,
  trainingCategorySchema,
  trainingDeliverySchema,
  type TrainingCourseFormValues,
} from "@trenova/shared/types/worker-training";
import { useFormContext, useWatch } from "react-hook-form";

const CATEGORY_OPTIONS = trainingCategorySchema.options.map((value) => ({
  value,
  label: TRAINING_CATEGORY_LABELS[value],
}));

const DELIVERY_OPTIONS = trainingDeliverySchema.options.map((value) => ({
  value,
  label: TRAINING_DELIVERY_LABELS[value],
}));

type TrainingCourseFormProps = {
  isEdit: boolean;
  openRecordCount?: number;
};

export function TrainingCourseForm({ isEdit, openRecordCount = 0 }: TrainingCourseFormProps) {
  const t = useT();

  const { control } = useFormContext<TrainingCourseFormValues>();
  const [isRequired, delivery, passingScore] = useWatch({
    control,
    name: ["isRequired", "delivery", "passingScore"],
  });
  const selfServe = TRAINING_DELIVERY_SELF_SERVE.has(delivery);

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("General")}
          hint={t("Name and code identify the course on worker records and in Dash.")}
        />
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label={t("Code")}
              placeholder={t("e.g. DEFENSIVE")}
              rules={{ required: true }}
              description={t("Short unique identifier. Letters, digits, dashes and underscores.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label={t("Name")}
              placeholder={t("e.g. Defensive Driving")}
              rules={{ required: true }}
              description={t("Shown on the worker's training record and in Dash.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="category"
              label={t("Category")}
              options={CATEGORY_OPTIONS}
              rules={{ required: true }}
              placeholder={t("Select a category")}
              description={t("Groups the course on worker records and in the course list.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              options={statusChoices}
              rules={{ required: true }}
              placeholder={t("Select a status")}
              description={
                isEdit && openRecordCount > 0
                  ? `${openRecordCount} worker${openRecordCount === 1 ? " has" : "s have"} this course open; finish or cancel those first to deactivate.`
                  : "Inactive courses drop out of the required matrix and cannot be assigned."
              }
            />
          </FormControl>
          <FormControl className="col-span-2">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("What the course covers and the regulation behind it, if any")}
              maxLength={1000}
              description={t("Optional notes on the course content and why it is required.")}
            />
          </FormControl>
        </FormGroup>
      </section>

      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("Delivery")}
          hint={t("How a worker takes the course and what it takes to pass.")}
        />
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="delivery"
              label={t("Delivery")}
              options={DELIVERY_OPTIONS}
              rules={{ required: true }}
              placeholder={t("Select a delivery method")}
              description={TRAINING_DELIVERY_HINTS[delivery]}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="durationMinutes"
              label={t("Duration")}
              sideText="min"
              min={0}
              placeholder="60"
              description={t("Shown to the driver so they can plan for it.")}
            />
          </FormControl>
          <FormControl className="col-span-2">
            <InputField
              control={control}
              name="contentUrl"
              label={t("Link")}
              placeholder="https://"
              rules={{ required: delivery === "Online" }}
              description={
                delivery === "Online"
                  ? "Opened from Dash. Required for online courses."
                  : "Optional material the driver can open from Dash."
              }
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="passingScore"
              label={t("Passing score")}
              placeholder={t("No score")}
              sideText="%"
              description={
                passingScore
                  ? "The office records the score; below this mark is a fail."
                  : selfServe
                    ? "Leave empty and the driver's acknowledgement completes the course."
                    : "Leave empty for attendance-only courses."
              }
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="requiresAcknowledgement"
              label={t("Driver acknowledges")}
              description={t("The driver confirms in Dash that they took the course.")}
              position="left"
              outlined
            />
          </FormControl>
        </FormGroup>
      </section>

      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("Requirement & validity")}
          hint={t(
            "Required courses appear as slots on every matching worker; an overdue, failed or lapsed one leaves the worker unqualified.",
          )}
        />
        <FormGroup cols={2}>
          <FormControl className="col-span-2">
            <SwitchField
              control={control}
              name="isRequired"
              label={t("Required")}
              description={t("Assigned automatically on hire, rehire and driver-type change.")}
              position="left"
              outlined
            />
          </FormControl>
          {isRequired ? (
            <FormControl className="col-span-2">
              <MultiCheckboxField
                control={control}
                name="requiredForDriverTypes"
                label={t("Required for driver types")}
                options={driverTypeChoices}
                description={t("Leave all unchecked to require it for every worker.")}
              />
            </FormControl>
          ) : null}
          <FormControl>
            <NumberField
              control={control}
              name="dueDaysAfterAssignment"
              label={t("Due after")}
              sideText="days"
              min={0}
              max={730}
              placeholder="30"
              description={t("How long a worker has once the course is assigned. 0 = no due date.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="validityMonths"
              label={t("Valid for")}
              sideText="months"
              min={1}
              placeholder="12"
              description={t(
                "A completion lapses after this and a renewal is assigned. Leave empty for one-time courses.",
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="renewalWindowDays"
              label={t("Renewal alert window")}
              sideText="days"
              min={0}
              max={365}
              placeholder="30"
              description={t("How far ahead of lapsing the driver and the office are reminded.")}
            />
          </FormControl>
        </FormGroup>
      </section>
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
