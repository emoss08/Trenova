import {
  AccessorialChargeAutocompleteField,
  CommodityAutocompleteField,
  CustomerAutocompleteField,
  FormulaTemplateAutocompleteField,
  LocationAutocompleteField,
  ServiceTypeAutocompleteField,
  ShipmentTypeAutocompleteField,
} from "@/components/autocomplete-fields";
import type { EDIMappingEntityType } from "@/types/edi";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import {
  getTargetOptionLabel,
  getTargetOptionValue,
  type TargetLookupSelection,
} from "./edi-display-utils";

type TargetLookupProps = {
  entityType: EDIMappingEntityType;
  label?: string;
  value: string;
  onChange: (target: TargetLookupSelection) => void;
};

export function TargetLookup({ entityType, label, value, onChange }: TargetLookupProps) {
  const { control, setValue } = useForm<{ targetId: string }>({
    defaultValues: { targetId: value },
  });

  useEffect(() => {
    setValue("targetId", value);
  }, [setValue, value]);

  const handleOptionChange = (option: unknown) => {
    onChange({
      targetId: getTargetOptionValue(option),
      targetLabel: getTargetOptionLabel(option),
    });
  };

  const commonProps = {
    control,
    name: "targetId" as const,
    label,
    placeholder: "Select local record",
    clearable: true,
    onOptionChange: handleOptionChange,
  };

  switch (entityType) {
    case "Customer":
      return <CustomerAutocompleteField {...commonProps} />;
    case "Location":
      return <LocationAutocompleteField {...commonProps} />;
    case "Commodity":
      return <CommodityAutocompleteField {...commonProps} />;
    case "AccessorialCharge":
      return <AccessorialChargeAutocompleteField {...commonProps} />;
    case "ServiceType":
      return <ServiceTypeAutocompleteField {...commonProps} />;
    case "ShipmentType":
      return <ShipmentTypeAutocompleteField {...commonProps} />;
    case "FormulaTemplate":
      return <FormulaTemplateAutocompleteField {...commonProps} />;
    default:
      return null;
  }
}
