import { BadgeAttrProps } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import {
  PermissionOperation,
  PermissionOperations,
} from "@/types/_gen/permissions";
import { Resource } from "@/types/audit-entry";

export function ActionBadge({
  operation,
  withDot,
  className,
}: {
  operation?: PermissionOperation;
  withDot?: boolean;
  className?: string;
}) {
  if (!operation) return null;

  const actionAttributes: Record<PermissionOperation, BadgeAttrProps> = {
    [PermissionOperations.CREATE]: {
      variant: "active",
      text: "Created",
    },
    [PermissionOperations.UPDATE]: {
      variant: "purple",
      text: "Updated",
    },
    [PermissionOperations.SUBMIT]: {
      variant: "purple",
      text: "Submitted",
    },
    [PermissionOperations.DELETE]: {
      variant: "inactive",
      text: "Deleted",
    },
    [PermissionOperations.READ]: {
      variant: "pink",
      text: "Viewed",
    },
    [PermissionOperations.EXPORT]: {
      variant: "pink",
      text: "Exported",
    },
    [PermissionOperations.IMPORT]: {
      variant: "pink",
      text: "Imported",
    },
    [PermissionOperations.DUPLICATE]: {
      variant: "pink",
      text: "Duplicated",
    },
    [PermissionOperations.CLOSE]: {
      variant: "inactive",
      text: "Closed",
    },
    [PermissionOperations.LOCK]: {
      variant: "warning",
      text: "Locked",
    },
    [PermissionOperations.UNLOCK]: {
      variant: "active",
      text: "Unlocked",
    },
    [PermissionOperations.ACTIVATE]: {
      variant: "active",
      text: "Activated",
    },
    [PermissionOperations.APPROVE]: {
      variant: "active",
      text: "Approved",
    },
    [PermissionOperations.REJECT]: {
      variant: "inactive",
      text: "Rejected",
    },
    [PermissionOperations.ASSIGN]: {
      variant: "info",
      text: "Assigned",
    },
  };

  return (
    <Badge
      withDot={withDot}
      variant={actionAttributes[operation].variant}
      className={cn(className, "max-h-6")}
    >
      {actionAttributes[operation].text}
    </Badge>
  );
}

