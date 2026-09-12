import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useFieldArray, useFormContext } from "react-hook-form";

export function CarrierContactsForm() {
  const t = useT();

  const { control } = useFormContext<Carrier>();
  const { fields, append, remove } = useFieldArray({ control, name: "contacts" });

  const appendContact = () => {
    append({
      name: "",
      title: null,
      email: null,
      phone: null,
      isPrimary: false,
      receivesRateConfirmations: false,
    });
  };

  return (
    <div className="space-y-6">
      <FormSection
        title={t("Contacts")}
        description={t(
          "People at this carrier your team communicates with for dispatch and billing.",
        )}
      >
        <div className="flex flex-col gap-3">
          {fields.length === 0 && (
            <p className="text-muted-foreground text-sm">
              {t(
                "No contacts yet. Add dispatch, billing, and after-hours contacts so your team knows who to reach.",
              )}
            </p>
          )}

          {fields.map((field, index) => (
            <div key={field.id} className="rounded-md border p-3">
              <div className="mb-2 flex items-center justify-between">
                <p className="text-xs font-medium">{t("Contact {0}", index + 1)}</p>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  className="h-6 text-xs"
                  onClick={() => remove(index)}
                >
                  {t("Remove")}
                </Button>
              </div>

              <FormGroup cols={2}>
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.name`}
                    label={t("Name")}
                    placeholder={t("e.g., Jane Smith")}
                    rules={{ required: true }}
                    description={t("Contact's full name.")}
                    maxLength={255}
                  />
                </FormControl>
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.title`}
                    label={t("Title")}
                    placeholder={t("e.g., Dispatch Manager")}
                    description={t("Contact's role at the carrier.")}
                    maxLength={100}
                  />
                </FormControl>
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.email`}
                    label={t("Email")}
                    placeholder={t("e.g., jane@carrier.com")}
                    description={t("Required when this contact receives rate confirmations.")}
                    maxLength={255}
                  />
                </FormControl>
                <FormControl>
                  <PhoneNumberField
                    control={control}
                    name={`contacts.${index}.phone`}
                    label={t("Phone")}
                    placeholder={t("Phone")}
                    description={t("Direct phone number for this contact.")}
                  />
                </FormControl>
                <FormControl>
                  <SwitchField
                    control={control}
                    name={`contacts.${index}.isPrimary`}
                    label={t("Primary Contact")}
                    description={t("Main point of contact for this carrier.")}
                  />
                </FormControl>
                <FormControl>
                  <SwitchField
                    control={control}
                    name={`contacts.${index}.receivesRateConfirmations`}
                    label={t("Receives Rate Confirmations")}
                    description={t("Send rate confirmations to this contact's email.")}
                  />
                </FormControl>
              </FormGroup>
            </div>
          ))}

          <div>
            <Button type="button" size="sm" variant="outline" onClick={appendContact}>
              {t("Add contact")}
            </Button>
          </div>
        </div>
      </FormSection>
    </div>
  );
}
