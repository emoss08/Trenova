import { useT } from "@trenova/shared/i18n/use-t";
import { NumberField } from "@/components/fields/number-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { CAPABILITIES, getProfile, hasCapability } from "@trenova/shared/lib/capability";
import { shipment } from "@/lib/queries/shipment";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { type FieldPath, useFormContext, useWatch } from "react-hook-form";

/**
 * A stop's or commodity line's own temperature range, shown when the mode controls temperature or
 * the line already carries one (a partner's tender can set it on any load).
 */
export function TemperatureRangeFields({
  prefix,
  description,
}: {
  /** The form path of the stop or line, such as `moves.0.stops.1` or `commodities.2`. */
  prefix: string;
  description: string;
}) {
  const t = useT();
  const { control } = useFormContext<Shipment>();
  const { data: shipmentUIPolicy } = useQuery({ ...shipment.uiPolicy() });
  const minName = `${prefix}.temperatureMin` as FieldPath<Shipment>;
  const maxName = `${prefix}.temperatureMax` as FieldPath<Shipment>;
  const min = useWatch({ control, name: minName });
  const max = useWatch({ control, name: maxName });

  const profile = getProfile(shipmentUIPolicy);
  const controlled = !profile || hasCapability(profile, CAPABILITIES.temperatureControl);
  if (!controlled && min == null && max == null) return null;

  return (
    <FormGroup cols={2} dense>
      <FormControl>
        <NumberField
          control={control}
          name={minName}
          label={t("Temperature min")}
          description={description}
          placeholder={t("No minimum")}
          sideText={t("°F")}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name={maxName}
          label={t("Temperature max")}
          description={description}
          placeholder={t("No maximum")}
          sideText={t("°F")}
        />
      </FormControl>
    </FormGroup>
  );
}
