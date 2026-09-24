import { EmptyState } from "@/components/empty-state";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import type { AgentExtensionCatalogItem } from "@/types/agent-extension";
import { useQuery } from "@tanstack/react-query";
import { Input } from "@trenova/shared/components/ui/input";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { GlobeIcon, PuzzleIcon, SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { ExtensionCard } from "./extension-card";
import {
  ALL_CATEGORIES,
  extensionState,
  filterExtensions,
  sortExtensions,
  type ExtensionSort,
  type ExtensionStatusFilter,
} from "./extension-roster";
import { ExtensionSettingsDialog } from "./extension-settings-dialog";

const SORT_LABELS: Record<ExtensionSort, string> = {
  featured: "Featured",
  name: "Name",
  newest: "Newest",
};

/**
 * The extension marketplace: capabilities an organization turns on for its
 * agents only, each with its own credentials. An extension that is on adds
 * its tools to every agent or to the agents an administrator picks.
 */
export default function ExtensionsTab() {
  const t = useT();
  const catalogQuery = useQuery(queries.agentExtension.catalog());
  const { allowed: canUpdate } = usePermission(Resource.AgentExtension, Operation.Update);

  const [query, setQuery] = useState("");
  const [category, setCategory] = useState<string>(ALL_CATEGORIES);
  const [status, setStatus] = useState<ExtensionStatusFilter>("all");
  const [sort, setSort] = useState<ExtensionSort>("featured");
  const [openType, setOpenType] = useState<string | null>(null);

  const items = useMemo(() => catalogQuery.data?.items ?? [], [catalogQuery.data]);
  const visible = useMemo(
    () => sortExtensions(filterExtensions(items, { query, category, status }), sort),
    [items, query, category, status, sort],
  );
  const onCount = items.filter((item) => extensionState(item) === "on").length;
  const selected = items.find((item) => item.type === openType) ?? null;

  const categoryItems = useMemo(
    () => [
      { value: ALL_CATEGORIES, label: t("All categories") },
      ...(catalogQuery.data?.categories ?? []).map((option) => ({
        value: option.value,
        label: t(option.label),
      })),
    ],
    [catalogQuery.data, t],
  );
  const sortItems = useMemo(
    () =>
      (Object.keys(SORT_LABELS) as ExtensionSort[]).map((value) => ({
        value,
        label: t(SORT_LABELS[value]),
      })),
    [t],
  );
  const statusItems = [
    { value: "all" as const, label: t("All") },
    { value: "on" as const, label: t("On") },
    { value: "off" as const, label: t("Off") },
  ];

  const open = (extension: AgentExtensionCatalogItem) => setOpenType(extension.type);

  return (
    <section className="flex flex-col gap-4" aria-label={t("Extensions")}>
      <p className="text-muted-foreground px-1 text-xs">
        {t(
          "Extensions give agents abilities that work only inside AI features, using your organization's own account with the vendor.",
        )}
      </p>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          inputContainerClassName="w-full sm:max-w-xs"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("Search extensions")}
          className="h-8"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          aria-label={t("Search extensions")}
        />
        <Select
          items={categoryItems}
          value={category}
          onValueChange={(value) => setCategory(value ?? ALL_CATEGORIES)}
        >
          <SelectTrigger className="h-8 text-xs" aria-label={t("Category")}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {categoryItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Select
          items={sortItems}
          value={sort}
          onValueChange={(value) => setSort((value as ExtensionSort | null) ?? "featured")}
        >
          <SelectTrigger className="h-8 text-xs" aria-label={t("Sort by")}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {sortItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <SegmentedControl
          items={statusItems}
          value={status}
          onValueChange={setStatus}
          aria-label={t("Status")}
        />
        <span className="text-muted-foreground ml-auto text-xs tabular-nums">
          {t("{0} of {1} on", onCount, items.length)}
        </span>
      </div>

      {catalogQuery.isLoading ? (
        <div className="grid gap-3 md:grid-cols-2">
          {Array.from({ length: 2 }).map((_, index) => (
            <Skeleton key={index} className="h-56" />
          ))}
        </div>
      ) : catalogQuery.isError ? (
        <p className="text-danger px-1 py-6 text-center text-sm">
          {t("The extensions could not be loaded. Reload the page to try again.")}
        </p>
      ) : items.length === 0 ? (
        <div className="flex justify-center py-6">
          <EmptyState
            icons={[PuzzleIcon, GlobeIcon]}
            title={t("No extensions available")}
            description={t("This installation does not offer any extensions yet.")}
          />
        </div>
      ) : visible.length === 0 ? (
        <p className="text-muted-foreground px-1 py-6 text-center text-sm">
          {t("No extensions match these filters.")}
        </p>
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {visible.map((extension) => (
            <ExtensionCard
              key={extension.type}
              extension={extension}
              canUpdate={canUpdate}
              onOpen={open}
            />
          ))}
        </div>
      )}

      <ExtensionSettingsDialog
        extension={selected}
        open={selected !== null}
        onOpenChange={(next) => !next && setOpenType(null)}
        canUpdate={canUpdate}
      />
    </section>
  );
}
