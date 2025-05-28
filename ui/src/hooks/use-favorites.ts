import { queries } from "@/lib/queries";
import { getPageTitle } from "@/lib/route-utils";
import { api } from "@/services/api";
import type { ToggleFavoriteRequest } from "@/types/favorite";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocation } from "react-router";
import { toast } from "sonner";

export function useToggleFavorite() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: ToggleFavoriteRequest) =>
      api.favorites.toggle(request),
    onSuccess: (data) => {
      // Invalidate and refetch favorites
      queryClient.invalidateQueries({ queryKey: queries.favorite.list._def });
      queryClient.invalidateQueries({
        queryKey: queries.favorite.check._def,
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
  const currentUrl = `${window.location.origin}${location.pathname}${location.search}`;

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
    const currentUrl = `${window.location.origin}${location.pathname}${location.search}`;

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
