import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { generateDateOnlyString, toDate } from "@/lib/date";
import { cn, truncateText } from "@/lib/utils";

type DataTableDescriptionProps = {
  description?: string;
  truncateLength?: number;
};

export function DataTableDescription({
  description,
  truncateLength = 50,
}: DataTableDescriptionProps) {
  if (!description) {
    return <span>-</span>;
  }

  return (
    <Tooltip delayDuration={500}>
      <TooltipTrigger>
        <span>{truncateText(description, truncateLength)}</span>
      </TooltipTrigger>
      <TooltipContent>
        <p className="max-w-[300px] text-wrap">{description}</p>
      </TooltipContent>
    </Tooltip>
  );
}

export function DataTableColorColumn({
  color,
  text,
  className,
}: {
  color?: string;
  text: string;
  className?: string;
}) {
  const isColor = !!color;
  return isColor ? (
    <div
      className={cn(
        "flex items-center gap-x-1.5 text-sm font-medium text-foreground",
        className,
      )}
    >
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

export function LastInspectionDateBadge({
  value,
}: {
  value: number | null | undefined;
}) {
  const inspectionDate = toDate(value ?? undefined);

  if (!inspectionDate)
    return <Badge variant="inactive">No inspection date</Badge>;

  return (
    <Badge variant="active">{generateDateOnlyString(inspectionDate)}</Badge>
  );
}

export function BooleanBadge({ value }: { value: boolean }) {
  return (
    <Badge variant={value ? "active" : "inactive"}>
      {value ? "Yes" : "No"}
    </Badge>
  );
}
