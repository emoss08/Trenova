import { useUserFavorites } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import { upperFirst } from "@/lib/utils";
import { routes } from "@/routing/AppRoutes";
import { useUserStore } from "@/stores/AuthStore";
import { useBreadcrumbStore } from "@/stores/BreadcrumbStore";
import { UserFavorite } from "@/types/accounts";
import { StarFilledIcon, StarIcon } from "@radix-ui/react-icons";
import { QueryClient, useQueryClient } from "@tanstack/react-query";
import { AxiosRequestConfig, AxiosResponse } from "axios";
import { pathToRegexp } from "path-to-regexp";
import { useCallback, useEffect, useMemo } from "react";
import { useLocation } from "react-router-dom";
import { toast } from "sonner";
import { Skeleton } from "../ui/skeleton";
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

const useRouteMatching = (
  setLoading: (loading: boolean) => void,
  setCurrentRoute: (route: any) => void,
) => {
  const location = useLocation();

  useEffect(() => {
    setLoading(true);
    const excludedPath = "/shipment-management/";
    const matchedRoute =
      location.pathname !== excludedPath &&
      routes.find((route) => {
        return (
          route.path !== "*" &&
          route.path !== excludedPath &&
          pathToRegexp(route.path).test(location.pathname)
        );
      });
    setCurrentRoute(matchedRoute || null);
    setLoading(false);
  }, [location, setLoading, setCurrentRoute]);
};

const useDocumentTitle = (currentRoute: any) => {
  useEffect(() => {
    if (currentRoute) {
      document.title = currentRoute.title;
    }
  }, [currentRoute]);
};

function FavoriteIcon({
  isFavorite,
  isFavoriteLoading,
  isFavoriteError,
  onFavoriteToggle,
}: {
  isFavorite: boolean;
  isFavoriteLoading: boolean;
  isFavoriteError: boolean;
  onFavoriteToggle: (isFavorite: boolean) => void;
}) {
  if (isFavoriteError || isFavoriteLoading) {
    return null;
  }

  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          {isFavorite ? (
            <StarFilledIcon
              onClick={onFavoriteToggle.bind(null, !isFavorite)}
              className="mx-2 mt-1 size-4 cursor-pointer text-center text-orange-400 transition-colors hover:text-orange-300"
            />
          ) : (
            <StarIcon
              onClick={onFavoriteToggle.bind(null, !isFavorite)}
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

export function Breadcrumb() {
  const queryClient = useQueryClient();
  const [currentRoute, setCurrentRoute] =
    useBreadcrumbStore.use("currentRoute");
  const [loading, setLoading] = useBreadcrumbStore.use("loading");
  const {
    data: userFavorites,
    isLoading: isFavoritesLoading,
    isError: isFavoritesError,
  } = useUserFavorites();

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

  // Custom Hooks for functionality
  useRouteMatching(setLoading, setCurrentRoute);
  useDocumentTitle(currentRoute);

  // Construct breadcrumb text
  const breadcrumbText = useMemo(() => {
    if (!currentRoute) return "";
    const parts = [currentRoute.group, currentRoute.subMenu, currentRoute.title]
      .filter((str): str is string => str !== undefined)
      .map(upperFirst);
    return parts.join(" - ");
  }, [currentRoute]);

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

  if (loading) {
    return (
      <>
        <Skeleton className="h-[30px] w-[200px]" />
        <Skeleton className="mt-5 h-[30px] w-[200px]" />
      </>
    );
  }

  if (!currentRoute) {
    return null;
  }

  return (
    <div className="pb-4 pt-5 md:py-4">
      <div>
        <h2 className=" mt-10 flex scroll-m-20 items-center pb-2 text-xl font-semibold tracking-tight transition-colors first:mt-0">
          {currentRoute.title}
          <FavoriteIcon
            isFavorite={isFavorite}
            isFavoriteLoading={isFavoritesLoading}
            isFavoriteError={isFavoritesError}
            onFavoriteToggle={handleFavoriteToggle}
          />
        </h2>
        <div className="flex items-center">
          <a className="text-muted-foreground hover:text-muted-foreground/80 text-sm font-medium">
            {breadcrumbText}
          </a>
        </div>
      </div>
    </div>
  );
}
