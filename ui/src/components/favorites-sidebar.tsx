import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuAction,
} from "@/components/ui/sidebar";
import { SHOW_FAVORITES_KEY } from "@/constants/env";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { queries } from "@/lib/queries";
import { api } from "@/services/api";
import { faStar } from "@fortawesome/pro-regular-svg-icons";
import { ChevronRightIcon } from "@radix-ui/react-icons";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Loader2 } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { useLocation } from "react-router";
import { PageFavoriteCard } from "./page-favorite/page-favorite-card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "./ui/collapsible";
import { Icon } from "./ui/icons";
import { VirtualCompatibleScrollArea } from "./ui/virtual-scroll-area";

export function FavoritesSidebar() {
  const location = useLocation();
  const [showFavorites, setShowFavorites] = useLocalStorage(
    SHOW_FAVORITES_KEY,
    true,
  );
  const query = useInfiniteQuery({
    queryKey: [...queries.favorite.list._def],
    queryFn: async ({ pageParam }) => {
      return await api.favorites.list({
        limit: 5,
        offset: pageParam,
      });
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.next || lastPage.results.length === 5) {
        return lastPageParam + 5;
      }
      return undefined;
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;
  const allFavorite = useMemo(
    () => query.data?.pages.flatMap((page) => page.results) ?? [],
    [query.data?.pages],
  );

  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const observerTarget = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: allFavorite.length,
    getScrollElement: () => scrollAreaRef.current,
    estimateSize: () => 40,
    overscan: 5,
  });

  const virtualItems = virtualizer.getVirtualItems();
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const extractPath = (url: string) => {
    try {
      const urlObj = new URL(url);
      return urlObj.pathname + urlObj.search;
    } catch {
      return url;
    }
  };

  if (query.isLoading) {
    return (
      <SidebarGroup>
        <SidebarGroupLabel className="select-none font-semibold uppercase">
          <Icon icon={faStar} className="mr-2" />
          Favorites
        </SidebarGroupLabel>
        <SidebarGroupContent>
          <div className="px-2 py-1 text-sm text-muted-foreground">
            Loading...
          </div>
        </SidebarGroupContent>
      </SidebarGroup>
    );
  }

  if (!query.isLoading && allFavorite.length === 0) {
    return (
      <SidebarGroup>
        <SidebarGroupLabel className="select-none font-semibold uppercase">
          <Icon icon={faStar} className="mr-2" />
          Favorites
        </SidebarGroupLabel>
        <SidebarGroupContent>
          <div className="px-2 py-1 text-sm text-muted-foreground">
            No favorites yet. Click the star icon on any page to add it to your
            favorites.
          </div>
        </SidebarGroupContent>
      </SidebarGroup>
    );
  }

  return (
    <SidebarGroup>
      <Collapsible open={showFavorites} onOpenChange={setShowFavorites}>
        <SidebarGroupLabel className="flex justify-between items-center select-none font-semibold uppercase">
          <div className="flex items-center justify-center gap-0.5">
            <Icon icon={faStar} className="mr-1 mb-0.5" />
            Favorites
          </div>
          <CollapsibleTrigger asChild>
            <SidebarMenuAction className="data-[state=open]:rotate-90 mt-0.5 static right-0 top-0">
              <ChevronRightIcon className="size-4" />
            </SidebarMenuAction>
          </CollapsibleTrigger>
        </SidebarGroupLabel>

        <CollapsibleContent>
          <SidebarGroupContent>
            <SidebarMenu>
              {!query.isLoading && allFavorite.length > 0 && (
                <VirtualCompatibleScrollArea
                  viewportRef={scrollAreaRef}
                  className="border border-border rounded-md flex-1"
                  viewportClassName="py-2 px-3 h-[150px]"
                >
                  {query.isLoading && (
                    <div className="flex items-center justify-center h-[250px]">
                      <Loader2 className="size-5 animate-spin text-muted-foreground" />
                    </div>
                  )}
                  {allFavorite.length > 0 && (
                    <div
                      style={{
                        height: `${virtualizer.getTotalSize()}px`,
                        width: "100%",
                        position: "relative",
                      }}
                    >
                      {virtualItems.map((virtualItem) => {
                        const favorite = allFavorite[virtualItem.index];
                        const favoritePath = extractPath(favorite.pageUrl);
                        const isActive = location.pathname === favoritePath;

                        return (
                          <div
                            key={virtualItem.key}
                            style={{
                              position: "absolute",
                              top: 0,
                              left: 0,
                              width: "100%",
                              height: `${virtualItem.size}px`,
                              transform: `translateY(${virtualItem.start}px)`,
                            }}
                          >
                            <div className="pb-1">
                              <PageFavoriteCard
                                pageFavorite={favorite}
                                isActive={isActive}
                                favoritePath={favoritePath}
                              />
                            </div>
                          </div>
                        );
                      })}
                      {isFetchingNextPage && (
                        <div
                          style={{
                            position: "absolute",
                            top: `${virtualizer.getTotalSize()}px`,
                            left: 0,
                            width: "100%",
                          }}
                        >
                          <Loader2 className="size-4 animate-spin text-muted-foreground" />
                        </div>
                      )}
                      <div
                        ref={observerTarget}
                        style={{
                          position: "absolute",
                          top: `${virtualizer.getTotalSize()}px`,
                          left: 0,
                          width: "100%",
                          height: "1px",
                        }}
                      />
                    </div>
                  )}
                </VirtualCompatibleScrollArea>
              )}
            </SidebarMenu>
          </SidebarGroupContent>
        </CollapsibleContent>
      </Collapsible>
    </SidebarGroup>
  );
}
