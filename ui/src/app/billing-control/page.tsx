/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { QueryLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { Icon } from "@/components/ui/icons";
import { queries } from "@/lib/queries";
import { faExclamationTriangle } from "@fortawesome/pro-regular-svg-icons";
import { lazy, memo } from "react";

const BillingControlForm = lazy(
  () => import("./_components/billing-control-form"),
);

export function BillingControl() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags title="Billing Control" description="Billing Control" />
      <Header />
      <BillingControlAlert />
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

const BillingControlAlert = memo(() => {
  return (
    <div className="flex bg-amber-500/10 border border-amber-600/50 p-4 rounded-md justify-between items-center mb-4 w-full">
      <div className="flex items-center gap-3 w-full text-amber-600">
        <Icon icon={faExclamationTriangle} className="size-5" />
        <div className="flex flex-col">
          <p className="text-sm font-semibold">
            Critical Financial Configuration
          </p>
          <p className="text-xs">
            Billing Control settings directly impact your organization&apos;s
            revenue processing, financial reporting, and customer invoicing.
            Changes to these settings should be made infrequently and only after
            thorough review by financial stakeholders.
          </p>
        </div>
      </div>
    </div>
  );
});
BillingControlAlert.displayName = "BillingControlAlert";
