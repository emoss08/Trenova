import { Button } from "@/components/ui/button";
import { faStar } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "./ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./ui/tooltip";

export function NavActions() {
  return (
    <div className="flex items-center gap-2 text-sm">
      <TooltipProvider>
        <Tooltip delayDuration={0}>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" className="size-7">
              <Icon icon={faStar} />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">Favorite Page</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}
