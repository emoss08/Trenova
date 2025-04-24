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
