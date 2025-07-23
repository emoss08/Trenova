/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { IntegrationGrid } from "./_components/integration-grid";

export function IntegrationsPage() {
  return (
    <>
      <MetaTags title="Apps & Integrations" />
      <div className="flex flex-col">
        <h1 className="text-xl font-semibold">Apps & Integrations</h1>
        <p className="text-sm text-muted-foreground">
          Enhance your Trenova experience with a wide variety of add-ons and
          integrations
        </p>
      </div>
      <FormSaveProvider>
        <IntegrationGrid />
      </FormSaveProvider>
    </>
  );
}
