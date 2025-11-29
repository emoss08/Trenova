import { ChevronDown, ChevronRight } from "lucide-react";
import { Activity, useState } from "react";

interface CollapsibleSectionProps {
  title: string;
  icon: React.ElementType;
  children: React.ReactNode;
  defaultOpen?: boolean;
}

export function CollapsibleSection({
  title,
  icon: SectionIcon,
  children,
  defaultOpen = true,
}: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="rounded-lg border border-border/50 bg-muted/30">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex w-full items-center gap-2 p-3 text-left transition-colors hover:bg-muted/50"
      >
        <SectionIcon className="size-4 text-primary" />
        <span className="flex-1 text-sm font-medium">{title}</span>
        {isOpen ? (
          <ChevronDown className="size-4 text-muted-foreground" />
        ) : (
          <ChevronRight className="size-4 text-muted-foreground" />
        )}
      </button>
      <Activity mode={isOpen ? "visible" : "hidden"}>
        <div className="border-t border-border/50 p-3">{children}</div>
      </Activity>
    </div>
  );
}
