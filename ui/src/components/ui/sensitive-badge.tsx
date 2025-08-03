/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";

interface SensitiveBadgeProps {
  variant?: "default" | "warning" | "destructive";
  size?: "xs" | "sm" | "md";
  className?: string;
}

export function SensitiveBadge({
  variant = "warning",
  size = "sm",
  className,
}: SensitiveBadgeProps) {
  const variantStyles = {
    default: "bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300",
    warning: "bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400",
    destructive: "bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-400",
  };

  const sizeStyles = {
    xs: "text-[10px] px-1 py-0.5",
    sm: "text-xs px-1.5 py-0.5",
    md: "text-sm px-2 py-1",
  };

  return (
    <span
      title="This field contains sensitive data that has been masked for security."
      className={cn(
        "inline-flex items-center rounded-sm font-medium select-none",
        variantStyles[variant],
        sizeStyles[size],
        className
      )}
    >
      Sensitive
    </span>
  );
}