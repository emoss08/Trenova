import { useT } from "@trenova/shared/i18n/use-t";
import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Separator } from "@trenova/shared/components/ui/separator";
import { statusChoices } from "@/lib/choices";
import type { Customer } from "@trenova/shared/types/customer";
import { BuildingIcon, LinkIcon, PackageIcon, UserIcon } from "lucide-react";
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
      <div className="bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-lg">
        <Icon className="size-4" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-semibold tracking-tight">{title}</h3>
        <p className="text-muted-foreground mt-1 text-xs">{description}</p>
      </div>
    </div>
  );
}

export function CustomerForm() {
  const t = useT();

  const { control } = useFormContext<Customer>();

  return (
    <div className="space-y-6">
      <SectionHeader
        icon={UserIcon}
        title={t("General Information")}
        description={t("Core identifiers used across the system to reference this customer")}
      />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            label={t("Status")}
            placeholder={t("Status")}
            description={t(
              "Controls whether this customer appears in active lookups. Inactive customers cannot be assigned to new shipments.",
            )}
            options={statusChoices}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label={t("Code")}
            placeholder={t("e.g., ACME")}
            description={t(
              "Short alphanumeric identifier used in shipment references, invoice numbers, and quick-search. Must be unique across your organization.",
            )}
            maxLength={10}
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label={t("Name")}
            placeholder={t("e.g., Acme Logistics Inc.")}
            description={t(
              "Full legal or trading name of the customer. This appears on invoices, BOLs, and all printed documents.",
            )}
            maxLength={255}
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={BuildingIcon}
        title={t("Address")}
        description={t(
          "Primary business address used for invoicing and geocoded distance calculations",
        )}
      />
      <FormGroup cols={2}>
        <FormControl cols="full" id="address-field-container">
          <AddressField control={control} />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name="addressLine2"
            label={t("Address Line 2")}
            placeholder={t("Suite, floor, building, etc.")}
            description={t(
              "Additional address details such as suite number, floor, or building name.",
            )}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="city"
            rules={{ required: true }}
            label={t("City")}
            placeholder={t("City")}
            description={t(
              "City where the customer's primary office or billing address is located.",
            )}
          />
        </FormControl>
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            name="stateId"
            label={t("State")}
            placeholder={t("State")}
            description={t(
              "U.S. state for the billing address. Used for tax jurisdiction determination and regional reporting.",
            )}
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            rules={{ required: true }}
            control={control}
            name="postalCode"
            label={t("Postal Code")}
            placeholder={t("e.g., 90210")}
            description={t(
              "ZIP or ZIP+4 code. Used for geocoding, mileage calculations, and tax jurisdiction lookups.",
            )}
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={LinkIcon}
        title={t("External Identifiers")}
        description={t(
          "Link this customer to records in external systems like your ERP, CRM, or mapping provider",
        )}
      />
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="placeId"
            label={t("Place ID")}
            placeholder={t("Automatically populated")}
            description={t(
              "Google Maps Place ID, set automatically when an address is geocoded. Used for precise location matching and map rendering.",
            )}
            readOnly
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalId"
            label={t("External ID")}
            placeholder={t("e.g., CRM-10042")}
            description={t(
              "Identifier from an external system (ERP, CRM, EDI partner ID). Useful for data imports, API integrations, and cross-system reconciliation.",
            )}
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={PackageIcon}
        title={t("Shipment Consolidation")}
        description={t(
          "Control whether multiple shipments for this customer can share trailer space",
        )}
      />
      <FormGroup cols={2}>
        <FormControl>
          <SwitchField
            control={control}
            name="allowConsolidation"
            label={t("Allow Consolidation")}
            description={t(
              "Permit this customer's shipments to be combined with other shipments on the same trailer to improve load utilization and reduce costs.",
            )}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="exclusiveConsolidation"
            label={t("Exclusive Consolidation")}
            description={t(
              "Only consolidate with other shipments from this same customer — never mix with other customers' freight. Requires 'Allow Consolidation' to be enabled.",
            )}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="consolidationPriority"
            label={t("Priority")}
            placeholder="1"
            description={t(
              "Lower numbers are consolidated first when trailer space is limited. Use 1 for highest priority customers.",
            )}
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
