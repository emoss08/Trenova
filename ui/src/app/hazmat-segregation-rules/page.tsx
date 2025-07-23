/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const HazmatSegregationRuleTable = lazy(
  () => import("./_components/hazmat-segregation-rule-table"),
);

export function HazmatSegregationRules() {
  return (
    <>
      <MetaTags
        title="Hazmat Segregation Rules"
        description="Hazmat Segregation Rules"
      />
      <Header />
      <LazyComponent>
        <HazmatSegregationRuleTable />
      </LazyComponent>
    </>
  );
}

function Header() {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Hazmat Segregation Rules
        </h1>
        <p className="text-muted-foreground">
          Manage and configure hazmat segregation rules for your organization
        </p>
      </div>
    </div>
  );
}
