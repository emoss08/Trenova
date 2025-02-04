import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { badgeVariants } from "@/lib/variants/badge";
import { type Status } from "@/types/common";
import { type PackingGroupChoiceProps } from "@/types/hazardous-material";
import { ShipmentStatus } from "@/types/shipment";
import { StopStatus } from "@/types/stop";
import { EquipmentStatus } from "@/types/tractor";
import { type VariantProps } from "class-variance-authority";
import { Badge } from "./ui/badge";

export function StatusBadge({ status }: { status: Status }) {
  return (
    <Badge variant={status === "Active" ? "active" : "inactive"}>
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

export function StopStatusBadge({ status }: { status: StopStatus }) {
  const stopStatusAttributes: Record<StopStatus, BadgeAttrProps> = {
    [StopStatus.New]: {
      variant: "purple",
      text: "New",
    },
    [StopStatus.InTransit]: {
      variant: "info",
      text: "In Transit",
    },
    [StopStatus.Completed]: {
      variant: "active",
      text: "Completed",
    },
    [StopStatus.Canceled]: {
      variant: "inactive",
      text: "Canceled",
    },
  };

  return (
    <Badge variant={stopStatusAttributes[status].variant} className="max-h-6">
      {stopStatusAttributes[status].text}
    </Badge>
  );
}

export function EquipmentStatusBadge({ status }: { status: EquipmentStatus }) {
  const statusAttributes: Record<EquipmentStatus, BadgeAttrProps> = {
    [EquipmentStatus.Available]: {
      variant: "active",
      text: "Available",
    },
    [EquipmentStatus.OOS]: {
      variant: "inactive",
      text: "Out of Service",
    },
    [EquipmentStatus.AtMaintenance]: {
      variant: "purple",
      text: "At Maintenance",
    },
    [EquipmentStatus.Sold]: {
      variant: "warning",
      text: "Sold",
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className="max-h-6">
      {statusAttributes[status].text}
    </Badge>
  );
}

export function ShipmentStatusBadge({ status }: { status: ShipmentStatus }) {
  const statusAttributes: Record<ShipmentStatus, BadgeAttrProps> = {
    [ShipmentStatus.New]: {
      variant: "purple",
      text: "New",
    },
    [ShipmentStatus.InTransit]: {
      variant: "info",
      text: "In Transit",
    },
    [ShipmentStatus.Delayed]: {
      variant: "warning",
      text: "Delayed",
    },
    [ShipmentStatus.Completed]: {
      variant: "active",
      text: "Completed",
    },
    [ShipmentStatus.Billed]: {
      variant: "teal",
      text: "Billed",
    },
    [ShipmentStatus.Canceled]: {
      variant: "inactive",
      text: "Canceled",
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className="max-h-6">
      {statusAttributes[status].text}
    </Badge>
  );
}

export function PackingGroupBadge({
  group,
}: {
  group: PackingGroupChoiceProps;
}) {
  const packingGroupAttributes: Record<
    PackingGroupChoiceProps,
    BadgeAttrProps
  > = {
    I: {
      variant: "inactive",
      text: "High Danger",
    },
    II: {
      variant: "warning",
      text: "Medium Danger",
    },
    III: {
      variant: "active",
      text: "Low Danger",
    },
  };

  return (
    <Badge variant={packingGroupAttributes[group].variant} className="max-h-6">
      {packingGroupAttributes[group].text}
    </Badge>
  );
}

export function HazmatBadge({ isHazmat }: { isHazmat: boolean }) {
  return (
    <Badge variant={isHazmat ? "active" : "inactive"} className="max-h-6">
      {isHazmat ? "Yes" : "No"}
    </Badge>
  );
}
