/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const LocationTable = lazy(() => import("./_components/location-table"));

export function Locations() {
  return (
    <>
      <MetaTags title="Locations" description="Locations" />
      <LazyComponent>
        <LocationTable />
      </LazyComponent>
    </>
  );
}
