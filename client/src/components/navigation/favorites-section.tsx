import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSkeleton,
} from "@/components/ui/sidebar";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, Star } from "lucide-react";
import { useState } from "react";
import { Link, useLocation } from "react-router";

export function FavoritesSection() {
  const [isOpen, setIsOpen] = useState(true);
  const location = useLocation();
  const { data: favorites, isLoading } = useQuery(queries.pageFavorite.all());

  const hasFavorites = favorites && favorites.length > 0;

  if (!isLoading && !hasFavorites) {
    return null;
  }

  return (
    <Collapsible
      open={isOpen}
      onOpenChange={setIsOpen}
      className="group/collapsible"
    >
      <SidebarGroup>
        <SidebarGroupLabel
          render={
            <CollapsibleTrigger className="flex w-full cursor-pointer items-center gap-2" />
          }
        >
          <Star className="size-4 fill-amber-400 text-amber-400" />
          <span className="flex-1 text-left">Favorites</span>
          <ChevronRight
            className={cn(
              "size-4 transition-transform duration-200",
              isOpen && "rotate-90",
            )}
          />
        </SidebarGroupLabel>
        <CollapsibleContent>
          <SidebarGroupContent>
            <SidebarMenu className="flex flex-col gap-0.5">
              {isLoading ? (
                <>
                  <SidebarMenuSkeleton />
                  <SidebarMenuSkeleton />
                  <SidebarMenuSkeleton />
                </>
              ) : (
                favorites?.map((favorite) => {
                  const isActive = location.pathname === favorite.pageUrl;
                  return (
                    <SidebarMenuItem key={favorite.id}>
                      <SidebarMenuButton
                        render={<Link to={favorite.pageUrl} />}
                        isActive={isActive}
                      >
                        <span className="truncate">{favorite.pageTitle}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })
              )}
            </SidebarMenu>
          </SidebarGroupContent>
        </CollapsibleContent>
      </SidebarGroup>
    </Collapsible>
  );
}
