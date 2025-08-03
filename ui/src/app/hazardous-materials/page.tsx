/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const HazardousMaterialTable = lazy(
  () => import("./_components/hazardous-material-table"),
);

export function HazardousMaterials() {
  return (
    <>
      <MetaTags title="Hazardous Materials" description="Hazardous Materials" />
      <LazyComponent>
        <HazardousMaterialTable />
      </LazyComponent>
    </>
  );
}
