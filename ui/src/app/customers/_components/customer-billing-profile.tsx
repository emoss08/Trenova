import { MultiSelectAutocompleteField } from "@/components/fields/async-multi-select";
import { ColorOptionValue } from "@/components/fields/select-components";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { billingCycleTypeChoices } from "@/lib/choices";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { DocumentTypeSchema } from "@/lib/schemas/document-type-schema";
import { useFormContext } from "react-hook-form";
import { BillingControlOverrides } from "./customer-billing-control-override";

export default function CustomerBillingProfile() {
  const { control } = useFormContext<CustomerSchema>();

  return (
    <div className="size-full">
      <div className="flex select-none flex-col px-4">
        <h2 className="mt-2 text-2xl font-semibold">Billing Profile</h2>
        <p className="text-xs text-muted-foreground">
          Configure billing settings for the customer.
        </p>
      </div>
      <Separator className="mt-2" />
      <div className="p-4">
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="billingProfile.billingCycleType"
              rules={{ required: true }}
              label="Billing Cycle Type"
              options={billingCycleTypeChoices}
              description="Select the billing cycle type for the customer."
            />
          </FormControl>
          <FormControl>
            <MultiSelectAutocompleteField<DocumentTypeSchema, CustomerSchema>
              control={control}
              name="billingProfile.documentTypeIds"
              label="Document Types"
              rules={{ required: true }}
              link="/document-types/"
              placeholder="Select Document Types"
              description="Select the document types that are required for this customer billing profile."
              getOptionValue={(option) => option.id || ""}
              getOptionLabel={(option) => option.name}
              renderOption={(option) => (
                <div className="flex flex-col gap-0.5 items-start size-full">
                  <ColorOptionValue color={option.color} value={option.code} />
                  {option?.description && (
                    <span className="text-2xs text-muted-foreground truncate w-full">
                      {option?.description}
                    </span>
                  )}
                </div>
              )}
              getDisplayValue={(option) => option.name}
              renderBadge={(option) => option.name}
            />
          </FormControl>
        </FormGroup>
        <BillingControlOverrides />
      </div>
    </div>
  );
}
