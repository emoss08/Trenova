import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import { faLocationDot } from "@fortawesome/pro-solid-svg-icons";
import React from "react";

interface TimelineItemProps {
  icon?: React.ReactNode;
  content: React.ReactNode;
  isLast?: boolean;
  className?: string;
}

const TimelineItem: React.FC<TimelineItemProps> = ({
  icon,
  content,
  isLast = false,
  className,
}) => {
  return (
    <div className={cn("flex items-start", className)}>
      <div className="flex flex-col items-center mr-2 relative">
        {isLast ? (
          <div className="size-3 flex items-center justify-center z-10">
            <Icon
              icon={faLocationDot}
              className="text-muted-foreground scale-110"
            />
          </div>
        ) : (
          <div className="size-3 bg-muted-foreground rounded-full flex items-center justify-center z-10">
            {icon || <div className="size-1.5 bg-background rounded-full" />}
          </div>
        )}
        {!isLast && (
          <div
            className="h-full w-0.5 bg-muted-foreground absolute top-3 left-1/2 -translate-x-1/2"
            aria-hidden="true"
          />
        )}
      </div>
      <div className="flex-1">{content}</div>
    </div>
  );
};

interface TimelineProps {
  items: Array<{
    id: string | number;
    icon?: React.ReactNode;
    content: React.ReactNode;
  }>;
  className?: string;
}

export const Timeline: React.FC<TimelineProps> = ({ items, className }) => {
  return (
    <div
      className={cn("max-w-md space-y-2", className)}
      role="list"
      aria-label="Timeline"
    >
      {items.map((item, index) => (
        <TimelineItem
          key={item.id}
          icon={item.icon}
          content={item.content}
          isLast={index === items.length - 1}
        />
      ))}
    </div>
  );
};
