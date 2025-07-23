/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const LocationCategoryTable = lazy(
  () => import("./_components/location-category-table"),
);

export function LocationCategories() {
  return (
    <>
      <MetaTags title="Location Categories" description="Location Categories" />
      <LazyComponent>
        <LocationCategoryTable />
      </LazyComponent>
    </>
  );
}
