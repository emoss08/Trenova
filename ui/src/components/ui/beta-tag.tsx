import { faSparkles } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "./icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./tooltip";

export function BetaTag() {
  return (
    <TooltipProvider delayDuration={0}>
      <Tooltip>
        <TooltipTrigger>
          <span className="inline-flex items-center rounded-full bg-primary/10 gap-1 px-2 py-0.5 text-2xs font-medium text-primary ring-1 ring-inset ring-primary/20">
            <Icon icon={faSparkles} className="size-3 text-primary/70" />
            BETA
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p className="max-w-[300px] text-wrap">
            This feature is in beta and may be unstable.
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
