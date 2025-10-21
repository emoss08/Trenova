import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { FavoriteSchema } from "@/lib/schemas/favorite-schema";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { faTrash } from "@fortawesome/pro-regular-svg-icons";
import { useMutation } from "@tanstack/react-query";
import { Link } from "react-router";
import { toast } from "sonner";
import { Button } from "../ui/button";
import { Icon } from "../ui/icons";
import { SidebarMenuButton, SidebarMenuItem } from "../ui/sidebar";

export function PageFavoriteCard({
  pageFavorite,
  isActive,
  favoritePath,
}: {
  pageFavorite: FavoriteSchema;
  isActive: boolean;
  favoritePath: string;
}) {
  const deleteFavorite = useMutation({
    mutationFn: (favoriteId: string) => api.favorites.delete(favoriteId),
    onSuccess: async () => {
      await broadcastQueryInvalidation({
        queryKey: ["favorite"],
        options: {
          correlationId: `delete-favorite-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

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

  return (
    <SidebarMenuItem>
      <SidebarMenuButton asChild isActive={isActive}>
        <Link to={favoritePath} className="flex items-center gap-2 text-xs">
          <div className="flex flex-col gap-0.5 min-w-0 flex-1">
            <span className="truncate">{pageFavorite.pageTitle}</span>
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
        onClick={(e) => handleRemoveFavorite(e, pageFavorite.id!)}
        disabled={deleteFavorite.isPending}
        title="Remove from favorites"
      >
        <Icon icon={faTrash} className="size-3" />
      </Button>
    </SidebarMenuItem>
  );
}
