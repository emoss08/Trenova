import { Badge, BadgeProps } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export interface NotificationBadgeProps extends BadgeProps {
  label?: string | number;
  show?: boolean;
}

export const NotificationBadge = ({
  label,
  className,
  show,
  children,
  ...props
}: NotificationBadgeProps) => {
  const showBadge =
    typeof label !== "undefined" && (typeof show === "undefined" || show);
  return (
    <div className="inline-flex relative">
      {children}
      {showBadge && (
        <Badge
          className={cn(
            "absolute top-0 right-0 rounded-full",
            typeof label !== "undefined" && ("" + label).length === 0
              ? "translate-x-1 -translate-y-1 px-1.5 py-1.5"
              : "translate-x-1.5 -translate-y-1.5 px-2",
            className,
          )}
          {...props}
        >
          {"" + label}
        </Badge>
      )}
    </div>
  );
};
