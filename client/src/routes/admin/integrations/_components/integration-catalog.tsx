import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useMemo, useState } from "react";
import { searchParamsParser } from "../integration-marketplace-state";
import { GoogleIntegrationModal } from "./google/google-integration-modal";
import { IntegrationMarketplaceHeader } from "./integration-marketplace-header";
import { OpenAIIntegrationModal } from "./openai/openai-integration-modal";
import { SamsaraIntegrationModal } from "./samsara/samsara-integration-modal";

function getProviderMonogram(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) {
    return "IN";
  }

  const words = trimmed.split(/\s+/).filter(Boolean);
  if (words.length === 1) {
    return words[0].slice(0, 2).toUpperCase();
  }

  return `${words[0][0] ?? ""}${words[1][0] ?? ""}`.toUpperCase();
}

const sortOptions = [
  { label: "Name (A-Z)", value: "name_asc" },
  { label: "Name (Z-A)", value: "name_desc" },
];

const statusOptions = [
  { label: "All Statuses", value: "all" },
  { label: "Connected", value: "connected" },
  { label: "Disconnected", value: "disconnected" },
];

export function IntegrationCatalogCard() {
  const { theme } = useTheme();
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const [isSamsaraModalOpen, setIsSamsaraModalOpen] = useState(false);
  const [isGoogleModalOpen, setIsGoogleModalOpen] = useState(false);
  const [isOpenAIModalOpen, setIsOpenAIModalOpen] = useState(false);

  const catalogQuery = useQuery({
    ...queries.integration.catalog(),
  });

  const items = useMemo(
    () => catalogQuery.data?.items ?? [],
    [catalogQuery.data?.items],
  );

  const categoryOptions = useMemo(() => {
    const uniqueCategories = new Map<string, string>();
    for (const item of items) {
      if (!item.category) {
        continue;
      }
      if (!uniqueCategories.has(item.category)) {
        uniqueCategories.set(
          item.category,
          item.categoryLabel || item.category,
        );
      }
    }

    return [
      { label: "All Categories", value: "all" },
      ...Array.from(uniqueCategories.entries())
        .map(([value, label]) => ({ value, label }))
        .sort((left, right) => left.label.localeCompare(right.label)),
    ];
  }, [items]);

  const filteredItems = useMemo(() => {
    const normalizedSearch = searchParams.query.trim().toLowerCase();

    return items.filter((item) => {
      if (searchParams.status === "connected" && !item.enabled) {
        return false;
      }
      if (searchParams.status === "disconnected" && item.enabled) {
        return false;
      }
      if (
        searchParams.category !== "all" &&
        item.category !== searchParams.category
      ) {
        return false;
      }

      if (!normalizedSearch) {
        return true;
      }

      return (
        item.name.toLowerCase().includes(normalizedSearch) ||
        item.description.toLowerCase().includes(normalizedSearch) ||
        item.categoryLabel.toLowerCase().includes(normalizedSearch)
      );
    });
  }, [items, searchParams.status, searchParams.category, searchParams.query]);

  const filteredAndSortedItems = useMemo(() => {
    const sortedItems = [...filteredItems];
    sortedItems.sort((left, right) => {
      const comparison = left.name.localeCompare(right.name);
      return searchParams.sortBy === "name_asc" ? comparison : -comparison;
    });
    return sortedItems;
  }, [filteredItems, searchParams.sortBy]);

  const openModal = (type: string) => {
    switch (type) {
      case "Samsara":
        setIsSamsaraModalOpen(true);
        break;
      case "GoogleMaps":
        setIsGoogleModalOpen(true);
        break;
      case "OpenAI":
        setIsOpenAIModalOpen(true);
        break;
      default:
        break;
    }
  };

  const hasModal = (type: string) =>
    type === "Samsara" || type === "GoogleMaps" || type === "OpenAI";

  return (
    <>
      <section className="relative overflow-hidden rounded-xl border border-border bg-background">
        <div className="relative border-b border-border/80 bg-sidebar px-5 py-5 sm:px-6">
          <IntegrationMarketplaceHeader />
          <div className="mt-4 flex flex-row items-center gap-1.5">
            <div className="flex shrink-0 flex-row items-center gap-0 text-center text-sm">
              <div className="flex h-8 items-center gap-1 rounded-s-lg rounded-e-none border border-r-0 border-input bg-muted px-1 font-medium text-muted-foreground focus:z-10">
                Sort By
              </div>
              <Select
                items={sortOptions}
                value={searchParams.sortBy}
                onValueChange={(value) => setSearchParams({ sortBy: value })}
              >
                <SelectTrigger className="h-9 rounded-s-none rounded-e-lg bg-background text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {sortOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <div className="shrink-0">
              <Select
                items={categoryOptions}
                value={searchParams.category}
                onValueChange={(value) => setSearchParams({ category: value })}
              >
                <SelectTrigger className="h-9 bg-background text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {categoryOptions.map((category) => (
                      <SelectItem key={category.value} value={category.value}>
                        {category.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <div className="shrink-0">
              <Select
                items={statusOptions}
                value={searchParams.status}
                onValueChange={(value) => setSearchParams({ status: value })}
              >
                <SelectTrigger className="h-9 bg-background text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {statusOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
        <div className="relative space-y-4 p-5 sm:p-6">
          {catalogQuery.isLoading && (
            <div className="inline-flex items-center gap-2 text-sm text-muted-foreground">
              <Spinner className="size-4" />
              Loading integration catalog...
            </div>
          )}
          {!catalogQuery.isLoading && filteredAndSortedItems.length === 0 && (
            <div className="rounded-md border border-border bg-muted/20 p-4 text-sm text-muted-foreground">
              No integrations match your current search and filter.
            </div>
          )}
          {!catalogQuery.isLoading && filteredAndSortedItems.length > 0 && (
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {filteredAndSortedItems.map((item) => {
                const canConfigure = hasModal(item.type);
                const logoURL =
                  theme === "dark"
                    ? item.logoDarkUrl || item.logoLightUrl || item.logoUrl
                    : item.logoLightUrl || item.logoDarkUrl || item.logoUrl;

                return (
                  <Card
                    key={item.type}
                    className="group relative overflow-hidden border-border/80 bg-linear-to-b from-background to-muted/10 transition-all"
                  >
                    <CardHeader className="space-y-2 pb-3">
                      <div className="relative flex items-start justify-between gap-3">
                        <div className="space-y-1 pr-14">
                          <CardTitle className="text-base">
                            {item.name}
                          </CardTitle>
                          <CardDescription className="text-xs">
                            <div className="flex items-center gap-3 text-xs text-muted-foreground">
                              {item.links.map((link) => (
                                <ExternalLink
                                  key={`${item.type}-${link.kind}-${link.url}`}
                                  href={link.url}
                                  className="inline-flex items-center gap-1 hover:text-foreground"
                                >
                                  {link.label}
                                </ExternalLink>
                              ))}
                            </div>
                          </CardDescription>
                        </div>
                        <div className="absolute -top-5 -right-10 inline-flex size-20">
                          {logoURL ? (
                            <LazyImage
                              src={logoURL}
                              alt={`${item.name} logo`}
                              className="size-24 object-contain"
                            />
                          ) : (
                            <span className="text-xs font-semibold text-foreground/80">
                              {getProviderMonogram(item.name)}
                            </span>
                          )}
                        </div>
                      </div>
                      <CatalogItemDescription description={item.description} />
                    </CardHeader>
                    <CardContent className="space-y-3 pt-0">
                      <div className="flex items-center justify-between gap-2 border-t border-border/80 pt-3">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => openModal(item.type)}
                          disabled={!canConfigure}
                        >
                          {item.primaryActionLabel}
                        </Button>
                        <div className="flex items-center gap-2">
                          <Switch
                            checked={item.enabled}
                            aria-label={`${item.name} integration enabled`}
                            onCheckedChange={() => openModal(item.type)}
                            disabled={!canConfigure}
                          />
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </div>
      </section>
      <SamsaraIntegrationModal
        open={isSamsaraModalOpen}
        onOpenChange={setIsSamsaraModalOpen}
      />
      <GoogleIntegrationModal
        open={isGoogleModalOpen}
        onOpenChange={setIsGoogleModalOpen}
      />
      <OpenAIIntegrationModal
        open={isOpenAIModalOpen}
        onOpenChange={setIsOpenAIModalOpen}
      />
    </>
  );
}

export function CatalogItemDescription({
  description,
}: {
  description: string;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <p className="line-clamp-2 min-h-10 w-87.5 text-sm text-pretty text-muted-foreground">
            {description}
          </p>
        }
      />
      <TooltipContent className="max-w-sm">{description}</TooltipContent>
    </Tooltip>
  );
}
