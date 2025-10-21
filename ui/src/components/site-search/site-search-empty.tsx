/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Badge } from "../ui/badge";

export function SiteSearchEmpty({ searchQuery }: { searchQuery: string }) {
  // TODO: Get popular searches from backend
  const popularSearches = [
    "Dashboard",
    "Shipments",
    "Invoices",
    "Reports",
    "Settings",
  ];

  return (
    <div className="flex flex-col items-center justify-center py-6">
      <div className="text-sm font-medium">
        No results found for &quot;{searchQuery}&quot;
      </div>
      <div className="text-muted-foreground mt-2 text-sm">
        Try one of these popular searches:
      </div>
      <div className="mt-4 flex flex-wrap justify-center gap-2">
        {popularSearches.map((search) => (
          <Badge
            key={search}
            variant="orange"
            className="cursor-pointer"
            // onClick={() => onPopularSearch(search)}
          >
            {search}
          </Badge>
        ))}
      </div>
    </div>
  );
}
