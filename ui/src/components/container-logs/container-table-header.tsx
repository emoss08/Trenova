/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { useContainerLogStore } from "@/stores/docker-store";
import { RefreshCw } from "lucide-react";

export function ContainerTableHeader({ refetch }: { refetch: () => void }) {
  const [showAll, setShowAll] = useContainerLogStore.use("showAll");
  return (
    <div className="flex items-center justify-end">
      <div className="flex items-center gap-2">
        <div className="flex items-center space-x-2">
          <Checkbox
            id="show-all"
            checked={showAll}
            onCheckedChange={(checked) => setShowAll(!!checked)}
          />
          <label
            htmlFor="show-all"
            className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
          >
            Show all containers
          </label>
        </div>
        <Button variant="outline" onClick={() => refetch()}>
          <RefreshCw className="size-4" />
          Refresh
        </Button>
      </div>
    </div>
  );
}