// TODO(Wolfred): We need to figure out how to auto-generate this.
export function AuditEntryResourceBadge({
  resource,
  withDot,
  className,
}: {
  resource?: Resource;
  withDot?: boolean;
  className?: string;
}) {
  if (!resource) return null;

  const resourceAttributes: Record<Resource, BadgeAttrProps> = {
    // Core Resources
    [Resource.User]: {
      variant: "orange",
      text: "User",
    },
    [Resource.EmailProfile]: {
      variant: "orange",
      text: "Email Profile",
    },
    [Resource.BusinessUnit]: {
      variant: "orange",
      text: "Business Unit",
    },
    [Resource.Organization]: {
      variant: "orange",
      text: "Organization",
    },
    [Resource.ShipmentControl]: {
      variant: "orange",
      text: "Shipment Control",
    },
    [Resource.BillingControl]: {
      variant: "orange",
      text: "Billing Control",
    },
    [Resource.HazmatSegregationRule]: {
      variant: "orange",
      text: "Hazmat Segregation Rule",
    },
    [Resource.DocumentQualityConfig]: {
      variant: "orange",
      text: "Document Quality Config",
    },
    [Resource.AILog]: {
      variant: "orange",
      text: "AI Log",
    },
    [Resource.Variable]: {
      variant: "orange",
      text: "Variable",
    },
    [Resource.VariableFormat]: {
      variant: "orange",
      text: "Format",
    },
    [Resource.ResourceEditor]: {
      variant: "orange",
      text: "Resource Editor",
    },
    [Resource.FiscalYear]: {
      variant: "orange",
      text: "Fiscal Year",
    },
    [Resource.DispatchControl]: {
      variant: "orange",
      text: "Dispatch Control",
    },
    [Resource.DistanceOverride]: {
      variant: "indigo",
      text: "Distance Override",
    },
    [Resource.AccountType]: {
      variant: "indigo",
      text: "Account Type",
    },
    // Operations Resources
    [Resource.Worker]: {
      variant: "indigo",
      text: "Worker",
    },
    [Resource.WorkerPTO]: {
      variant: "orange",
      text: "Worker PTO",
    },
    [Resource.DedicatedLaneSuggestion]: {
      variant: "orange",
      text: "Dedicated Lane Suggestion",
    },
    [Resource.Tractor]: {
      variant: "indigo",
      text: "Tractor",
    },
    [Resource.Trailer]: {
      variant: "indigo",
      text: "Trailer",
    },
    [Resource.AIClassification]: {
      variant: "purple",
      text: "AI Classification",
    },
    [Resource.Docker]: {
      variant: "indigo",
      text: "Docker",
    },
    [Resource.Shipment]: {
      variant: "indigo",
      text: "Shipment",
    },
    [Resource.Consolidation]: {
      variant: "indigo",
      text: "Shipment",
    },
    [Resource.ConsolidationSettings]: {
      variant: "indigo",
      text: "Consolidation Group",
    },
    [Resource.DocumentType]: {
      variant: "indigo",
      text: "Document Type",
    },
    [Resource.Assignment]: {
      variant: "indigo",
      text: "Assignment",
    },
    [Resource.ShipmentMove]: {
      variant: "indigo",
      text: "Shipment Move",
    },
    [Resource.Stop]: {
      variant: "indigo",
      text: "Stop",
    },
    [Resource.FleetCode]: {
      variant: "indigo",
      text: "Fleet Code",
    },
    [Resource.Document]: {
      variant: "indigo",
      text: "Document",
    },
    [Resource.ShipmentComment]: {
      variant: "indigo",
      text: "Shipment Comment",
    },
    [Resource.EquipmentType]: {
      variant: "indigo",
      text: "Equipment Type",
    },
    [Resource.EquipmentManufacturer]: {
      variant: "indigo",
      text: "Equipment Manufacturer",
    },
    [Resource.ShipmentType]: {
      variant: "indigo",
      text: "Shipment Type",
    },
    [Resource.ServiceType]: {
      variant: "indigo",
      text: "Service Type",
    },
    [Resource.LocationCategory]: {
      variant: "indigo",
      text: "Location Category",
    },
    [Resource.AccessorialCharge]: {
      variant: "indigo",
      text: "Accessorial Charge",
    },
    [Resource.Location]: {
      variant: "indigo",
      text: "Location",
    },
    [Resource.Customer]: {
      variant: "indigo",
      text: "Customer",
    },
    [Resource.Invoice]: {
      variant: "indigo",
      text: "Invoice",
    },
    [Resource.Dispatch]: {
      variant: "indigo",
      text: "Dispatch",
    },
    // Commodity Resource
    [Resource.HazardousMaterial]: {
      variant: "inactive",
      text: "Hazardous Material",
    },
    [Resource.Commodity]: {
      variant: "inactive",
      text: "Commodity",
    },
    [Resource.Report]: {
      variant: "pink",
      text: "Report",
    },
    [Resource.AuditEntry]: {
      variant: "pink",
      text: "Audit Entry",
    },
    [Resource.TableConfiguration]: {
      variant: "pink",
      text: "Table Configuration",
    },
    [Resource.Integration]: {
      variant: "pink",
      text: "Integration",
    },
    [Resource.ShipmentHold]: {
      variant: "pink",
      text: "Shipment Hold",
    },
    [Resource.HoldReason]: {
      variant: "pink",
      text: "Hold Reason",
    },
    [Resource.Setting]: {
      variant: "warning",
      text: "Setting",
    },
    [Resource.AccountingControl]: {
      variant: "warning",
      text: "Accounting Control",
    },
    [Resource.FiscalPeriod]: {
      variant: "warning",
      text: "Fiscal Period",
    },
    [Resource.Template]: {
      variant: "warning",
      text: "Template",
    },
    [Resource.Backup]: {
      variant: "warning",
      text: "Backup",
    },
    [Resource.PageFavorite]: {
      variant: "warning",
      text: "Page Favorite",
    },
    [Resource.Dashboard]: {
      variant: undefined,
      text: "Dashboard",
    },
    [Resource.BillingManagement]: {
      variant: undefined,
      text: "Billing Management",
    },
    [Resource.BillingClient]: {
      variant: undefined,
      text: "Billing Client",
    },
    [Resource.ConfigurationFiles]: {
      variant: undefined,
      text: "Configuration Files",
    },
    [Resource.RateManagement]: {
      variant: undefined,
      text: "Rate Management",
    },
    [Resource.SystemLog]: {
      variant: undefined,
      text: "System Log",
    },
    [Resource.DataRetention]: {
      variant: undefined,
      text: "Data Retention",
    },
    [Resource.Equipment]: {
      variant: undefined,
      text: "Equipment",
    },
    [Resource.Maintenance]: {
      variant: undefined,
      text: "Maintenance",
    },
    [Resource.FormulaTemplate]: {
      variant: undefined,
      text: "Formula Template",
    },
    [Resource.ShipmentManagement]: {
      variant: undefined,
      text: "Shipment Management",
    },
    [Resource.DelayCode]: {
      variant: undefined,
      text: "Delay Code",
    },
    [Resource.ChargeType]: {
      variant: undefined,
      text: "Charge Type",
    },
    [Resource.DivisionCode]: {
      variant: undefined,
      text: "Division Code",
    },
    [Resource.GLAccount]: {
      variant: undefined,
      text: "GL Account",
    },
    [Resource.RevenueCode]: {
      variant: undefined,
      text: "Revenue Code",
    },
    [Resource.PatternConfig]: {
      variant: "orange",
      text: "Pattern Config",
    },
    [Resource.GoogleMapsConfig]: {
      variant: "orange",
      text: "Google Maps Config",
    },
    [Resource.DedicatedLane]: {
      variant: "orange",
      text: "Dedicated Lane",
    },
    [Resource.Role]: {
      variant: "orange",
      text: "Role",
    },
    [Resource.BillingQueue]: {
      variant: "indigo",
      text: "Billing Queue",
    },
    [Resource.AuditLog]: {
      variant: "pink",
      text: "Audit Log",
    },
    [Resource.Permission]: {
      variant: "warning",
      text: "Permission",
    },
  };

  return (
    <Badge
      withDot={withDot}
      variant={resourceAttributes[resource].variant}
      className={cn(className, "max-h-6")}
    >
      {resourceAttributes[resource].text}
    </Badge>
  );
}
