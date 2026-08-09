import { InputField } from "@/components/fields/input-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useFieldArray, useFormContext } from "react-hook-form";

export function CarrierContactsForm() {
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
        title="Contacts"
        description="People at this carrier your team communicates with for dispatch and billing."
      >
        <div className="flex flex-col gap-3">
          {fields.length === 0 && (
            <p className="text-muted-foreground text-sm">
              No contacts yet. Add dispatch, billing, and after-hours contacts so your team knows
              who to reach.
            </p>
          )}

          {fields.map((field, index) => (
            <div key={field.id} className="rounded-md border p-3">
              <div className="mb-2 flex items-center justify-between">
                <p className="text-xs font-medium">Contact {index + 1}</p>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  className="h-6 text-xs"
                  onClick={() => remove(index)}
                >
                  Remove
                </Button>
              </div>

              <FormGroup cols={2}>
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.name`}
                    label="Name"
                    placeholder="e.g., Jane Smith"
                    rules={{ required: true }}
                    description="Contact's full name."
                    maxLength={255}
                  />
                </FormControl>
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.title`}
                    label="Title"
                    placeholder="e.g., Dispatch Manager"
                    description="Contact's role at the carrier."
                    maxLength={100}
                  />
                </FormControl>
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.email`}
                    label="Email"
                    placeholder="e.g., jane@carrier.com"
                    description="Required when this contact receives rate confirmations."
                    maxLength={255}
                  />
                </FormControl>
                <FormControl>
                  <PhoneNumberField
                    control={control}
                    name={`contacts.${index}.phone`}
                    label="Phone"
                    placeholder="Phone"
                    description="Direct phone number for this contact."
                  />
                </FormControl>
                <FormControl>
                  <SwitchField
                    control={control}
                    name={`contacts.${index}.isPrimary`}
                    label="Primary Contact"
                    description="Main point of contact for this carrier."
                  />
                </FormControl>
                <FormControl>
                  <SwitchField
                    control={control}
                    name={`contacts.${index}.receivesRateConfirmations`}
                    label="Receives Rate Confirmations"
                    description="Send rate confirmations to this contact's email."
                  />
                </FormControl>
              </FormGroup>
            </div>
          ))}

          <div>
            <Button type="button" size="sm" variant="outline" onClick={appendContact}>
              Add contact
            </Button>
          </div>
        </div>
      </FormSection>
    </div>
  );
}
