import { cn } from "@/lib/utils";
import { SparklesIcon } from "lucide-react";
import { Badge } from "./ui/badge";
type BetaTagProps = {
  label?: string;
  includeIcon?: boolean;
  className?: string;
};

export function BetaTag({ label = "BETA", includeIcon = true, className }: BetaTagProps) {
  return (
    <Badge tabIndex={0} variant="info" className={cn("ml-auto h-5", className)}>
      {includeIcon && <SparklesIcon />}
      {label}
    </Badge>
  );
}
