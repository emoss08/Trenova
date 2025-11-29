import { Badge } from "@/components/ui/badge";
import type { TemplateStatusSchema } from "@/lib/schemas/document-template-schema";

export function TemplateStatusBadge({
  status,
}: {
  status: TemplateStatusSchema;
}) {
  const statusMap: Record<
    TemplateStatusSchema,
    { variant: "purple" | "active" | "inactive"; text: string }
  > = {
    Draft: { variant: "purple", text: "Draft" },
    Active: { variant: "active", text: "Active" },
    Archived: { variant: "inactive", text: "Archived" },
  };

  const { variant, text } = statusMap[status];
  return (
    <Badge variant={variant} withDot={false}>
      {text}
    </Badge>
  );
}
