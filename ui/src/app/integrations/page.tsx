import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { TooltipProvider } from "@/components/ui/tooltip";
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
        <TooltipProvider>
          <IntegrationGrid />
        </TooltipProvider>
      </FormSaveProvider>
    </>
  );
}
