import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { driverTypeChoices, statusChoices } from "@/lib/choices";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  CREDENTIAL_CATEGORY_LABELS,
  credentialCategorySchema,
  type CredentialTypeFormValues,
} from "@trenova/shared/types/worker-credential";
import { InfoIcon } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

const CATEGORY_OPTIONS = credentialCategorySchema.options.map((value) => ({
  value,
  label: CREDENTIAL_CATEGORY_LABELS[value],
}));

type CredentialTypeFormProps = {
  isEdit: boolean;
  isSystem?: boolean;
  profileField?: string | null;
  activeCredentialCount?: number;
};

export function CredentialTypeForm({
  isEdit,
  isSystem = false,
  profileField = null,
  activeCredentialCount = 0,
}: CredentialTypeFormProps) {
  const t = useT();

  const { control } = useFormContext<CredentialTypeFormValues>();
  const isRequired = useWatch({ control, name: "isRequired" });

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("General")}
          hint={
            isSystem
              ? "System types ship with Trenova. Their code is fixed; everything else is yours to tune."
              : "Name and code identify the credential on worker records and reports."
          }
        />
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label={t("Code")}
              placeholder={t("e.g. HAZMAT")}
              rules={{ required: true }}
              disabled={isSystem}
              description={t("Short unique identifier. Letters, digits, dashes and underscores.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label={t("Name")}
              placeholder={t("e.g. Hazmat Endorsement")}
              rules={{ required: true }}
              description={t(
                "Shown on worker records, in credential pickers and on compliance reports.",
              )}
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
              description={t("Groups the credential on the worker's record and in the type list.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              options={statusChoices}
              rules={{ required: true }}
              isReadOnly={Boolean(profileField)}
              placeholder={t("Select a status")}
              description={
                profileField
                  ? "Mirrors a worker-profile field and cannot be deactivated."
                  : isEdit && activeCredentialCount > 0
                    ? `${activeCredentialCount} worker${activeCredentialCount === 1 ? " holds" : "s hold"} this credential; archive those first to deactivate.`
                    : "Inactive types are hidden from pickers and stop counting toward compliance."
              }
            />
          </FormControl>
          <FormControl className="col-span-2">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("What this credential is and the regulation behind it")}
              maxLength={1000}
              description={t("Optional notes on what the credential covers and why it is tracked.")}
            />
          </FormControl>
        </FormGroup>
      </section>

      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("Compliance")}
          hint={t(
            "Required types appear as slots on every matching worker; a missing or expired one makes the worker non-compliant.",
          )}
        />
        <Alert variant="default">
          <InfoIcon className="size-4" />
          <AlertTitle>{t("One active credential per worker")}</AlertTitle>
          <AlertDescription>
            {t(
              "A worker holds a single active credential of each type. Renewing supersedes the earlier one instead of adding a second; the renewal window and validity below drive when that renewal is prompted.",
            )}
          </AlertDescription>
        </Alert>
        <FormGroup cols={2}>
          <FormControl className="col-span-2">
            <SwitchField
              control={control}
              name="isRequired"
              label={t("Required")}
              description={t("Workers must hold a valid credential of this type to be compliant.")}
              position="left"
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
              name="renewalWindowDays"
              label={t("Renewal alert window")}
              sideText="days"
              min={0}
              max={365}
              placeholder="30"
              description={t("How far ahead of expiry the credential is flagged as expiring soon.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="validityMonths"
              label={t("Typical validity")}
              sideText="months"
              min={1}
              placeholder="24"
              description={t(
                "Pre-fills the expiry from the issue date when adding or renewing. Leave empty if it varies.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requiresNumber"
              label={t("Requires a number")}
              description={t("The credential cannot be saved without an identifying number.")}
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requiresDocument"
              label={t("Requires a document")}
              description={t("A scan must be attached before the credential can be verified.")}
              position="left"
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
