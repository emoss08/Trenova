import { useT } from "@trenova/shared/i18n/use-t";
import { useSidebarFold } from "@/components/navigation/sidebar-fold";
import { useBreadcrumbs } from "@/hooks/use-breadcrumb";
import { useHistoryNavigation } from "@/hooks/use-history-navigation";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { pageFavorite } from "@/lib/queries/page-favorite";
import { getPageTitle } from "@/lib/route-utils";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import type { ToggleFavoriteRequest } from "@/types/page-favorite";
import { useQuery } from "@tanstack/react-query";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  LayoutLeftIcon,
  Star01Icon,
} from "@trenova/shared/components/icons";
import { useLocation } from "react-router";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";

export function HistoryNavigation() {
  const t = useT();

  const { canGoBack, canGoForward, goBack, goForward } = useHistoryNavigation();

  return (
    <div className="flex items-center gap-1">
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              onClick={goBack}
              disabled={!canGoBack}
              aria-label={t("Go back")}
            >
              <ChevronLeftIcon className="size-3.5" />
            </Button>
          }
        />
        <TooltipContent>{t("Go back")}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              onClick={goForward}
              disabled={!canGoForward}
              aria-label={t("Go forward")}
            >
              <ChevronRightIcon className="size-3.5" />
            </Button>
          }
        />
        <TooltipContent>{t("Go forward")}</TooltipContent>
      </Tooltip>
    </div>
  );
}

export function FavoriteToggle({ className }: { className?: string }) {
  const t = useT();

  const location = useLocation();
  const breadcrumbs = useBreadcrumbs();
  const pageUrl = location.pathname;
  const lastCrumb = breadcrumbs.at(-1)?.crumb;
  const pageTitle = typeof lastCrumb === "string" ? lastCrumb : getPageTitle(pageUrl);

  const { data, isLoading } = useQuery({
    ...pageFavorite.check(pageUrl),
    enabled: !!pageUrl,
  });

  const isFavorited = data?.favorited;

  const { mutateAsync, isPending } = useOptimisticMutation({
    queryKey: pageFavorite.check(pageUrl).queryKey,
    mutationFn: async (values: ToggleFavoriteRequest) =>
      apiService.pageFavoriteService.togglePageFavorite(values),
    resourceName: "Page Favorite",
    invalidateQueries: [pageFavorite.all._def, pageFavorite.check._def],
    optimisticUpdate: (_variables, currentData) => !currentData,
  });

  const handleToggle = () => {
    void mutateAsync({ pageUrl, pageTitle });
  };

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            className={cn("cursor-pointer", className)}
            size="xs"
            onClick={handleToggle}
            disabled={isLoading || isPending}
            aria-label={isFavorited ? "Remove from favorites" : "Add to favorites"}
          />
        }
      >
        <Star01Icon
          className={cn(
            "size-3 transition-colors",
            isFavorited && "fill-warning text-warning-foreground",
          )}
        />
      </TooltipTrigger>
      <TooltipContent>
        {isFavorited ? t("Remove from favorites") : t("Add to favorites")}
      </TooltipContent>
    </Tooltip>
  );
}

export function SidebarToggle({ className }: { className?: string }) {
  const { shown, toggle } = useSidebarFold();
  const label = shown ? "Hide sidebar" : "Show sidebar";

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={toggle}
            aria-label={label}
            aria-pressed={shown}
            className={className}
          />
        }
      >
        <LayoutLeftIcon className="size-3.5" />
      </TooltipTrigger>
      <TooltipContent>
        {label} ({formatShortcut("B")})
      </TooltipContent>
    </Tooltip>
  );
}
