import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { billingCycleTypeChoices } from "@/lib/choices";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
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
              label="Billing Cycle Type"
              options={billingCycleTypeChoices}
            />
          </FormControl>
        </FormGroup>
        <BillingControlOverrides />
      </div>
    </div>
  );
}
