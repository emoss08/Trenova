import { BadgeAttrProps } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { AuditEntryAction, Resource } from "@/types/audit-entry";

export function AuditEntryActionBadge({
  action,
  withDot,
  className,
}: {
  action?: AuditEntryAction;
  withDot?: boolean;
  className?: string;
}) {
  if (!action) return null;

  const actionAttributes: Record<AuditEntryAction, BadgeAttrProps> = {
    [AuditEntryAction.Create]: {
      variant: "active",
      text: "Created",
    },
    [AuditEntryAction.Update]: {
      variant: "purple",
      text: "Updated",
    },
    [AuditEntryAction.Delete]: {
      variant: "inactive",
      text: "Deleted",
    },
    [AuditEntryAction.View]: {
      variant: "pink",
      text: "Viewed",
    },
    [AuditEntryAction.Approve]: {
      variant: "active",
      text: "Approved",
    },
    [AuditEntryAction.Reject]: {
      variant: "inactive",
      text: "Rejected",
    },
    [AuditEntryAction.Submit]: {
      variant: "active",
      text: "Submitted",
    },
    [AuditEntryAction.Cancel]: {
      variant: "inactive",
      text: "Cancelled",
    },
    [AuditEntryAction.Assign]: {
      variant: "active",
      text: "Assigned",
    },
    [AuditEntryAction.Reassign]: {
      variant: "warning",
      text: "Reassigned",
    },
    [AuditEntryAction.Complete]: {
      variant: "active",
      text: "Completed",
    },
    [AuditEntryAction.Duplicate]: {
      variant: "orange",
      text: "Duplicated",
    },
    [AuditEntryAction.ManageDefaults]: {
      variant: "active",
      text: "Managed Defaults",
    },
    [AuditEntryAction.Share]: {
      variant: "warning",
      text: "Shared",
    },
    [AuditEntryAction.Export]: {
      variant: "orange",
      text: "Exported",
    },
    [AuditEntryAction.Import]: {
      variant: "orange",
      text: "Imported",
    },
    [AuditEntryAction.Archive]: {
      variant: "inactive",
      text: "Archived",
    },
    [AuditEntryAction.Restore]: {
      variant: "active",
      text: "Restored",
    },
    [AuditEntryAction.Manage]: {
      variant: "active",
      text: "Managed",
    },
    [AuditEntryAction.Audit]: {
      variant: "active",
      text: "Audited",
    },
    [AuditEntryAction.Delegate]: {
      variant: "warning",
      text: "Delegated",
    },
    [AuditEntryAction.Configure]: {
      variant: "active",
      text: "Configured",
    },
    [AuditEntryAction.Split]: {
      variant: "orange",
      text: "Split",
    },
  };

  return (
    <Badge
      withDot={withDot}
      variant={actionAttributes[action].variant}
      className={cn(className, "max-h-6")}
    >
      {actionAttributes[action].text}
    </Badge>
  );
}

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
    [Resource.HazmatSegregationRule]: {
      variant: "orange",
      text: "Hazmat Segregation Rule",
    },
    [Resource.DocumentQualityConfig]: {
      variant: "orange",
      text: "Document Quality Config",
    },
    // Operations Resources
    [Resource.Worker]: {
      variant: "indigo",
      text: "Worker",
    },
    [Resource.Tractor]: {
      variant: "indigo",
      text: "Tractor",
    },
    [Resource.Trailer]: {
      variant: "indigo",
      text: "Trailer",
    },
    [Resource.Shipment]: {
      variant: "indigo",
      text: "Shipment",
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
    [Resource.AuditEntries]: {
      variant: "pink",
      text: "Audit Log",
    },
    [Resource.TableConfiguration]: {
      variant: "pink",
      text: "Table Configuration",
    },
    [Resource.Integration]: {
      variant: "pink",
      text: "Integration",
    },
    [Resource.Setting]: {
      variant: "warning",
      text: "Setting",
    },
    [Resource.Template]: {
      variant: "warning",
      text: "Template",
    },
    [Resource.Backup]: {
      variant: "warning",
      text: "Backup",
    },
    [Resource.Dashboard]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.BillingManagement]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.BillingClient]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.ConfigurationFiles]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.RateManagement]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.SystemLog]: {
      variant: undefined,
      text: "",
      description: undefined,
    },

    [Resource.DataRetention]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.Equipment]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.Maintenance]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.FormulaTemplate]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.ShipmentManagement]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.Route]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.CommentType]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.DelayCode]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.ChargeType]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.DivisionCode]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.GlAccount]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.RevenueCode]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.AccessorialCharge]: {
      variant: undefined,
      text: "",
      description: undefined,
    },
    [Resource.DocumentClassification]: {
      variant: undefined,
      text: "",
      description: undefined,
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
