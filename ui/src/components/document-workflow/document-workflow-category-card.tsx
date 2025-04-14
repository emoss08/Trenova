import { Badge } from "@/components/ui/badge";
import { Icon } from "@/components/ui/icons";
import { cn, truncateText } from "@/lib/utils";
import { type DocumentCategory } from "@/types/document";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import { useMemo } from "react";

export function CategoryCard({
  category,
  isActive,
  onClick,
}: {
  category: DocumentCategory;
  isActive: boolean;
  onClick: () => void;
}) {
  const categoryStyle = useMemo(() => {
    const bgColor = isActive ? "bg-accent/50" : "bg-background";

    return `border-border ${bgColor}`;
  }, [isActive]);

  return (
    <div
      className={cn(
        "p-3 rounded-md mb-2 cursor-pointer transition-all",
        "border hover:bg-accent/30",
        categoryStyle,
        !category.complete && "bg-muted/20",
      )}
      onClick={onClick}
      style={{
        borderLeftWidth: "4px",
        borderLeftColor: category.color,
      }}
    >
      <div className="flex items-center justify-between mb-1">
        <h3 className="font-medium truncate">{category.name}</h3>
        {category.complete ? (
          <Badge
            withDot={false}
            variant="active"
            className="flex items-center gap-1 text-xs"
          >
            <Icon icon={faCheck} className="size-3" />
            Complete
          </Badge>
        ) : (
          <Badge withDot={false} variant="inactive" className="text-xs">
            Not Complete
          </Badge>
        )}
      </div>

      <div className="text-xs text-muted-foreground flex justify-between">
        <span className="truncate">
          {truncateText(category.description, 25)}
        </span>
        <span>
          {category.documentsCount} Document
          {category.documentsCount > 1 ? "s" : ""}
        </span>
      </div>
    </div>
  );
}
