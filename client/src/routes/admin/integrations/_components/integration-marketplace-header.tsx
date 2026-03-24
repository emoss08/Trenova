import { Input } from "@/components/ui/input";
import { SearchIcon, SparklesIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { searchParamsParser } from "../integration-marketplace-state";

export function IntegrationMarketplaceHeader() {
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);

  return (
    <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
      <div className="space-y-2">
        <div className="inline-flex items-center gap-2 rounded-full border border-border bg-background px-2.5 py-1 text-xs text-muted-foreground">
          <SparklesIcon className="size-3.5" />
          Marketplace
        </div>
        <h2 className="text-2xl font-semibold tracking-tight">Integrations and Connected Apps</h2>
        <p className="text-sm text-muted-foreground">
          Connect your stack, sync operational data, and manage telematics from one workspace.
        </p>
      </div>
      <div className="w-full lg:max-w-sm">
        <Input
          value={searchParams.query}
          onChange={(event) => setSearchParams({ query: event.target.value })}
          placeholder="Search integrations"
          className="h-9 bg-background"
          leftElement={<SearchIcon className="size-4 text-muted-foreground" />}
        />
      </div>
    </div>
  );
}
