import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { memo } from "react";
import { FormProvider, useForm } from "react-hook-form";
import ShipmentTable from "./_components/shipment-table";

export function Shipment() {
  const form = useForm<ShipmentFilterSchema>({
    defaultValues: {
      search: undefined,
      status: undefined,
    },
  });

  return (
    <FormSaveProvider>
      <div className="space-y-6 p-6">
        <MetaTags title="Shipments" description="Shipments" />
        <Header />
        <SuspenseLoader>
          <FormProvider {...form}>
            <ShipmentTable />
          </FormProvider>
        </SuspenseLoader>
      </div>
    </FormSaveProvider>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Shipments</h1>
        <p className="text-muted-foreground">
          Manage and track all shipments in your system
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
