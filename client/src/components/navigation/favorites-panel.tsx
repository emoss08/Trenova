import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeftIcon, Star } from "lucide-react";
import { Link, useLocation } from "react-router";

type FavoritesPanelProps = {
  collapsed: boolean;
  onToggleCollapse: () => void;
};

function isRouteActive(currentPath: string, itemPath: string): boolean {
  if (itemPath === "/") return currentPath === "/";
  const a = currentPath.endsWith("/") ? currentPath.slice(0, -1) : currentPath;
  const b = itemPath.endsWith("/") ? itemPath.slice(0, -1) : itemPath;
  return a.startsWith(b);
}

export function FavoritesPanel({ collapsed, onToggleCollapse }: FavoritesPanelProps) {
  const { pathname } = useLocation();
  const { data: favorites, isLoading } = useQuery(queries.pageFavorite.all());

  if (collapsed) {
    return <div className="w-0 overflow-hidden transition-all duration-200" />;
  }

  return (
    <div className="flex h-full w-[220px] flex-col border-r border-border bg-background transition-all duration-200">
      <div className="flex items-center justify-between px-3 py-3">
        <div className="flex items-center gap-1.5">
          <Star className="size-3.5 fill-amber-400 text-amber-400" />
          <h2 className="text-sm font-semibold truncate">Favorites</h2>
        </div>
        <button
          type="button"
          onClick={onToggleCollapse}
          className="flex size-6 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
        >
          <ChevronLeftIcon className="size-4" />
        </button>
      </div>

      <ScrollArea className="flex-1" maskHeight={20}>
        <div className="flex flex-col gap-0.5 px-2 pb-3">
          {isLoading ? (
            <>
              <Skeleton className="h-7 w-full rounded-md" />
              <Skeleton className="h-7 w-full rounded-md" />
              <Skeleton className="h-7 w-full rounded-md" />
            </>
          ) : favorites && favorites.length > 0 ? (
            favorites.map((fav) => {
              const active = isRouteActive(pathname, fav.pageUrl);
              return (
                <Link
                  key={fav.id}
                  to={fav.pageUrl}
                  className={cn(
                    "flex items-center gap-2 rounded-md px-2 h-7 text-[13px] transition-colors",
                    active
                      ? "bg-accent text-accent-foreground font-medium"
                      : "text-foreground/70 hover:bg-muted hover:text-foreground",
                  )}
                >
                  <span className="truncate">{fav.pageTitle}</span>
                </Link>
              );
            })
          ) : (
            <p className="px-2 py-4 text-xs text-muted-foreground">
              Star pages from the header to add them here.
            </p>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
