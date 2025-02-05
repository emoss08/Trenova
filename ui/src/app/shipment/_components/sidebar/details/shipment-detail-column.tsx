import { cn } from "@/lib/utils";

export function ShipmentDetailColumn({
  color,
  text,
  className,
}: {
  color?: string;
  text: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-x-1 text-sm font-medium text-foreground",
        className,
      )}
    >
      {color && (
        <div
          className="size-2 rounded-full"
          style={{ backgroundColor: color }}
        />
      )}
      <span className="text-wrap">{text}</span>
    </div>
  );
}

export function DetailsRow({
  label,
  value,
  className,
}: {
  label: string;
  value: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex justify-between items-center", className)}>
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="text-sm">{value}</p>
    </div>
  );
}
