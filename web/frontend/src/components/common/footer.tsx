import { cn } from "@/lib/utils";

export function FooterContainer({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("bg-accent", className)}>
      <div className="font-display flex items-center justify-between gap-2">
        <div className="relative">{children}</div>
      </div>
    </div>
  );
}
