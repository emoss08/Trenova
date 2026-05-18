import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { EDITemplate } from "@/types/edi";

type TemplateListProps = {
  templates: EDITemplate[];
  selectedTemplateId: string;
  onSelect: (templateId: string) => void;
};

export default function TemplateList({
  templates,
  selectedTemplateId,
  onSelect,
}: TemplateListProps) {
  return (
    <div className="min-h-0 flex-1 overflow-auto">
      {templates.map((template) => (
        <button
          key={template.id}
          type="button"
          onClick={() => onSelect(template.id)}
          className={cn(
            "block w-full border-b px-3 py-2 text-left hover:bg-muted",
            selectedTemplateId === template.id && "bg-muted",
          )}
        >
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-sm font-medium">{template.name}</span>
            <Badge variant={template.status === "Active" ? "active" : "outline"}>
              {template.status}
            </Badge>
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {template.transactionSet} {template.direction} / {template.versions.length} versions
          </div>
        </button>
      ))}
      {templates.length === 0 ? (
        <div className="p-3 text-sm text-muted-foreground">No matching templates.</div>
      ) : null}
    </div>
  );
}
