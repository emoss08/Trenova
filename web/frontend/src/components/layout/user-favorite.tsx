import { useUserFavorites } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import { useUserStore } from "@/stores/AuthStore";
import { useBreadcrumbStore } from "@/stores/BreadcrumbStore";
import { UserFavorite } from "@/types/accounts";
import { StarFilledIcon, StarIcon } from "@radix-ui/react-icons";
import { QueryClient, useQueryClient } from "@tanstack/react-query";
import { AxiosRequestConfig, AxiosResponse } from "axios";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";

async function manageFavoriteRequest(
  axiosConfig: AxiosRequestConfig,
  queryClient: QueryClient,
): Promise<AxiosResponse> {
  try {
    const response = await axios(axiosConfig);
    queryClient.invalidateQueries({ queryKey: ["userFavorites"] });
    return response.data;
  } catch (error) {
    console.error("[Trenova] Failed to manage favorite:", error);
    throw error;
  }
}

export function FavoriteIcon() {
  const queryClient = useQueryClient();
  const [currentRoute] = useBreadcrumbStore.use("currentRoute");
  const {
    data: userFavorites,
    isLoading: isFavoritesLoading,
    isError: isFavoritesError,
  } = useUserFavorites();

  async function manageFavorite(
    action: "add" | "remove",
    pageId: string,
    queryClient: QueryClient,
  ) {
    const user = useUserStore.get("user");

    const method = action === "add" ? "post" : "delete";
    const data = { pageLink: pageId, userId: user.id };

    const axiosConfig: AxiosRequestConfig = {
      method,
      url: "/user-favorites/",
      data,
    };

    toast.promise(manageFavoriteRequest(axiosConfig, queryClient), {
      loading: "Updating favorites...",
      success: "Favorites updated!",
      error: "Error updating favorites.",
    });
  }

  // Add and Remove Favorite Functions
  const addFavorite = useCallback(
    async (pageId: string) => {
      await manageFavorite("add", pageId, queryClient);
    },
    [queryClient],
  );

  const removeFavorite = useCallback(
    async (pageId: string) => {
      await manageFavorite("remove", pageId, queryClient);
    },
    [queryClient],
  );

  // Check if current page is favorite
  const isFavorite = useMemo(() => {
    const favoritesArray = Array.isArray(userFavorites) ? userFavorites : [];
    if (!currentRoute || favoritesArray.length === 0) return false;
    return favoritesArray.some(
      (favorite: UserFavorite) => favorite.pageLink === currentRoute.path,
    );
  }, [currentRoute, userFavorites]);

  const handleFavoriteToggle = useCallback(
    (isFavorite: boolean) => {
      const pageId = currentRoute?.path;
      if (pageId) {
        if (isFavorite) {
          addFavorite(pageId);
        } else {
          removeFavorite(pageId);
        }
      }
    },
    [addFavorite, removeFavorite, currentRoute],
  );

  const show = isFavoritesError || isFavoritesLoading || !currentRoute;

  if (show) {
    return null;
  }

  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          {isFavorite ? (
            <StarFilledIcon
              onClick={handleFavoriteToggle.bind(null, !isFavorite)}
              className="mx-2 mt-1 size-4 cursor-pointer text-center text-orange-400 transition-colors hover:text-orange-300"
            />
          ) : (
            <StarIcon
              onClick={handleFavoriteToggle.bind(null, !isFavorite)}
              className="mx-2 mt-1 size-4 cursor-pointer text-center text-orange-400 transition-colors hover:text-orange-300"
            />
          )}
        </TooltipTrigger>
        <TooltipContent side="right" sideOffset={10} className="font-normal">
          {isFavorite ? "Remove from favorites" : "Add to favorites"}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
