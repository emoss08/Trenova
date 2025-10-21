/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Icon, type IconProp } from "@/components/ui/icons";

interface StatCardProps {
  title: string;
  value: string | number;
  description?: string;
  icon?: IconProp;
  trend?: number;
  loading?: boolean;
  color?: "default" | "primary" | "success" | "warning" | "danger" | "info";
}

const colorMap = {
  default: "text-gray-500",
  primary: "text-primary",
  success: "text-green-500",
  warning: "text-amber-500",
  danger: "text-red-500",
  info: "text-blue-500",
};

export function StatCard({
  title,
  value,
  description,
  icon,
  trend,
  loading = false,
  color = "default",
}: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {icon && <Icon icon={icon} className={`size-4 ${colorMap[color]}`} />}
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="h-7 w-16 animate-pulse rounded bg-muted" />
        ) : (
          <div className="text-2xl font-bold">{value}</div>
        )}
        {description && (
          <CardDescription className="text-xs text-muted-foreground">
            {description}
            {trend !== undefined && (
              <span
                className={`ml-1 ${trend > 0 ? "text-green-500" : trend < 0 ? "text-red-500" : ""}`}
              >
                {trend > 0 ? "↑" : trend < 0 ? "↓" : ""} {Math.abs(trend)}%
              </span>
            )}
          </CardDescription>
        )}
      </CardContent>
    </Card>
  );
}
