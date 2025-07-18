import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy, memo } from "react";

const PatternConfigForm = lazy(() => import("./pattern-config-form"));

export function PatternConfig() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags title="Pattern Config" description="Pattern Config" />
      <Header />
      <QueryLazyComponent queryKey={queries.patternConfig.get._def}>
        <PatternConfigForm />
      </QueryLazyComponent>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Pattern Config</h1>
        <p className="text-muted-foreground">
          Configure and manage your pattern detection settings
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
