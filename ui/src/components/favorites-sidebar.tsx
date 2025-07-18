import { Button } from "@/components/ui/button";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { SHOW_FAVORITES_KEY } from "@/constants/env";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { faStar, faTrash } from "@fortawesome/pro-regular-svg-icons";
import { ChevronRightIcon } from "@radix-ui/react-icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalStorage } from "@uidotdev/usehooks";
import { Link, useLocation } from "react-router";
import { toast } from "sonner";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "./ui/collapsible";
import { Icon } from "./ui/icons";
import { ScrollArea } from "./ui/scroll-area";

export function FavoritesSidebar() {
  const location = useLocation();
  const { data: favorites = [], isLoading } = useQuery({
    ...queries.favorite.list(),
  });
  const queryClient = useQueryClient();
  const [showFavorites, setShowFavorites] = useLocalStorage(
    SHOW_FAVORITES_KEY,
    true,
  );

  const deleteFavorite = useMutation({
    mutationFn: (favoriteId: string) => api.favorites.delete(favoriteId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queries.favorite.list._def });
      toast.success("Favorite removed");
    },
    onError: () => {
      toast.error("Failed to remove favorite");
    },
  });

  const handleRemoveFavorite = (e: React.MouseEvent, favoriteId: string) => {
    e.preventDefault();
    e.stopPropagation();
    deleteFavorite.mutate(favoriteId);
  };

  const extractPath = (url: string) => {
    try {
      const urlObj = new URL(url);
      return urlObj.pathname + urlObj.search;
    } catch {
      return url;
    }
  };

  if (isLoading) {
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

  if (favorites.length === 0) {
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
              <ScrollArea className="flex flex-col gap-4 overflow-y-auto max-h-[200px] p-3 border border-border bg-muted-foreground/5 rounded-md">
                {favorites.map((favorite) => {
                  const favoritePath = extractPath(favorite.pageUrl);
                  const isActive = location.pathname === favoritePath;

                  return (
                    <SidebarMenuItem key={favorite.id}>
                      <SidebarMenuButton
                        asChild
                        isActive={isActive}
                        className="group pr-8"
                      >
                        <Link
                          to={favoritePath}
                          className="flex items-center gap-2 text-sm"
                          title={favorite.description || favorite.pageTitle}
                        >
                          {favorite.icon && (
                            <Icon
                              icon={favorite.icon as any}
                              className="size-4"
                            />
                          )}
                          <div className="flex flex-col gap-0.5 min-w-0 flex-1">
                            <span className="truncate font-medium">
                              {favorite.pageTitle}
                            </span>
                            {favorite.pageSection && (
                              <span className="truncate text-xs text-muted-foreground">
                                {favorite.pageSection}
                              </span>
                            )}
                          </div>
                        </Link>
                      </SidebarMenuButton>
                      <Button
                        variant="ghost"
                        size="icon"
                        className={cn(
                          "absolute right-1 top-1/2 -translate-y-1/2 size-6",
                          "opacity-0 group-hover:opacity-100 transition-opacity",
                          "hover:bg-destructive/10 hover:text-destructive",
                        )}
                        onClick={(e) => handleRemoveFavorite(e, favorite.id)}
                        disabled={deleteFavorite.isPending}
                        title="Remove from favorites"
                      >
                        <Icon icon={faTrash} className="size-3" />
                      </Button>
                    </SidebarMenuItem>
                  );
                })}
              </ScrollArea>
            </SidebarMenu>
          </SidebarGroupContent>
        </CollapsibleContent>
      </Collapsible>
    </SidebarGroup>
  );
}
