/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Badge } from "@/components/ui/badge";
import { Icon } from "@/components/ui/icons";
import { cn, truncateText } from "@/lib/utils";
import { type DocumentCategory } from "@/types/document";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import React, { useMemo } from "react";

function CategoryCardContent({
  isActive,
  category,
  onClick,
  children,
}: {
  isActive: boolean;
  category: DocumentCategory;
  onClick: () => void;
  children: React.ReactNode;
}) {
  const categoryStyle = useMemo(() => {
    const bgColor = isActive ? "bg-muted" : "bg-background";

    return `border-border ${bgColor}`;
  }, [isActive]);

  return (
    <div
      className={cn(
        "p-3 rounded-md mb-2 cursor-pointer transition-all",
        "border hover:bg-muted-foreground/10",
        categoryStyle,
        !category.complete && "bg-background",
      )}
      onClick={onClick}
      style={{
        borderLeftWidth: "4px",
        borderLeftColor: category.color,
      }}
    >
      {children}
    </div>
  );
}

function CategoryCardHeader({ category }: { category: DocumentCategory }) {
  return (
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
  );
}

function CategoryCardFooter({ category }: { category: DocumentCategory }) {
  return (
    <div className="text-xs text-muted-foreground flex justify-between">
      <span className="truncate">{truncateText(category.description, 25)}</span>
      <span>
        {category.documentsCount} Document
        {category.documentsCount > 1 ? "s" : ""}
      </span>
    </div>
  );
}

export function CategoryCard({
  category,
  isActive,
  onClick,
}: {
  category: DocumentCategory;
  isActive: boolean;
  onClick: () => void;
}) {
  return (
    <CategoryCardContent
      isActive={isActive}
      category={category}
      onClick={onClick}
    >
      <CategoryCardHeader category={category} />
      <CategoryCardFooter category={category} />
    </CategoryCardContent>
  );
}
