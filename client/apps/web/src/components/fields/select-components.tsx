import { cn } from "@trenova/shared/lib/utils";

export function ColorOptionValue({
  color,
  value,
  className,
  textClassName,
}: {
  value: any;
  color?: string;
  className?: string;
  textClassName?: string;
}) {
  const isColor = !!color;

  return isColor ? (
    <div
      className={cn(
        "text-foreground flex h-5 items-center text-sm font-normal",
        isColor && "gap-x-1.5",
        className,
      )}
    >
      <div
        className="size-2 rounded-full"
        style={{
          backgroundColor: color,
        }}
      />
      <p className={cn("text-xs", textClassName)}>{value}</p>
    </div>
  ) : (
    value
  );
}
