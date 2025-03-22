import { LazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { lazy, memo } from "react";
import { FormProvider, useForm } from "react-hook-form";

const ShipmentTable = lazy(() => import("./_components/shipment-table"));

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
        <LazyComponent>
          <FormProvider {...form}>
            <ShipmentTable />
          </FormProvider>
        </LazyComponent>
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
