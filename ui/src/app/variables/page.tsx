import { DataTableLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { lazy, memo } from "react";

const VariableContent = lazy(() => import("./_components/variable-table"));

export function Variables() {
  return (
    <FormSaveProvider>
      <div className="space-y-6 p-6">
        <MetaTags
          title="Variables & Formats"
          description="Variables & Formats"
        />
        <Header />
        <DataTableLazyComponent>
          <VariableContent />
        </DataTableLazyComponent>
      </div>
    </FormSaveProvider>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Variables & Formats
        </h1>
        <p className="text-muted-foreground">
          Variables are placeholders that automatically fill in with real data
          from your system, while formats control how that data appears.
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
