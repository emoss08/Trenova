/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Icon } from "@/components/ui/icons";
import { Input } from "@/components/ui/input";
import { faSearch } from "@fortawesome/pro-solid-svg-icons";

export function DataTableSearch() {
  return (
    <div className="flex items-center gap-2">
      <Input
        icon={<Icon icon={faSearch} className="size-3 text-muted-foreground" />}
        placeholder="Filter..."
        className="w-full"
      />
    </div>
  );
}
