/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const DocumentTypesDataTable = lazy(
  () => import("./_components/document-type-table"),
);

export function DocumentTypes() {
  return (
    <>
      <MetaTags title="Document Types" description="Document Types" />
      <LazyComponent>
        <DocumentTypesDataTable />
      </LazyComponent>
    </>
  );
}
