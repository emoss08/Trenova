import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { useShipmentView } from "@/hooks/use-shipment-view";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { FormProvider, useForm } from "react-hook-form";
import ShipmentMap from "./_components/shipment-map";
import ShipmentTable from "./_components/shipment-table";

export function Shipment() {
  const { view } = useShipmentView();

  const form = useForm<ShipmentFilterSchema>({
    defaultValues: {
      search: undefined,
      status: undefined,
    },
  });

  return (
    <>
      <MetaTags title="Shipments" description="Shipments" />
      <SuspenseLoader>
        <FormProvider {...form}>
          {view === "list" ? <ShipmentTable /> : <ShipmentMap />}
        </FormProvider>
      </SuspenseLoader>
    </>
  );
}
