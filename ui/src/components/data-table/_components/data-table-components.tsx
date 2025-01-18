import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import { truncateText } from "@/lib/utils";

type DataTableDescriptionProps = {
  description: string;
  truncateLength?: number;
};

export function DataTableDescription({
  description,
  truncateLength = 50,
}: DataTableDescriptionProps) {
  return (
    <TooltipProvider delayDuration={0}>
      <Tooltip>
        <TooltipTrigger>
          <span>{truncateText(description, truncateLength)}</span>
        </TooltipTrigger>
        <TooltipContent>
          <p className="max-w-[300px] text-wrap">{description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
