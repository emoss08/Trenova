import { NotificationPopover } from "@/components/notification-center/notification-popover";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { useBreadcrumbs } from "@/hooks/use-breadcrumb";
import { useHistoryNavigation } from "@/hooks/use-history-navigation";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { queries } from "@/lib/queries";
import { getPageTitle } from "@/lib/route-utils";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import { useNavigationStore } from "@/stores/navigation-store";
import type { ToggleFavoriteRequest } from "@/types/page-favorite";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, ChevronRight, PanelLeftIcon, Star } from "lucide-react";
import React from "react";
import { Link, useLocation, useNavigation } from "react-router";
import { toast } from "sonner";
import { Button } from "./ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./ui/tooltip";

export function Header() {
  const navigation = useNavigation();
  const breadcrumbs = useBreadcrumbs();
  const isLoading = navigation.state === "loading";

  return (
    <header
      className={cn(
        "sticky top-0 z-50 flex h-10 shrink-0 items-center justify-between gap-2 border-b px-4 md:px-6",
        "bg-background/95 backdrop-blur-sm supports-backdrop-filter:bg-background/50",
      )}
    >
      <div className="flex items-center gap-3">
        <ModulePanelToggle />
        <HistoryNavigation />
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink
                render={<Link to="/" />}
                className={cn(
                  "text-muted-foreground transition-opacity hover:text-foreground",
                  isLoading ? "opacity-50" : "",
                )}
              >
                Home
              </BreadcrumbLink>
            </BreadcrumbItem>
            {breadcrumbs.length > 0 && <BreadcrumbSeparator />}
            {breadcrumbs.map((crumb, index) => (
              <React.Fragment key={crumb.id}>
                <BreadcrumbItem>
                  {index < breadcrumbs.length - 1 ? (
                    <BreadcrumbLink
                      render={<Link to={crumb.pathname} />}
                      className={cn(
                        "text-muted-foreground transition-opacity hover:text-foreground",
                        isLoading ? "opacity-50" : "",
                      )}
                    >
                      {crumb.crumb}
                    </BreadcrumbLink>
                  ) : (
                    <BreadcrumbPage className="line-clamp-1">{crumb.crumb}</BreadcrumbPage>
                  )}
                </BreadcrumbItem>
                {index < breadcrumbs.length - 1 && <BreadcrumbSeparator />}
              </React.Fragment>
            ))}
          </BreadcrumbList>
        </Breadcrumb>
      </div>
      <NavActions />
    </header>
  );
}

function HistoryNavigation() {
  const { canGoBack, canGoForward, goBack, goForward } = useHistoryNavigation();

  return (
    <TooltipProvider>
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
                aria-label="Go back"
              >
                <ChevronLeft className="size-3.5" />
              </Button>
            }
          />
          <TooltipContent>Go back</TooltipContent>
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
                aria-label="Go forward"
              >
                <ChevronRight className="size-3.5" />
              </Button>
            }
          />
          <TooltipContent>Go forward</TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  );
}

function NavActions() {
  const location = useLocation();
  const pageUrl = location.pathname;
  const pageTitle = getPageTitle(pageUrl);

  const { data, isLoading } = useQuery({
    ...queries.pageFavorite.check(pageUrl),
    enabled: !!pageUrl,
  });

  const isFavorited = data?.favorited;

  const { mutateAsync, isPending } = useOptimisticMutation({
    queryKey: queries.pageFavorite.check(pageUrl).queryKey,
    mutationFn: async (values: ToggleFavoriteRequest) =>
      apiService.pageFavoriteService.togglePageFavorite(values),
    resourceName: "Page Favorite",
    invalidateQueries: [queries.pageFavorite.all._def, queries.pageFavorite.check._def],
    optimisticUpdate: (_variables, currentData) => !currentData,
    onSuccess: (result) => {
      toast.success(result.favorited ? "Added to favorites" : "Removed from favorites");
    },
  });

  const handleToggle = () => {
    void mutateAsync({ pageUrl, pageTitle });
  };

  return (
    <div className="ml-auto flex items-center gap-1 px-3">
      <NotificationPopover />
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                className="cursor-pointer"
                size="icon-xs"
                onClick={handleToggle}
                disabled={isLoading || isPending}
                aria-label={isFavorited ? "Remove from favorites" : "Add to favorites"}
              >
                <Star
                  className={cn(
                    "size-3.5 transition-colors",
                    isFavorited && "fill-amber-400 text-amber-400",
                  )}
                />
              </Button>
            }
          />
          <TooltipContent>
            {isFavorited ? "Remove from favorites" : "Add to favorites"}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}

function ModulePanelToggle() {
  const toggleModulePanel = useNavigationStore((state) => state.toggleModulePanel);

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-xs"
      onClick={toggleModulePanel}
      aria-label="Toggle navigation panel"
    >
      <PanelLeftIcon className="size-3.5" />
    </Button>
  );
}
