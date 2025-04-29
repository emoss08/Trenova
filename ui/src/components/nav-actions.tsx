import { Button } from "@/components/ui/button";
import { faStar } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "./ui/icons";

export function NavActions() {
  return (
    <div className="flex items-center gap-2 text-sm">
      <Button
        title="Favorite Page"
        variant="ghost"
        size="icon"
        className="size-7"
      >
        <Icon icon={faStar} />
      </Button>
    </div>
  );
}
