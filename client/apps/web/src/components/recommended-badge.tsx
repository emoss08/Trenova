import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import { CircleCheckIcon } from "lucide-react";
import type React from "react";

type RecommendedBadgeVariant = "default" | "premium" | "success" | "warning";

interface RecommendedBadgeProps {
  text?: string;
  size?: "sm" | "md" | "lg";
  variant?: RecommendedBadgeVariant;
  className?: string;
  tooltip?: React.ReactNode;
}

const variantTone = {
  default: "brand",
  premium: "brand",
  success: "success",
  warning: "brand",
} as const satisfies Record<RecommendedBadgeVariant, "brand" | "success">;

const sizeClasses = {
  sm: "h-4.5 px-1.5 text-2xs",
  md: "",
  lg: "h-5.5 px-2.5 text-sm",
} as const;

export function RecommendedBadge({
  text = "Recommended",
  size = "md",
  variant = "default",
  className,
  tooltip,
}: RecommendedBadgeProps) {
  const badge = (
    <Badge variant={variantTone[variant]} className={cn("select-none", sizeClasses[size], className)}>
      <CircleCheckIcon aria-hidden />
      {text}
    </Badge>
  );

  if (!tooltip) {
    return badge;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger render={badge} />
        <TooltipContent>{tooltip}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
