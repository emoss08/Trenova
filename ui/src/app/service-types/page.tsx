/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const ServiceTypeTable = lazy(() => import("./_components/service-type-table"));

export function ServiceTypes() {
  return (
    <>
      <MetaTags title="Service Types" description="Service Types" />
      <LazyComponent>
        <ServiceTypeTable />
      </LazyComponent>
    </>
  );
}
