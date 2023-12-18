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
      <div className="flex items-center justify-between gap-2 font-display">
        <div className="relative">{children}</div>
      </div>
    </div>
  );
}
