import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

type DataTableColorColumnProps = {
  text: string;
  color?: string;
  className?: string;
};

export function DataTableColorColumn({
  text,
  color,
  className,
}: DataTableColorColumnProps) {
  return (
    <div className={cn("flex items-center gap-2", className)}>
      {color && (
        <span
          className="size-2 shrink-0 rounded-full"
          style={{ backgroundColor: color }}
        />
      )}
      <span className="truncate">{text}</span>
    </div>
  );
}

type DataTableDescriptionProps = {
  description?: string | null;
  truncateLength?: number;
  className?: string;
};

export function DataTableDescription({
  description,
  truncateLength = 50,
  className,
}: DataTableDescriptionProps) {
  if (!description) {
    return <span className="text-muted-foreground">-</span>;
  }

  const shouldTruncate = description.length > truncateLength;
  const displayText = shouldTruncate
    ? `${description.slice(0, truncateLength)}...`
    : description;

  if (!shouldTruncate) {
    return <span className={cn("text-sm", className)}>{description}</span>;
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            className={cn(
              "cursor-help text-sm underline decoration-muted-foreground decoration-dashed",
              className,
            )}
          >
            {displayText}
          </span>
        }
      />
      <TooltipContent className="max-w-xs">
        <p className="text-sm">{description}</p>
      </TooltipContent>
    </Tooltip>
  );
}

type DataTableLinkProps = {
  text: string;
  href?: string;
  onClick?: () => void;
  className?: string;
};

export function DataTableLink({
  text,
  href,
  onClick,
  className,
}: DataTableLinkProps) {
  if (href) {
    return (
      <a
        href={href}
        className={cn(
          "text-sm text-primary underline-offset-4 hover:underline",
          className,
        )}
        onClick={(e) => {
          if (onClick) {
            e.preventDefault();
            onClick();
          }
        }}
      >
        {text}
      </a>
    );
  }

  return (
    <button
      type="button"
      className={cn(
        "text-left text-sm text-primary underline-offset-4 hover:underline",
        className,
      )}
      onClick={onClick}
    >
      {text}
    </button>
  );
}

type DataTablePlaceholderProps = {
  text?: string;
  className?: string;
};

export function DataTablePlaceholder({
  text = "-",
  className,
}: DataTablePlaceholderProps) {
  return <span className={cn("text-muted-foreground", className)}>{text}</span>;
}
