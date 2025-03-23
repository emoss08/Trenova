import { QueryLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy, memo } from "react";

const BillingControlForm = lazy(
  () => import("./_components/billing-control-form"),
);

export function BillingControl() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags title="Billing Control" description="Billing Control" />
      <Header />
      <QueryLazyComponent
        queryKey={queries.organization.getBillingControl._def}
      >
        <FormSaveProvider>
          <BillingControlForm />
        </FormSaveProvider>
      </QueryLazyComponent>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Billing Control</h1>
        <p className="text-muted-foreground">
          Configure and manage your billing control settings
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
