import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { statusChoices } from "@/lib/choices";
import type { Customer } from "@/types/customer";
import {
  BuildingIcon,
  LinkIcon,
  PackageIcon,
  UserIcon,
} from "lucide-react";
import { useFormContext } from "react-hook-form";

function SectionHeader({
  icon: Icon,
  title,
  description,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
        <Icon className="size-4" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-semibold tracking-tight">
          {title}
        </h3>
        <p className="mt-1 text-xs text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

export function CustomerForm() {
  const { control } = useFormContext<Customer>();

  return (
    <div className="space-y-6">
      <SectionHeader
        icon={UserIcon}
        title="General Information"
        description="Core identifiers used across the system to reference this customer"
      />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            label="Status"
            placeholder="Status"
            description="Controls whether this customer appears in active lookups. Inactive customers cannot be assigned to new shipments."
            options={statusChoices}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            placeholder="e.g., ACME"
            description="Short alphanumeric identifier used in shipment references, invoice numbers, and quick-search. Must be unique across your organization."
            maxLength={10}
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="e.g., Acme Logistics Inc."
            description="Full legal or trading name of the customer. This appears on invoices, BOLs, and all printed documents."
            maxLength={255}
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={BuildingIcon}
        title="Address"
        description="Primary business address used for invoicing and geocoded distance calculations"
      />
      <FormGroup cols={2}>
        <FormControl cols="full" id="address-field-container">
          <AddressField control={control} />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name="addressLine2"
            label="Address Line 2"
            placeholder="Suite, floor, building, etc."
            description="Additional address details such as suite number, floor, or building name."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="city"
            rules={{ required: true }}
            label="City"
            placeholder="City"
            description="City where the customer's primary office or billing address is located."
          />
        </FormControl>
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            name="stateId"
            label="State"
            placeholder="State"
            description="U.S. state for the billing address. Used for tax jurisdiction determination and regional reporting."
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            rules={{ required: true }}
            control={control}
            name="postalCode"
            label="Postal Code"
            placeholder="e.g., 90210"
            description="ZIP or ZIP+4 code. Used for geocoding, mileage calculations, and tax jurisdiction lookups."
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={LinkIcon}
        title="External Identifiers"
        description="Link this customer to records in external systems like your ERP, CRM, or mapping provider"
      />
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="placeId"
            label="Place ID"
            placeholder="Automatically populated"
            description="Google Maps Place ID, set automatically when an address is geocoded. Used for precise location matching and map rendering."
            readOnly
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalId"
            label="External ID"
            placeholder="e.g., CRM-10042"
            description="Identifier from an external system (ERP, CRM, EDI partner ID). Useful for data imports, API integrations, and cross-system reconciliation."
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={PackageIcon}
        title="Shipment Consolidation"
        description="Control whether multiple shipments for this customer can share trailer space"
      />
      <FormGroup cols={2}>
        <FormControl>
          <SwitchField
            control={control}
            name="allowConsolidation"
            label="Allow Consolidation"
            description="Permit this customer's shipments to be combined with other shipments on the same trailer to improve load utilization and reduce costs."
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="exclusiveConsolidation"
            label="Exclusive Consolidation"
            description="Only consolidate with other shipments from this same customer — never mix with other customers' freight. Requires 'Allow Consolidation' to be enabled."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="consolidationPriority"
            label="Priority"
            placeholder="1"
            description="Lower numbers are consolidated first when trailer space is limited. Use 1 for highest priority customers."
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
