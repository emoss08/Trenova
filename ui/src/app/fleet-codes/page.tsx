/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const FleetCodesDataTable = lazy(
  () => import("./_components/fleet-code-table"),
);

export function FleetCodes() {
  return (
    <>
      <MetaTags title="Fleet Codes" description="Fleet Codes" />
      <LazyComponent>
        <FleetCodesDataTable />
      </LazyComponent>
    </>
  );
}
