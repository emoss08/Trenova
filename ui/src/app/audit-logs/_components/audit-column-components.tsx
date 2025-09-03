/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { BadgeAttrProps } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { Resource } from "@/types/audit-entry";
import { Action } from "@/types/roles-permissions";

export function ActionBadge({
  action,
  withDot,
  className,
}: {
  action?: Action;
  withDot?: boolean;
  className?: string;
}) {
  if (!action) return null;

  const actionAttributes: Record<Action, BadgeAttrProps> = {
    [Action.Create]: {
      variant: "active",
      text: "Created",
    },
    [Action.Update]: {
      variant: "purple",
      text: "Updated",
    },
    [Action.Delete]: {
      variant: "inactive",
      text: "Deleted",
    },
    [Action.Read]: {
      variant: "pink",
      text: "Viewed",
    },
    [Action.Approve]: {
      variant: "active",
      text: "Approved",
    },
    [Action.Classify]: {
      variant: "purple",
      text: "Classified",
    },
    [Action.Release]: {
      variant: "purple",
      text: "Released",
    },
    [Action.Reject]: {
      variant: "inactive",
      text: "Rejected",
    },
    [Action.Submit]: {
      variant: "active",
      text: "Submitted",
    },
    [Action.Cancel]: {
      variant: "inactive",
      text: "Cancelled",
    },
    [Action.Assign]: {
      variant: "active",
      text: "Assigned",
    },
    [Action.Reassign]: {
      variant: "warning",
      text: "Reassigned",
    },
    [Action.Complete]: {
      variant: "active",
      text: "Completed",
    },
    [Action.Duplicate]: {
      variant: "orange",
      text: "Duplicated",
    },
    [Action.ManageDefaults]: {
      variant: "active",
      text: "Managed Defaults",
    },
    [Action.Share]: {
      variant: "warning",
      text: "Shared",
    },
    [Action.Export]: {
      variant: "orange",
      text: "Exported",
    },
    [Action.Import]: {
      variant: "orange",
      text: "Imported",
    },
    [Action.Archive]: {
      variant: "inactive",
      text: "Archived",
    },
    [Action.Restore]: {
      variant: "active",
      text: "Restored",
    },
    [Action.Manage]: {
      variant: "active",
      text: "Managed",
    },
    [Action.Audit]: {
      variant: "active",
      text: "Audited",
    },
    [Action.Delegate]: {
      variant: "warning",
      text: "Delegated",
    },
    [Action.Configure]: {
      variant: "active",
      text: "Configured",
    },
    [Action.Split]: {
      variant: "orange",
      text: "Split",
    },
    [Action.ModifyField]: {
      variant: "purple",
      text: "Modified Field",
    },
    [Action.ViewField]: {
      variant: "pink",
      text: "Viewed Field",
    },
    [Action.ReadyToBill]: {
      variant: "active",
      text: "Ready to Bill",
    },
    [Action.ReleaseToBilling]: {
      variant: "active",
      text: "Released to Billing",
    },
    [Action.BulkTransfer]: {
      variant: "orange",
      text: "Bulk Transfer",
    },
    [Action.ReviewInvoice]: {
      variant: "orange",
      text: "Reviewed Invoice",
    },
    [Action.PostInvoice]: {
      variant: "orange",
      text: "Posted Invoice",
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
    [Resource.ResourceEditor]: {
      variant: "orange",
      text: "Resource Editor",
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
    [Resource.GlAccount]: {
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
