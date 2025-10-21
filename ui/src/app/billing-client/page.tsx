/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { AnalyticsPage } from "@/types/analytics";
import { lazy, memo } from "react";

const BillingQueueAnalytics = lazy(
  () => import("./_components/billing-queue-analytics"),
);

export function BillingClient() {
  return (
    <div className="space-y-6 p-6">
      <MetaTags title="Billing Client" description="Billing Client" />
      <Header />
      <QueryLazyComponent queryKey={["analytics", AnalyticsPage.BillingClient]}>
        <BillingQueueAnalytics />
      </QueryLazyComponent>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Billing Client</h1>
        <p className="text-muted-foreground">
          Billing client is a service that automatically generates billing
          invoices for your clients
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
