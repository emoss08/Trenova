import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { useShipmentParams } from "@/hooks/use-shipment-params";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { FormProvider, useForm } from "react-hook-form";
import ShipmentTable from "./_components/shipment-table";

export function Shipment() {
  const { view } = useShipmentParams();

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
          <ShipmentTable />
        </FormProvider>
      </SuspenseLoader>
    </>
  );
}
