import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";

export function SidebarSectionLabel({
  children,
  endContent,
  className,
}: {
  children: React.ReactNode;
  endContent?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex h-6 items-center justify-between px-2", className)}>
      <span className="text-2xs text-muted-foreground font-semibold tracking-wider uppercase select-none">
        {children}
      </span>
      {endContent}
    </div>
  );
}

export function SidebarNavLink({
  to,
  active,
  disabled,
  className,
  children,
}: {
  to: string;
  active?: boolean;
  disabled?: boolean;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <Link
      to={to}
      className={cn(
        "relative flex h-6 items-center gap-2 rounded-md px-2 text-base transition-colors",
        active
          ? cn(
              "from-brand/15 via-brand/5 text-foreground bg-gradient-to-r to-transparent font-medium",
              "before:bg-brand before:shadow-brand/60 before:absolute before:top-1 before:bottom-1 before:left-0 before:w-0.5 before:rounded-full before:shadow-[0_0_6px]",
            )
          : "text-foreground/70 hover:bg-muted hover:text-foreground",
        disabled && "pointer-events-none opacity-40",
        className,
      )}
      aria-disabled={disabled}
      tabIndex={disabled ? -1 : undefined}
    >
      {children}
    </Link>
  );
}
