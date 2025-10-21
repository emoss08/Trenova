/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { queries } from "@/lib/queries";
import { getPageTitle } from "@/lib/route-utils";
import { ToggleFavoriteSchema } from "@/lib/schemas/favorite-schema";
import { api } from "@/services/api";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useLocation } from "react-router";
import { toast } from "sonner";
import { broadcastQueryInvalidation } from "./use-invalidate-query";

export function useToggleFavorite() {
  return useMutation({
    mutationFn: (request: ToggleFavoriteSchema) =>
      api.favorites.toggle(request),
    onSuccess: async (data) => {
      await broadcastQueryInvalidation({
        queryKey: ["favorite"],
        options: {
          correlationId: `toggle-favorite-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      // Show success toast
      if (data.action === "added") {
        toast.success("Page added to favorites");
      } else {
        toast.success("Page removed from favorites");
      }
    },
    onError: (error) => {
      console.error("Failed to toggle favorite:", error);
      toast.error("Failed to update favorite");
    },
  });
}

export function useCurrentPageFavorite() {
  const location = useLocation();
  const currentUrl = `${window.location.origin}${location.pathname}`;

  return useQuery({
    ...queries.favorite.check(currentUrl),
  });
}

export function useToggleCurrentPageFavorite(options?: {
  pageTitle?: string;
  pageSection?: string;
  icon?: string;
  description?: string;
}) {
  const location = useLocation();
  const toggleFavorite = useToggleFavorite();

  const toggleCurrentPage = () => {
    const currentUrl = `${window.location.origin}${location.pathname}`;

    // Generate page title from route configuration or pathname if not provided
    const defaultTitle = options?.pageTitle || getPageTitle(location.pathname);

    toggleFavorite.mutate({
      pageUrl: currentUrl,
      pageTitle: defaultTitle,
      pageSection: options?.pageSection,
      icon: options?.icon,
      description: options?.description,
    });
  };

  return {
    toggle: toggleCurrentPage,
    ...toggleFavorite,
  };
}
