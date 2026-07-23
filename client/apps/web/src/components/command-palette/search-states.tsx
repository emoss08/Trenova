import { EmptyState } from "@/components/empty-state";
import { Spinner } from "@/components/ui/spinner";
import { Package, Search, Truck } from "lucide-react";

export function SearchLoading({ query }: { query: string }) {
  return (
    <div className="flex h-60 flex-col items-center justify-center gap-2">
      <Spinner variant="bars" className="size-8 text-primary" />
      <div className="flex flex-col items-center text-center">
        <h3 className="text-lg font-semibold text-foreground">
          Searching for <span className="font-mono">&quot;{query}&quot;</span>...
        </h3>
        <p className="text-sm text-muted-foreground">This may take a few seconds...</p>
      </div>
    </div>
  );
}

export function SearchEmpty() {
  return (
    <div className="flex items-center justify-center p-4">
      <EmptyState
        title="No results found"
        description="Try adjusting your search query"
        icons={[Search, Package, Truck]}
        className="size-full border-none bg-transparent hover:bg-transparent"
      />
    </div>
  );
}

export function SearchKeepTyping() {
  return (
    <div className="flex h-60 flex-col items-center justify-center gap-2">
      <Spinner variant="infinite" className="size-8 text-primary" />
      <div className="flex flex-col items-center text-center">
        <p className="text-sm font-medium text-foreground">Keep typing to search records...</p>
      </div>
    </div>
  );
}

export function SearchError() {
  return (
    <div className="flex items-center justify-center p-4">
      <EmptyState
        title="Search unavailable"
        description="Record search is temporarily unavailable. Try again later."
        icons={[Search]}
        className="size-full border-none bg-transparent hover:bg-transparent"
      />
    </div>
  );
}
