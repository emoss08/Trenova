"use no memo";
import { useSearch } from "@/hooks/use-search";
import { cn } from "@/lib/utils";
import { SearchEntityType } from "@/types/search";
import { faXmark } from "@fortawesome/pro-regular-svg-icons";
import React, { useMemo, useState } from "react";
import { Badge } from "../ui/badge";
import { CommandDialog, CommandInput } from "../ui/command";
import { Icon } from "../ui/icons";
import { SiteSearchDialogContent } from "./_components/site-search-dialog-content";

export function SearchDialog() {
  const [open, setOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<SearchEntityType>(
    SearchEntityType.All,
  );
  const { searchQuery, setSearchQuery, searchResults, isLoading } =
    useSearch(activeTab);
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionText, setMentionText] = useState("");
  const [mentionIndex, setMentionIndex] = useState(0);

  const entityOptions = useMemo(
    () => [
      { key: SearchEntityType.All, label: "All", aliases: ["all", "a"] },
      {
        key: SearchEntityType.Shipment,
        label: "Shipments",
        aliases: ["s", "ship", "shipment", "shipments"],
      },
      {
        key: SearchEntityType.Customer,
        label: "Customers",
        aliases: ["c", "cust", "customer", "customers"],
      },
    ],
    [],
  );

  const filteredEntityOptions = useMemo(() => {
    // Exclude 'All' from suggestions; it's the implicit default
    const list = entityOptions.filter(
      (opt) => opt.key !== SearchEntityType.All,
    );
    const q = mentionText.toLowerCase();
    if (!q) return list;
    return list.filter(
      (opt) =>
        opt.label.toLowerCase().includes(q) ||
        opt.aliases.some((a) => a.startsWith(q)),
    );
  }, [mentionText, entityOptions]);

  // note: we reset mentionIndex when opening/updating mention via input handler

  const handleSelectEntity = (entity: SearchEntityType) => {
    setActiveTab(entity);
    const lastAtIndex = searchQuery.lastIndexOf("@");
    if (lastAtIndex >= 0) {
      const cleaned = searchQuery.slice(0, lastAtIndex).trimStart();
      setSearchQuery(cleaned);
    }
    setMentionOpen(false);
    setMentionText("");
  };

  const handleInputChange = (value: string) => {
    const lastAtIndex = value.lastIndexOf("@");
    if (lastAtIndex >= 0) {
      const after = value.slice(lastAtIndex + 1);
      const hasSpace = after.includes(" ");
      if (!hasSpace) {
        setMentionOpen(true);
        setMentionText(after);
        setMentionIndex(0);
      } else {
        setMentionOpen(false);
        setMentionText("");
      }
    } else {
      setMentionOpen(false);
      setMentionText("");
    }
    // If input cleared, drop the active filter badge
    if (value.length === 0 && activeTab !== SearchEntityType.All) {
      setActiveTab(SearchEntityType.All);
    }
    setSearchQuery(value);
  };

  const handleInputKeyDown: React.KeyboardEventHandler<HTMLInputElement> = (
    e,
  ) => {
    if (!mentionOpen || filteredEntityOptions.length === 0) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setMentionIndex((i) => {
        const len = filteredEntityOptions.length;
        return (i + 1) % len;
      });
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setMentionIndex((i) => {
        const len = filteredEntityOptions.length;
        return (i - 1 + len) % len;
      });
    } else if (e.key === "Enter") {
      e.preventDefault();
      const target = filteredEntityOptions[mentionIndex];
      if (target) handleSelectEntity(target.key);
    } else if (e.key === "Escape") {
      e.preventDefault();
      setMentionOpen(false);
    }
  };

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "j" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  return (
    <>
      <p className="text-muted-foreground text-sm">
        Press{" "}
        <kbd className="bg-muted text-muted-foreground pointer-events-none inline-flex h-5 items-center gap-1 rounded border px-1.5 font-mono text-[10px] font-medium opacity-100 select-none">
          <span className="text-xs">⌘</span>J
        </kbd>
      </p>
      <CommandDialog
        open={open}
        onOpenChange={setOpen}
        shouldFilter={false}
        contentClassName={cn(
          "z-50 grid w-full max-w-[calc(100%-2rem)] gap-4 border duration-200 sm:max-w-3xl",
          "rounded-xl border-none bg-clip-padding shadow-2xl ring-4",
          "ring-neutral-200/80 dark:bg-neutral-900 dark:ring-neutral-800",
        )}
      >
        <div className="relative">
          <CommandInput
            className={cn(
              activeTab !== SearchEntityType.All ? "pl-24" : undefined,
            )}
            placeholder={
              activeTab === "all"
                ? "Type @ to filter (e.g., @shipments, @customers) and search..."
                : activeTab === "shipment"
                  ? "Search shipments..."
                  : "Search customers..."
            }
            value={searchQuery}
            onValueChange={handleInputChange}
            onKeyDown={handleInputKeyDown}
          />
          {activeTab !== SearchEntityType.All && (
            <div className="group absolute left-8 top-1/2 -translate-y-1/2">
              <Badge
                withDot={false}
                variant="secondary"
                className="h-5 py-0 pl-2 pr-1.5 text-2xs flex items-center gap-1"
              >
                {activeTab === SearchEntityType.Shipment
                  ? "Shipments"
                  : "Customers"}
                <button
                  type="button"
                  aria-label="Clear entity filter"
                  onClick={() => setActiveTab(SearchEntityType.All)}
                  className="ml-0.5 rounded-sm opacity-60 transition-opacity hover:opacity-100 focus:opacity-100 focus:outline-hidden"
                >
                  <Icon icon={faXmark} className="size-3" />
                </button>
              </Badge>
            </div>
          )}
          {mentionOpen && (
            <div className="absolute left-2 right-2 top-11 z-50 rounded-md border border-border bg-popover shadow-md w-[200px] p-2">
              <div className="p-1 text-xs text-muted-foreground">
                Filter by entity
              </div>
              <div className="flex flex-col gap-0.5">
                {filteredEntityOptions.map((opt, idx) => (
                  <button
                    key={opt.key}
                    onClick={() => handleSelectEntity(opt.key)}
                    className={cn(
                      "w-full text-left px-3 py-1.5 text-sm rounded-md",
                      idx === mentionIndex ? "bg-muted" : "hover:bg-muted",
                    )}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
        <SiteSearchDialogContent
          searchQuery={searchQuery}
          isLoading={isLoading}
          setOpen={setOpen}
          setSearchQuery={setSearchQuery}
          searchResults={searchResults}
          activeTab={activeTab}
        />
      </CommandDialog>
    </>
  );
}
