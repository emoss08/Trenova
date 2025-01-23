import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { truncateText } from "@/lib/utils";

type DataTableDescriptionProps = {
  description?: string;
  truncateLength?: number;
};

export function DataTableDescription({
  description,
  truncateLength = 50,
}: DataTableDescriptionProps) {
  if (!description) {
    return <span>No description</span>;
  }

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

export function DataTableColorColumn({
  color,
  text,
}: {
  color?: string;
  text: string;
}) {
  const isColor = !!color;
  return isColor ? (
    <div className="flex items-center gap-x-1.5 text-sm font-medium text-foreground">
      <div
        className="size-2 rounded-full"
        style={{
          backgroundColor: color,
        }}
      />
      <p>{text}</p>
    </div>
  ) : (
    <p>{text}</p>
  );
}
