import { Button } from "@/components/ui/button";
import {
  useCurrentPageFavorite,
  useToggleCurrentPageFavorite,
} from "@/hooks/use-favorites";
import { faStar } from "@fortawesome/pro-regular-svg-icons";
import { faStar as faStarSolid } from "@fortawesome/pro-solid-svg-icons";
import { Icon } from "./ui/icons";

export function NavActions() {
  const { data: favoriteData, isLoading } = useCurrentPageFavorite();
  const { toggle, isPending } = useToggleCurrentPageFavorite();

  const isFavorite = favoriteData?.isFavorite ?? false;
  console.info("favorite information", {
    isFavorite,
    favoriteData,
  });

  return (
    <div className="flex items-center gap-2 text-sm">
      <Button
        title={isFavorite ? "Remove from favorites" : "Add to favorites"}
        variant="ghost"
        size="icon"
        className="size-7"
        onClick={toggle}
        disabled={isLoading || isPending}
      >
        <Icon
          icon={isFavorite ? faStarSolid : faStar}
          className={isFavorite ? "text-yellow-500" : "text-muted-foreground"}
        />
      </Button>
    </div>
  );
}
