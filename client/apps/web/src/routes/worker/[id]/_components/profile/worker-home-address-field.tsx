import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { integration } from "@/lib/queries/integration";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Worker } from "@trenova/shared/types/worker";
import { useFormContext } from "react-hook-form";

/**
 * The street line of the worker's residence. With a Google Maps key it looks the address up,
 * which also records where home is, so dispatch can credit a move that heads home.
 */
export function WorkerHomeAddressField() {
  const t = useT();
  const { control } = useFormContext<Worker>();
  const googleMapsQuery = useQuery({
    ...integration.runtimeConfig("GoogleMaps"),
    staleTime: 5 * 60 * 1000,
  });

  if (googleMapsQuery.data?.config.apiKey) {
    return (
      <AddressField
        control={control}
        required
        nameField={null}
        placeIdField="homePlaceId"
        latitudeField="homeLatitude"
        longitudeField="homeLongitude"
        label={t("Street address")}
        placeholder={t("Start typing an address")}
        description={t("Pick it from the list so dispatch knows where home is.")}
      />
    );
  }

  return (
    <InputField
      control={control}
      required
      name="addressLine1"
      label={t("Street address")}
      placeholder={t("e.g. 1200 W Main St")}
      maxLength={150}
    />
  );
}
