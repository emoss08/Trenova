import { useT } from "@trenova/shared/i18n/use-t";
import { useSidebarFold } from "@/components/navigation/sidebar-fold";
import { useBreadcrumbs } from "@/hooks/use-breadcrumb";
import { useHistoryNavigation } from "@/hooks/use-history-navigation";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { pageFavorite } from "@/lib/queries/page-favorite";
import { getPageTitle } from "@/lib/route-utils";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { pageFavoriteService } from "@/services/page-favorite";
import type { CheckFavoriteResponse, ToggleFavoriteRequest } from "@/types/page-favorite";
import { useQuery } from "@tanstack/react-query";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  LayoutLeftIcon,
  Star01Icon,
} from "@trenova/shared/components/icons";
import { useState } from "react";
import { useLocation } from "react-router";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";

/**
 * The size of every icon button the shell puts in the page bar: the same 28px as the `sm`
 * buttons a page passes as its actions, so the bar's controls share one height from the
 * fold on the left to the star on the right.
 */
const PAGE_BAR_BUTTON_SIZE = "icon-sm";

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
              size={PAGE_BAR_BUTTON_SIZE}
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
              size={PAGE_BAR_BUTTON_SIZE}
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
  const pageTitle = breadcrumbs.at(-1)?.label ?? getPageTitle(pageUrl);

  const { data, isLoading } = useQuery({
    ...pageFavorite.check(pageUrl),
    enabled: !!pageUrl,
  });

  const isFavorited = data?.favorited;

  const { mutateAsync, isPending } = useOptimisticMutation({
    queryKey: pageFavorite.check(pageUrl).queryKey,
    mutationFn: async (values: ToggleFavoriteRequest) =>
      pageFavoriteService.togglePageFavorite(values),
    resourceName: "Page Favorite",
    invalidateQueries: [pageFavorite.all._def, pageFavorite.check._def],
    // The star answers the press at once: the check is flipped in place and the server's
    // answer replaces it when it arrives.
    optimisticUpdate: (_variables, current: CheckFavoriteResponse | undefined) => ({
      favorited: !current?.favorited,
    }),
  });
  // The page whose star was just pressed, so the star lands only for that press and not
  // every time a page that is already a favorite is opened.
  const [landedOn, setLandedOn] = useState<string | null>(null);

  const handleToggle = () => {
    // The press is refused while the last one is still on its way. The button is not
    // disabled for it, because dimming it would fight the colour it is changing to.
    if (isPending) return;
    setLandedOn(isFavorited ? null : pageUrl);
    void mutateAsync({ pageUrl, pageTitle });
  };

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="outline"
            className={cn(
              "cursor-pointer duration-200",
              isFavorited &&
                "border-warning-border bg-warning-subtle text-warning-foreground hover:border-warning-border hover:bg-warning-subtle-hover hover:text-warning-foreground active:bg-warning-subtle-hover",
              className,
            )}
            size={PAGE_BAR_BUTTON_SIZE}
            onClick={handleToggle}
            disabled={isLoading}
            aria-busy={isPending}
            aria-label={isFavorited ? t("Remove from favorites") : t("Add to favorites")}
          />
        }
      >
        <Star01Icon
          onAnimationEnd={() => setLandedOn(null)}
          className={cn(
            "size-3.5 transition-colors duration-200",
            isFavorited && "fill-warning",
            isFavorited && landedOn === pageUrl && "animate-confirm motion-reduce:animate-none",
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
  const t = useT();
  const { shown, toggle } = useSidebarFold();
  const label = shown ? t("Hide sidebar") : t("Show sidebar");

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size={PAGE_BAR_BUTTON_SIZE}
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
