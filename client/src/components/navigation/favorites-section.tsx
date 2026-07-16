import { SidebarNavLink, SidebarSectionLabel } from "@/components/navigation/sidebar-primitives";
import { ScrollArea } from "@/components/ui/scroll-area";
import { queries } from "@/lib/queries";
import { isRouteActive } from "@/lib/route-utils";
import { useQuery } from "@tanstack/react-query";
import { Star } from "lucide-react";
import { useLocation } from "react-router";

export function FavoritesSection() {
  const { pathname } = useLocation();
  const { data: favorites } = useQuery(queries.pageFavorite.all());

  if (!favorites || favorites.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-0.5">
      <SidebarSectionLabel>Favorites</SidebarSectionLabel>
      <ScrollArea viewportClassName="max-h-20" maskHeight={12} maskVariant="sidebar">
        <div className="flex w-full flex-col gap-0.5 pr-2.5 pl-2">
          {favorites.map((favorite) => (
            <SidebarNavLink
              key={favorite.id}
              to={favorite.pageUrl}
              active={isRouteActive(pathname, favorite.pageUrl)}
            >
              <Star className="size-3 shrink-0 fill-amber-400 text-amber-400" />
              <span className="truncate">{favorite.pageTitle}</span>
            </SidebarNavLink>
          ))}
        </div>
      </ScrollArea>
    </div>
  );
}
