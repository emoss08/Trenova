import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { cn } from "@/lib/utils";
import { faCheck, faClipboard } from "@fortawesome/pro-regular-svg-icons";
import React from "react";
import { Button } from "./button";
import { Icon } from "./icons";
import { Label } from "./label";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./tooltip";

export function CopyText({
  label,
  value,
  className,
}: {
  label: string;
  value: string;
  className?: string;
}) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className={cn("group flex flex-col gap-1", className)}>
      <Label>{label}</Label>
      <div className="flex h-7 w-full items-center justify-between gap-1 truncate rounded-md border border-muted-foreground/20 bg-primary/5 px-2 py-1 font-mono text-sm text-muted-foreground">
        <span className="flex-grow">{value}</span>
        <CopyButton active={isCopied} onClick={() => copy(value)} />
      </div>
    </div>
  );
}

function CopyButton({
  active,
  onClick,
}: {
  active: boolean;
  onClick: () => void;
}) {
  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    e.stopPropagation();
    onClick();
  };

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="xs"
            variant="outline"
            onClick={handleClick}
            className="size-5"
          >
            {active ? (
              <Icon icon={faCheck} className="size-4" />
            ) : (
              <Icon icon={faClipboard} className="size-4" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent>{active ? "Copied" : "Copy"}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
