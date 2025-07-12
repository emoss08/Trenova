import { cn } from "@/lib/utils";
import { detectSensitiveDataType } from "@/lib/json-sensitive-utils";
import { SensitiveBadge } from "./sensitive-badge";

interface SensitiveValueProps {
  value: any;
  className?: string;
  showQuotes?: boolean;
  prefix?: string;
}

export function SensitiveValue({ 
  value, 
  className, 
  showQuotes = true,
  prefix = "" 
}: SensitiveValueProps) {
  const sensitiveInfo = detectSensitiveDataType(value);
  
  // Format the display value
  const displayValue = (() => {
    if (typeof value === "string" && showQuotes) {
      return `"${value}"`;
    }
    return String(value);
  })();

  if (sensitiveInfo.isSensitive) {
    return (
      <div className="inline-flex items-center gap-1.5">
        <span
          className={cn(
            "font-mono",
            sensitiveInfo.type === "redacted"
              ? "text-red-600 dark:text-red-400 font-medium"
              : "text-orange-600 dark:text-orange-400",
            className
          )}
          title={`${value}`}
        >
          {prefix}{displayValue}
        </span>
        <SensitiveBadge
          variant={
            sensitiveInfo.type === "redacted" ? "destructive" : "warning"
          }
          size="xs"
        />
      </div>
    );
  }

  // Return non-sensitive value with appropriate styling
  const valueClassName = (() => {
    if (typeof value === "string") {
      return "text-green-600 dark:text-green-400";
    } else if (typeof value === "number") {
      return "text-blue-600 dark:text-blue-400";
    } else if (typeof value === "boolean") {
      return "text-purple-600 dark:text-purple-400";
    } else if (value === null) {
      return "text-muted-foreground italic";
    }
    return "";
  })();

  return (
    <span
      className={cn("font-mono", valueClassName, className)}
      title={typeof value === "string" ? value : undefined}
    >
      {prefix}{displayValue}
    </span>
  );
}