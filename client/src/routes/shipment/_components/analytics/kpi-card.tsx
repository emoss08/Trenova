import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

export function KPICard({
  label,
  value,
  icon: Icon,
  detail,
  children,
  onClick,
}: {
  label: string;
  value: string;
  icon: LucideIcon;
  detail?: string;
  children?: React.ReactNode;
  onClick?: () => void;
}) {
  return (
    <Card
      className={cn(
        "group relative gap-0 overflow-hidden rounded-md border-border/80 shadow-none transition-colors hover:border-border",
        onClick && "cursor-pointer",
      )}
      onClick={onClick}
    >
      <CardHeader className="relative flex flex-row items-start justify-between space-y-0 pb-2">
        <CardTitle className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
          {label}
        </CardTitle>
        <span className="inline-flex size-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
          <Icon className="size-4" />
        </span>
      </CardHeader>
      <CardContent className="relative space-y-1 pt-0">
        <p className="text-3xl leading-none font-semibold tracking-tight">{value}</p>
        {detail && <p className="text-[11px] text-muted-foreground">{detail}</p>}
        {children}
      </CardContent>
    </Card>
  );
}
