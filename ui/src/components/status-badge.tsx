import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { cn } from "@/lib/utils";
import { badgeVariants } from "@/lib/variants/badge";
import { type Status } from "@/types/common";
import { DocumentType } from "@/types/document";
import { type PackingGroupChoiceProps } from "@/types/hazardous-material";
import { MoveStatus } from "@/types/move";
import { ShipmentStatus } from "@/types/shipment";
import { StopStatus } from "@/types/stop";
import { EquipmentStatus } from "@/types/tractor";
import { type VariantProps } from "class-variance-authority";
import { Badge } from "./ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./ui/tooltip";

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
  description?: string;
};

export type PlainBadgeAttrProps = {
  text: string;
  description?: string;
  className?: string;
};

export function WorkerTypeBadge({ type }: { type: WorkerSchema["type"] }) {
  const typeAttr: Record<WorkerSchema["type"], BadgeAttrProps> = {
    Employee: {
      variant: "indigo",
      text: "Employee",
    },
    Contractor: {
      variant: "orange",
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

export function MoveStatusBadge({ status }: { status: MoveStatus }) {
  const moveStatusAttributes: Record<MoveStatus, BadgeAttrProps> = {
    [MoveStatus.New]: {
      variant: "purple",
      text: "New",
    },
    [MoveStatus.Assigned]: {
      variant: "warning",
      text: "Assigned",
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
    <Badge variant={moveStatusAttributes[status].variant} className="max-h-6">
      {moveStatusAttributes[status].text}
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

export function ShipmentStatusBadge({
  status,
  withDot,
  className,
}: {
  status?: ShipmentStatus;
  withDot?: boolean;
  className?: string;
}) {
  if (!status) return null;

  const statusAttributes: Record<ShipmentStatus, BadgeAttrProps> = {
    [ShipmentStatus.New]: {
      variant: "purple",
      text: "New",
      description:
        "Shipment has been created and is pending initial assignment.",
    },
    [ShipmentStatus.PartiallyAssigned]: {
      variant: "indigo",
      text: "Partially Assigned",
      description:
        "Equipment or worker assignments are pending for one or more moves within this shipment.",
    },
    [ShipmentStatus.PartiallyCompleted]: {
      variant: "indigo",
      text: "Partially Completed",
      description:
        "Some moves within this shipment have been completed, but not all.",
    },
    [ShipmentStatus.Assigned]: {
      variant: "warning",
      text: "Assigned",
      description:
        "All required equipment and workers have been assigned to this shipment's moves.",
    },
    [ShipmentStatus.InTransit]: {
      variant: "info",
      text: "In Transit",
      description:
        "Active shipment with cargo currently in transport between designated locations.",
    },
    [ShipmentStatus.Delayed]: {
      variant: "orange",
      text: "Delayed",
      description:
        "Shipment has exceeded scheduled arrival or delivery timeframes at one or more stops.",
    },
    [ShipmentStatus.Completed]: {
      variant: "active",
      text: "Completed",
      description:
        "All transportation activities for this shipment have been successfully completed.",
    },
    [ShipmentStatus.Billed]: {
      variant: "teal",
      text: "Billed",
      description:
        "Invoice has been generated and posted for completed transportation services.",
    },
    [ShipmentStatus.Canceled]: {
      variant: "inactive",
      text: "Canceled",
      description:
        "Shipment has been terminated and will not be completed as originally planned.",
    },
  };

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <Badge
            withDot={withDot}
            variant={statusAttributes[status].variant}
            className={cn(className, "max-h-6")}
          >
            {statusAttributes[status].text}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs text-wrap text-center">
          <p>{statusAttributes[status].description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function DocumentTypeBadge({
  documentType,
}: {
  documentType: DocumentType;
}) {
  const documentTypeAttributes: Record<DocumentType, BadgeAttrProps> = {
    [DocumentType.License]: {
      variant: "purple",
      text: "License",
    },
    [DocumentType.Registration]: {
      variant: "indigo",
      text: "Registration",
    },
    [DocumentType.Insurance]: {
      variant: "warning",
      text: "Insurance",
    },
    [DocumentType.Invoice]: {
      variant: "teal",
      text: "Invoice",
    },
    [DocumentType.ProofOfDelivery]: {
      variant: "info",
      text: "Proof of Delivery",
    },
    [DocumentType.BillOfLading]: {
      variant: "purple",
      text: "Bill of Lading",
    },
    [DocumentType.DriverLog]: {
      variant: "info",
      text: "Driver Log",
    },
    [DocumentType.MedicalCert]: {
      variant: "purple",
      text: "Medical Certificate",
    },
    [DocumentType.Contract]: {
      variant: "purple",
      text: "Contract",
    },
    [DocumentType.Maintenance]: {
      variant: "purple",
      text: "Maintenance",
    },
    [DocumentType.AccidentReport]: {
      variant: "purple",
      text: "Accident Report",
    },
    [DocumentType.TrainingRecord]: {
      variant: "purple",
      text: "Training Record",
    },
    [DocumentType.Other]: {
      variant: "purple",
      text: "Other",
    },
  };

  return (
    <Badge
      variant={documentTypeAttributes[documentType].variant}
      className="max-h-6"
      withDot={false}
    >
      {documentTypeAttributes[documentType].text}
    </Badge>
  );
}

export function PlainShipmentStatusBadge({
  status,
}: {
  status: ShipmentStatus;
}) {
  const statusAttributes: Record<ShipmentStatus, PlainBadgeAttrProps> = {
    [ShipmentStatus.New]: {
      className: "bg-purple-600",
      text: "New",
      description:
        "Shipment has been created and is pending initial assignment.",
    },
    [ShipmentStatus.PartiallyAssigned]: {
      className: "bg-indigo-600",
      text: "Partially Assigned",
      description:
        "Equipment or worker assignments are pending for one or more moves within this shipment.",
    },
    [ShipmentStatus.Assigned]: {
      className: "bg-warning",
      text: "Assigned",
      description:
        "All required equipment and workers have been assigned to this shipment's moves.",
    },
    [ShipmentStatus.InTransit]: {
      className: "bg-info",
      text: "In Transit",
      description:
        "Active shipment with cargo currently in transport between designated locations.",
    },
    [ShipmentStatus.Delayed]: {
      className: "bg-orange-600",
      text: "Delayed",
      description:
        "Shipment has exceeded scheduled arrival or delivery timeframes at one or more stops.",
    },
    [ShipmentStatus.Completed]: {
      className: "bg-green-600",
      text: "Completed",
      description:
        "All transportation activities for this shipment have been successfully completed.",
    },
    [ShipmentStatus.PartiallyCompleted]: {
      className: "bg-indigo-600",
      text: "Partially Completed",
      description:
        "Some moves within this shipment have been completed, but not all.",
    },
    [ShipmentStatus.Billed]: {
      className: "bg-teal-600",
      text: "Billed",
      description:
        "Invoice has been generated and posted for completed transportation services.",
    },
    [ShipmentStatus.Canceled]: {
      className: "bg-red-600",
      text: "Canceled",
      description:
        "Shipment has been terminated and will not be completed as originally planned.",
    },
  };

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <div className="flex items-center gap-x-1">
            <div
              className={cn(
                "size-2 rounded-full",
                statusAttributes[status].className,
              )}
            />
            <p>{statusAttributes[status].text}</p>
          </div>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs text-wrap text-center">
          <p>{statusAttributes[status].description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
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
