import { useT } from "@trenova/shared/i18n/use-t";
import { NotificationSheet } from "@/components/notification-center/notification-sheet";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@trenova/shared/components/ui/breadcrumb";
import { useBreadcrumbs } from "@/hooks/use-breadcrumb";
import { useHistoryNavigation } from "@/hooks/use-history-navigation";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { queries } from "@/lib/queries";
import { getPageTitle } from "@/lib/route-utils";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import { useNavigationStore } from "@/stores/navigation-store";
import type { ToggleFavoriteRequest } from "@/types/page-favorite";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, ChevronRight, PanelLeftIcon, Star } from "lucide-react";
import React from "react";
import { Link, useLocation, useNavigation } from "react-router";
import { toast } from "sonner";
import { SystemInformation } from "./system-information";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";

export function Header() {
  return (
    <header
      className={cn(
        "sticky top-0 z-50 flex h-10 shrink-0 items-center justify-between gap-2 border-b px-4 md:px-6",
        "bg-background/95 supports-backdrop-filter:bg-background/50 backdrop-blur-sm",
      )}
    >
      <div className="flex items-center gap-3">
        <SidebarToggle />
        <HistoryNavigation />
        <HeaderBreadcrumbs />
      </div>
      <NavActions />
    </header>
  );
}

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
              <ChevronLeft className="size-3.5" />
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
              <ChevronRight className="size-3.5" />
            </Button>
          }
        />
        <TooltipContent>{t("Go forward")}</TooltipContent>
      </Tooltip>
    </div>
  );
}

function NavActions() {
  return (
    <div className="ml-auto flex items-center gap-1 px-3 text-center">
      <SystemInformation />
      <NotificationSheet />
      <FavoriteToggle />
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
        <Star
          className={cn("size-3 transition-colors", isFavorited && "fill-amber-400 text-amber-400")}
        />
      </TooltipTrigger>
      <TooltipContent>{isFavorited ? t("Remove from favorites") : t("Add to favorites")}</TooltipContent>
    </Tooltip>
  );
}

/**
 * The trail as the header draws it: Home first, then every crumb the route
 * provides, the last one as plain text.
 */
export function HeaderBreadcrumbs() {
  const t = useT();

  const navigation = useNavigation();
  const breadcrumbs = useBreadcrumbs();
  const isLoading = navigation.state === "loading";

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink
            render={<Link to="/" />}
            className={cn(
              "text-muted-foreground hover:text-foreground transition-opacity",
              isLoading ? "opacity-50" : "",
            )}
          >
            {t("Home")}
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
                    "text-muted-foreground hover:text-foreground transition-opacity",
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
  );
}

export function SidebarToggle() {
  const collapsed = useNavigationStore((state) => state.sidebarCollapsed);
  const toggleSidebar = useNavigationStore((state) => state.toggleSidebar);
  const label = collapsed ? "Show sidebar" : "Hide sidebar";

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={toggleSidebar}
            aria-label={label}
            aria-pressed={!collapsed}
          />
        }
      >
        <PanelLeftIcon className="size-3.5" />
      </TooltipTrigger>
      <TooltipContent>
        {label} ({formatShortcut("B")})
      </TooltipContent>
    </Tooltip>
  );
}
