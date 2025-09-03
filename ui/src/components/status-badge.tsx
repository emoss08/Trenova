/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ConsolidationStatus } from "@/lib/schemas/consolidation-schema";
import { MoveStatus, type MoveSchema } from "@/lib/schemas/move-schema";
import {
  HoldSeverity,
  HoldType,
  ShipmentHoldSchema,
} from "@/lib/schemas/shipment-hold-schema";
import {
  ShipmentStatus,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { StopStatus, type StopSchema } from "@/lib/schemas/stop-schema";
import { EquipmentStatus } from "@/lib/schemas/tractor-schema";
import {
  WorkerPTOSchema,
  type WorkerSchema,
} from "@/lib/schemas/worker-schema";
import { cn } from "@/lib/utils";
import { badgeVariants } from "@/lib/variants/badge";
import { type Status } from "@/types/common";
import { DocumentStatus, DocumentType } from "@/types/document";
import { type PackingGroupChoiceProps } from "@/types/hazardous-material";
import { IntegrationCategory } from "@/types/integration";
import { PTOStatus, PTOType } from "@/types/worker";
import {
  faBadgeCheck,
  faCheck,
  faCircleXmark,
  faClock,
  faFileInvoiceDollar,
  faPaperPlane,
  faSparkles,
  faSpinner,
  faXmark,
} from "@fortawesome/pro-solid-svg-icons";
import { type VariantProps } from "class-variance-authority";
import { Badge } from "./ui/badge";
import { Icon, type IconProp } from "./ui/icons";

export function StatusBadge({ status }: { status: Status }) {
  return (
    <Badge
      withDot={false}
      variant={status === "Active" ? "active" : "inactive"}
      className="[&_svg]:size-2.5"
    >
      {status === "Active" ? <Icon icon={faCheck} /> : <Icon icon={faXmark} />}
      {status}
    </Badge>
  );
}

export type BadgeAttrProps = {
  variant: VariantProps<typeof badgeVariants>["variant"];
  text: string;
  description?: string;
  icon?: IconProp;
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

export function StopStatusBadge({ status }: { status: StopSchema["status"] }) {
  const stopStatusAttributes: Record<StopSchema["status"], BadgeAttrProps> = {
    [StopStatus.enum.New]: {
      variant: "purple",
      text: "New",
    },
    [StopStatus.enum.InTransit]: {
      variant: "info",
      text: "In Transit",
    },
    [StopStatus.enum.Completed]: {
      variant: "active",
      text: "Completed",
    },
    [StopStatus.enum.Canceled]: {
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

export function PTOStatusBadge({
  status,
}: {
  status: WorkerPTOSchema["status"];
}) {
  const ptoStatusAttributes: Record<WorkerPTOSchema["status"], BadgeAttrProps> =
    {
      [PTOStatus.Requested]: {
        variant: "purple",
        text: "Requested",
      },
      [PTOStatus.Approved]: {
        variant: "active",
        text: "Approved",
      },
      [PTOStatus.Rejected]: {
        variant: "inactive",
        text: "Rejected",
      },
      [PTOStatus.Cancelled]: {
        variant: "warning",
        text: "Cancelled",
      },
    };

  return (
    <Badge variant={ptoStatusAttributes[status].variant} className="max-h-6">
      {ptoStatusAttributes[status].text}
    </Badge>
  );
}
export function PTOTypeBadge({ type }: { type: WorkerPTOSchema["type"] }) {
  const ptoTypeAttributes: Record<WorkerPTOSchema["type"], BadgeAttrProps> = {
    [PTOType.Vacation]: {
      variant: "purple",
      text: "Vacation",
    },
    [PTOType.Sick]: {
      variant: "active",
      text: "Sick",
    },
    [PTOType.Holiday]: {
      variant: "inactive",
      text: "Holiday",
    },
    [PTOType.Bereavement]: {
      variant: "warning",
      text: "Bereavement",
    },
    [PTOType.Maternity]: {
      variant: "warning",
      text: "Maternity",
    },
    [PTOType.Paternity]: {
      variant: "warning",
      text: "Paternity",
    },
  };

  return (
    <Badge variant={ptoTypeAttributes[type].variant} className="max-h-6">
      {ptoTypeAttributes[type].text}
    </Badge>
  );
}
export function IntegrationCategoryBadge({
  category,
}: {
  category: IntegrationCategory;
}) {
  const integrationCategoryAttributes: Record<
    IntegrationCategory,
    BadgeAttrProps
  > = {
    [IntegrationCategory.MappingRouting]: {
      variant: "purple",
      text: "Mapping & Routing",
    },
    [IntegrationCategory.FreightLogistics]: {
      variant: "indigo",
      text: "Freight Logistics",
    },
    [IntegrationCategory.Telematics]: {
      variant: "teal",
      text: "Telematics",
    },
  };

  return (
    <Badge
      variant={integrationCategoryAttributes[category].variant}
      className="max-h-6"
      withDot={false}
    >
      {integrationCategoryAttributes[category].text}
    </Badge>
  );
}

export function MoveStatusBadge({ status }: { status: MoveSchema["status"] }) {
  const moveStatusAttributes: Record<MoveSchema["status"], BadgeAttrProps> = {
    [MoveStatus.enum.New]: {
      variant: "purple",
      text: "New",
    },
    [MoveStatus.enum.Assigned]: {
      variant: "warning",
      text: "Assigned",
    },
    [MoveStatus.enum.InTransit]: {
      variant: "info",
      text: "In Transit",
      icon: faSpinner,
    },
    [MoveStatus.enum.Completed]: {
      variant: "active",
      text: "Completed",
      icon: faBadgeCheck,
    },
    [MoveStatus.enum.Canceled]: {
      variant: "inactive",
      text: "Canceled",
      icon: faXmark,
    },
  };

  return (
    <Badge
      withDot={moveStatusAttributes[status].icon ? false : true}
      variant={moveStatusAttributes[status].variant}
      className="max-h-6"
    >
      {moveStatusAttributes[status].icon && (
        <Icon
          icon={moveStatusAttributes[status].icon}
          className={cn(
            "!size-3",
            status === MoveStatus.enum.InTransit &&
              "motion-safe:animate-[spin_0.6s_linear_infinite]",
          )}
        />
      )}
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

export function DocumentStatusBadge({ status }: { status: DocumentStatus }) {
  const statusAttributes: Record<DocumentStatus, BadgeAttrProps> = {
    [DocumentStatus.Draft]: {
      variant: "purple",
      text: "Draft",
    },
    [DocumentStatus.Active]: {
      variant: "active",
      text: "Active",
    },
    [DocumentStatus.Archived]: {
      variant: "inactive",
      text: "Archived",
    },
    [DocumentStatus.Expired]: {
      variant: "warning",
      text: "Expired",
    },
    [DocumentStatus.Rejected]: {
      variant: "warning",
      text: "Rejected",
    },
    [DocumentStatus.PendingApproval]: {
      variant: "orange",
      text: "Pending Approval",
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className="max-h-6">
      {statusAttributes[status].text}
    </Badge>
  );
}

export function ConsolidationStatusBadge({
  status,
}: {
  status: ConsolidationStatus;
}) {
  const statusAttributes: Record<ConsolidationStatus, BadgeAttrProps> = {
    [ConsolidationStatus.New]: {
      variant: "purple",
      text: "New",
    },
    [ConsolidationStatus.InProgress]: {
      variant: "indigo",
      text: "In Progress",
    },
    [ConsolidationStatus.Completed]: {
      variant: "active",
      text: "Completed",
    },
    [ConsolidationStatus.Canceled]: {
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

export function HoldTypeBadge({ type }: { type: ShipmentHoldSchema["type"] }) {
  const typeAttributes: Record<ShipmentHoldSchema["type"], BadgeAttrProps> = {
    [HoldType.enum.OperationalHold]: {
      variant: "purple",
      text: "Operational Hold",
    },
    [HoldType.enum.ComplianceHold]: {
      variant: "indigo",
      text: "Compliance Hold",
    },
    [HoldType.enum.CustomerHold]: {
      variant: "warning",
      text: "Customer Hold",
    },
    [HoldType.enum.FinanceHold]: {
      variant: "active",
      text: "Finance Hold",
    },
  };

  return (
    <Badge variant={typeAttributes[type].variant} className="max-h-6">
      {typeAttributes[type].text}
    </Badge>
  );
}

export function HoldSeverityBadge({
  severity,
}: {
  severity: ShipmentHoldSchema["severity"];
}) {
  const severityAttributes: Record<
    ShipmentHoldSchema["severity"],
    BadgeAttrProps
  > = {
    [HoldSeverity.enum.Informational]: {
      variant: "purple",
      text: "Informational",
      description: "FYI, never blocks ops",
    },
    [HoldSeverity.enum.Advisory]: {
      variant: "indigo",
      text: "Advisory",
      description: "Warns, may block billing, not movement",
    },
    [HoldSeverity.enum.Blocking]: {
      variant: "warning",
      text: "Blocking",
      description: "Can block dispatch/delivery until released",
    },
  };

  return (
    <Badge
      variant={severityAttributes[severity].variant}
      className="max-h-6"
      withDot={false}
    >
      {severityAttributes[severity].text}
    </Badge>
  );
}

export function ShipmentStatusBadge({
  status,
  withDot = false,
  className,
}: {
  status?: ShipmentSchema["status"];
  withDot?: boolean;
  className?: string;
}) {
  if (!status) return null;

  const statusAttributes: Record<ShipmentSchema["status"], BadgeAttrProps> = {
    [ShipmentStatus.enum.New]: {
      variant: "purple",
      text: "New",
      icon: faSparkles,
      description:
        "Shipment has been created and is pending initial assignment.",
    },
    [ShipmentStatus.enum.PartiallyAssigned]: {
      variant: "indigo",
      text: "Partially Assigned",
      description:
        "Equipment or worker assignments are pending for one or more moves within this shipment.",
    },
    [ShipmentStatus.enum.PartiallyCompleted]: {
      variant: "indigo",
      text: "Partially Completed",
      description:
        "Some moves within this shipment have been completed, but not all.",
    },
    [ShipmentStatus.enum.Assigned]: {
      variant: "warning",
      text: "Assigned",
      description:
        "All required equipment and workers have been assigned to this shipment's moves.",
    },
    [ShipmentStatus.enum.InTransit]: {
      variant: "info",
      text: "In Transit",
      icon: faSpinner,
      description:
        "Active shipment with cargo currently in transport between designated locations.",
    },
    [ShipmentStatus.enum.Delayed]: {
      variant: "orange",
      text: "Delayed",
      icon: faClock,
      description:
        "Shipment has exceeded scheduled arrival or delivery timeframes at one or more stops.",
    },
    [ShipmentStatus.enum.Completed]: {
      variant: "active",
      text: "Completed",
      icon: faCheck,
      description:
        "All transportation activities for this shipment have been successfully completed.",
    },
    [ShipmentStatus.enum.Billed]: {
      variant: "teal",
      text: "Billed",
      icon: faFileInvoiceDollar,
      description:
        "Invoice has been generated and posted for completed transportation services.",
    },
    [ShipmentStatus.enum.ReadyToBill]: {
      variant: "pink",
      text: "Ready to Bill",
      icon: faPaperPlane,
      description:
        "All moves within this shipment have been completed, and the shipment is ready to be billed.",
    },
    [ShipmentStatus.enum.Canceled]: {
      variant: "inactive",
      text: "Canceled",
      icon: faCircleXmark,
      description:
        "Shipment has been terminated and will not be completed as originally planned.",
    },
  };

  return (
    <Badge
      withDot={withDot}
      variant={statusAttributes[status].variant}
      className={cn(className, "max-h-6")}
    >
      {statusAttributes[status].icon && (
        <Icon
          icon={statusAttributes[status].icon}
          className={cn(
            "!size-3",
            status === ShipmentStatus.enum.InTransit &&
              "motion-safe:animate-[spin_0.6s_linear_infinite]",
          )}
        />
      )}
      {statusAttributes[status].text}
    </Badge>
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
  status: ShipmentSchema["status"];
}) {
  const statusAttributes: Record<
    ShipmentSchema["status"],
    PlainBadgeAttrProps
  > = {
    [ShipmentStatus.enum.New]: {
      className: "bg-purple-600",
      text: "New",
      description:
        "Shipment has been created and is pending initial assignment.",
    },
    [ShipmentStatus.enum.PartiallyAssigned]: {
      className: "bg-indigo-600",
      text: "Partially Assigned",
      description:
        "Equipment or worker assignments are pending for one or more moves within this shipment.",
    },
    [ShipmentStatus.enum.Assigned]: {
      className: "bg-warning",
      text: "Assigned",
      description:
        "All required equipment and workers have been assigned to this shipment's moves.",
    },
    [ShipmentStatus.enum.InTransit]: {
      className: "bg-blue-600",
      text: "In Transit",
      description:
        "Active shipment with cargo currently in transport between designated locations.",
    },
    [ShipmentStatus.enum.Delayed]: {
      className: "bg-orange-600",
      text: "Delayed",
      description:
        "Shipment has exceeded scheduled arrival or delivery timeframes at one or more stops.",
    },
    [ShipmentStatus.enum.Completed]: {
      className: "bg-green-600",
      text: "Completed",
      description:
        "All transportation activities for this shipment have been successfully completed.",
    },
    [ShipmentStatus.enum.PartiallyCompleted]: {
      className: "bg-indigo-600",
      text: "Partially Completed",
      description:
        "Some moves within this shipment have been completed, but not all.",
    },
    [ShipmentStatus.enum.Billed]: {
      className: "bg-teal-600",
      text: "Billed",
      description:
        "Invoice has been generated and posted for completed transportation services.",
    },
    [ShipmentStatus.enum.ReadyToBill]: {
      className: "bg-teal-600",
      text: "Ready to Bill",
      description:
        "All moves within this shipment have been completed, and the shipment is ready to be billed.",
    },
    [ShipmentStatus.enum.Canceled]: {
      className: "bg-red-600",
      text: "Canceled",
      description:
        "Shipment has been terminated and will not be completed as originally planned.",
    },
  };

  return (
    <div className="flex items-center gap-x-1">
      <div
        className={cn(
          "size-2 mb-0.5 rounded-full",
          statusAttributes[status].className,
        )}
      />
      <p>{statusAttributes[status].text}</p>
    </div>
  );
}

export function PackingGroupBadge({
  group,
  withDot = false,
  className,
}: {
  group: PackingGroupChoiceProps;
  withDot?: boolean;
  className?: string;
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
    <Badge
      variant={packingGroupAttributes[group].variant}
      className={cn(className, "max-h-6")}
      withDot={withDot}
    >
      {packingGroupAttributes[group].text}
    </Badge>
  );
}

export function HazmatBadge({ isHazmat }: { isHazmat: boolean }) {
  return (
    <Badge variant={isHazmat ? "active" : "inactive"} className="max-h-6">
      {isHazmat ? "Hazmat" : "Non-Hazmat"}
    </Badge>
  );
}

export function BooleanBadge({ value }: { value: boolean }) {
  return (
    <Badge variant={value ? "active" : "inactive"} className="max-h-6">
      {value ? "Yes" : "No"}
    </Badge>
  );
}
