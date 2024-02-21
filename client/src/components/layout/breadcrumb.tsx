/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { useUserFavorites } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import { TOAST_STYLE } from "@/lib/constants";
import { upperFirst } from "@/lib/utils";
import { routes } from "@/routing/AppRoutes";
import { useBreadcrumbStore } from "@/stores/BreadcrumbStore";
import { UserFavorite } from "@/types/accounts";
import { faStar } from "@fortawesome/pro-regular-svg-icons";
import { faStar as faStarFilled } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { QueryClient, useQueryClient } from "@tanstack/react-query";
import { AxiosRequestConfig, AxiosResponse } from "axios";
import { pathToRegexp } from "path-to-regexp";
import { useCallback, useEffect, useMemo } from "react";
import toast from "react-hot-toast";
import { useLocation } from "react-router-dom";
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
): Promise<AxiosResponse> {
  const endpoint = action === "add" ? "/favorites/" : "/favorites/delete/";
  const method = action === "add" ? "post" : "delete";
  const data = { page: pageId };

  const axiosConfig: AxiosRequestConfig = {
    method,
    url: endpoint,
    data,
  };

  return toast.promise(
    manageFavoriteRequest(axiosConfig, queryClient),
    {
      loading: "Updating favorites...",
      success: "Favorites updated!",
      error: "Error updating favorites.",
    },
    {
      id: "theme-switcher",
      style: TOAST_STYLE,
      ariaProps: {
        role: "status",
        "aria-live": "polite",
      },
    },
  );
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
          <FontAwesomeIcon
            icon={isFavorite ? faStarFilled : faStar}
            title="Favorite"
            className="mx-1.5 mb-0.5 size-4 cursor-pointer text-center text-orange-400 transition-colors hover:text-orange-300"
            onClick={onFavoriteToggle.bind(null, !isFavorite)}
          />
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
      (favorite: UserFavorite) => favorite.page === currentRoute.path,
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
        <h2 className="mt-10 scroll-m-20 pb-2 text-xl font-semibold tracking-tight transition-colors first:mt-0">
          {currentRoute.title}
          <FavoriteIcon
            isFavorite={isFavorite}
            isFavoriteLoading={isFavoritesLoading}
            isFavoriteError={isFavoritesError}
            onFavoriteToggle={handleFavoriteToggle}
          />
        </h2>
        <div className="flex items-center">
          <a className="text-sm font-medium text-muted-foreground hover:text-muted-foreground/80">
            {breadcrumbText}
          </a>
        </div>
      </div>
    </div>
  );
}
