import { AutocompleteField } from "@/components/fields/autocomplete";
import { CheckboxField } from "@/components/fields/checkbox-field";
import {
  AutoCompleteDateField
} from "@/components/fields/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { ratingMethodChoices } from "@/lib/choices";
import { CustomerSchema } from "@/lib/schemas/customer-schema";
import { Shipment } from "@/types/shipment";
import { faWallet } from "@fortawesome/pro-solid-svg-icons";
import { useFormContext } from "react-hook-form";

export default function RatingDetails() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-x-2">
          <div className="border-border flex size-10 items-center justify-center rounded-lg border">
            <Icon icon={faWallet} className="size-5" />
          </div>
          <div className="flex flex-col gap-0.5">
            <h3 className="text-lg font-semibold">Rating Details</h3>
            <p className="text-muted-foreground text-xs font-normal">
              Provides a breakdown of freight charges, additional fees, and the
              total cost for the shipment.
            </p>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <RatingForm />
      </CardContent>
    </Card>
  );
}

function RatingForm() {
  const { control } = useFormContext<Shipment>();

  return (
    <FormGroup cols={2} className="gap-4">
      <FormControl>
        <AutocompleteField<CustomerSchema, Shipment>
          name="customerId"
          control={control}
          link="/customers/"
          label="Customer"
          rules={{ required: true }}
          placeholder="Select Customer"
          description="Select the customer of the shipment"
          getOptionValue={(option) => option.code}
          getDisplayValue={(option) => option.code}
          renderOption={(option) => option.code}
          // hasPermission
          // hasPopoutWindow
          // popoutLink="/customers/"
          // popoutLinkLabel="Customer"
          // valueKey={["code"]}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="ratingMethod"
          label="Rating Method"
          placeholder="Rating Method"
          description="Rating Method"
          options={ratingMethodChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="ratingUnit"
          label="Rating Unit"
          placeholder="Rating Unit"
          description="Rating Unit"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="otherChargeAmount"
          label="Other Charge Amount"
          placeholder="Other Charge Amount"
          description="Other Charge Amount"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="freightChargeAmount"
          label="Freight Charge Amount"
          placeholder="Freight Charge Amount"
          description="Freight Charge Amount"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="totalChargeAmount"
          label="Total Charge Amount"
          placeholder="Total Charge Amount"
          description="The total amount for the shipment, including the standard rate and additional charges."
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField
          control={control}
          rules={{ required: true }}
          name="readyToBillDate"
          label="Ready to Bill Date"
          placeholder="Ready to Bill Date"
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField
          control={control}
          rules={{ required: true }}
          name="sentToBillingDate"
          label="Sent to Billing Date"
          placeholder="Sent to Billing Date"
        />
      </FormControl>
      <FormControl>
        <CheckboxField
          outlined
          control={control}
          name="readyToBill"
          label="Ready to Bill"
          description="Ready to Bill"
        />
      </FormControl>
      <FormControl>
        <CheckboxField
          outlined
          control={control}
          name="sentToBilling"
          label="Sent to Billing"
          description="Sent to Billing"
        />
      </FormControl>
      <FormControl>
        <CheckboxField
          outlined
          control={control}
          name="billed"
          label="Billed"
          description="Billed"
        />
      </FormControl>
    </FormGroup>
  );
}
