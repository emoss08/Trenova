/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy, memo } from "react";

const ConsolidationSettingForm = lazy(
  () => import("./_components/consolidation-setting-form"),
);
export function ConsolidationSetting() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags
        title="Consolidation Settings"
        description="Consolidation Settings"
      />
      <Header />
      <QueryLazyComponent
        queryKey={queries.organization.getConsolidationSettings._def}
      >
        <ConsolidationSettingForm />
      </QueryLazyComponent>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Consolidation Settings
        </h1>
        <p className="text-muted-foreground">
          Configure and manage your consolidation settings
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
