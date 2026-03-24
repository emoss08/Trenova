import { Kbd } from "@/components/ui/kbd";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { Search } from "lucide-react";

export function QuickSearchTrigger() {
  const setOpen = useCommandPaletteStore((state) => state.setOpen);

  return (
    <button
      type="button"
      onClick={() => setOpen(true)}
      className="flex w-full items-center gap-2 rounded-md border border-sidebar-border bg-background px-2 py-1.5 text-left text-sidebar-foreground group-data-[collapsible=icon]:hidden hover:bg-sidebar-accent"
      aria-label="Open quick search"
    >
      <Search className="size-4 text-muted-foreground" />
      <span className="flex-1 text-sm text-muted-foreground">Quick search...</span>
      <Kbd className="h-5 px-1.5 text-2xs">Ctrl K</Kbd>
    </button>
  );
}
