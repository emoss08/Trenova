import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import type { HomeWidgetCatalog, HomeWidgetOption } from "@/lib/graphql/home-layout";
import { cn } from "@trenova/shared/lib/utils";
import { PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";

export type AddWidgetDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  catalog: HomeWidgetCatalog | undefined;
  loading: boolean;
  /** Keys already on the canvas, so a duplicate reads as a deliberate choice. */
  usedKeys: Set<string>;
  remaining: number;
  onAdd: (option: HomeWidgetOption) => void;
};

export function AddWidgetDialog({
  open,
  onOpenChange,
  catalog,
  loading,
  usedKeys,
  remaining,
  onAdd,
}: AddWidgetDialogProps) {
  const [search, setSearch] = useState("");

  const handleOpenChange = (next: boolean) => {
    if (!next) setSearch("");
    onOpenChange(next);
  };

  const grouped = useMemo(() => {
    if (!catalog) return [];
    const term = search.trim().toLowerCase();

    return catalog.categories
      .map((category) => ({
        category,
        widgets: catalog.widgets.filter(
          (widget) =>
            widget.category === category.key &&
            (term === "" ||
              widget.label.toLowerCase().includes(term) ||
              widget.description.toLowerCase().includes(term)),
        ),
      }))
      .filter((group) => group.widgets.length > 0);
  }, [catalog, search]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Add a widget</DialogTitle>
          <DialogDescription>
            {remaining > 0
              ? `You can add ${remaining} more.`
              : "This home screen has reached its widget limit."}
          </DialogDescription>
        </DialogHeader>

        <Input
          placeholder="Search widgets…"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />

        <ScrollArea className="max-h-100">
          <div className="flex flex-col gap-4 pr-2">
            {loading && <p className="text-muted-foreground py-6 text-center text-xs">Loading…</p>}

            {!loading && grouped.length === 0 && (
              <p className="text-muted-foreground py-6 text-center text-xs">
                No widgets match &ldquo;{search}&rdquo;.
              </p>
            )}

            {grouped.map(({ category, widgets }) => (
              <div key={category.key} className="flex flex-col gap-1.5">
                <div className="flex flex-col gap-0.5">
                  <span className="cc-label text-foreground">{category.label}</span>
                  <span className="text-2xs text-muted-foreground">{category.description}</span>
                </div>
                <div className="grid gap-1.5 sm:grid-cols-2">
                  {widgets.map((widget) => (
                    <button
                      key={widget.key}
                      type="button"
                      disabled={remaining <= 0}
                      onClick={() => onAdd(widget)}
                      className={cn(
                        "group border-border bg-background flex flex-col items-start gap-1 rounded-md border p-2.5 text-left transition-colors",
                        remaining > 0
                          ? "hover:border-foreground/20 hover:bg-muted/60"
                          : "cursor-not-allowed opacity-50",
                      )}
                    >
                      <span className="flex w-full items-center gap-1.5">
                        <span className="min-w-0 flex-1 truncate text-xs font-medium">
                          {widget.label}
                        </span>
                        {usedKeys.has(widget.key) && (
                          <span className="bg-muted text-muted-foreground shrink-0 rounded px-1 text-[9px]">
                            on canvas
                          </span>
                        )}
                        <PlusIcon className="text-muted-foreground size-3 shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
                      </span>
                      <span className="text-muted-foreground text-[10px] leading-snug">
                        {widget.description}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
