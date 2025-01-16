import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { badgeVariants } from "@/lib/variants/badge";
import { type Status } from "@/types/common";
import { type VariantProps } from "class-variance-authority";
import { Badge } from "./ui/badge";

export function StatusBadge({ status }: { status: Status }) {
  return (
    <Badge
      variant={status === "Active" ? "active" : "inactive"}
      // className="max-h-6"
    >
      {status}
    </Badge>
  );
}

export type BadgeAttrProps = {
  variant: VariantProps<typeof badgeVariants>["variant"];
  text: string;
};

export function WorkerTypeBadge({ type }: { type: WorkerSchema["type"] }) {
  const typeAttr: Record<WorkerSchema["type"], BadgeAttrProps> = {
    Employee: {
      variant: "active",
      text: "Employee",
    },
    Contractor: {
      variant: "purple",
      text: "Contractor",
    },
  };

  return <Badge {...typeAttr[type]}>{typeAttr[type].text}</Badge>;
}
