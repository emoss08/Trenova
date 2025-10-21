/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// Get filter options for a specific filter type, considering the entity type
export function getFilterOptions(
  filterType: string,
  entityType: string = "all",
) {
  // Common filters that apply across multiple entity types
  const commonFilters = {
    active: { value: "active", label: "Active" },
    inactive: { value: "inactive", label: "Inactive" },
    pending: { value: "pending", label: "Pending" },
    completed: { value: "completed", label: "Completed" },
    cancelled: { value: "cancelled", label: "Cancelled" },
  };

  // First check entity-specific filter options
  switch (entityType) {
    case "shipments":
      switch (filterType) {
        case "status":
          return [
            { value: "New", label: "New" },
            { value: "PartiallyAssigned", label: "Partially Assigned" },
            { value: "Assigned", label: "Assigned" },
            { value: "InTransit", label: "In Transit" },
            { value: "Delayed", label: "Delayed" },
            { value: "PartiallyCompleted", label: "Partially Completed" },
            { value: "Completed", label: "Completed" },
            { value: "Billed", label: "Billed" },
            { value: "Canceled", label: "Canceled" },
          ];
        case "priority":
          return [
            { value: "high", label: "High" },
            { value: "medium", label: "Medium" },
            { value: "low", label: "Low" },
          ];
        case "date":
          return [
            { value: "today", label: "Today" },
            { value: "this_week", label: "This Week" },
            { value: "this_month", label: "This Month" },
            { value: "custom", label: "Custom Range" },
          ];
        case "customer":
          return [
            { value: "retail", label: "Retail" },
            { value: "wholesale", label: "Wholesale" },
            { value: "direct", label: "Direct" },
            { value: "contract", label: "Contract" },
          ];
      }
      break;

    case "workers":
      switch (filterType) {
        case "status":
          return [
            { value: "on_duty", label: "On Duty" },
            { value: "off_duty", label: "Off Duty" },
            { value: "driving", label: "Driving" },
            { value: "rest", label: "Rest" },
            { value: "vacation", label: "Vacation" },
            { value: "sick", label: "Sick" },
            { value: "training", label: "Training" },
          ];
        case "availability":
          return [
            { value: "available", label: "Available" },
            { value: "unavailable", label: "Unavailable" },
            { value: "limited", label: "Limited Hours" },
          ];
        case "type":
          return [
            { value: "driver", label: "Driver" },
            { value: "technician", label: "Technician" },
            { value: "supervisor", label: "Supervisor" },
            { value: "admin", label: "Admin" },
          ];
        case "license":
          return [
            { value: "cdl_a", label: "CDL-A" },
            { value: "cdl_b", label: "CDL-B" },
            { value: "cdl_c", label: "CDL-C" },
            { value: "hazmat", label: "HAZMAT" },
          ];
      }
      break;

    case "equipment":
      switch (filterType) {
        case "status":
          return [
            { value: "available", label: "Available" },
            { value: "in_use", label: "In Use" },
            { value: "maintenance", label: "Maintenance" },
            { value: "out_of_service", label: "Out of Service" },
            { value: "scheduled", label: "Scheduled" },
          ];
        case "type":
          return [
            { value: "tractor", label: "Tractor" },
            { value: "trailer", label: "Trailer" },
            { value: "container", label: "Container" },
            { value: "chassis", label: "Chassis" },
            { value: "specialized", label: "Specialized" },
          ];
        case "maintenance":
          return [
            { value: "up_to_date", label: "Up to Date" },
            { value: "due_soon", label: "Due Soon" },
            { value: "overdue", label: "Overdue" },
          ];
        case "ownership":
          return [
            { value: "owned", label: "Owned" },
            { value: "leased", label: "Leased" },
            { value: "contracted", label: "Contracted" },
          ];
      }
      break;
  }

  // Fall back to generic filter types if no entity-specific ones defined
  switch (filterType) {
    case "status":
      return [
        commonFilters.active,
        commonFilters.inactive,
        commonFilters.pending,
        commonFilters.completed,
        commonFilters.cancelled,
      ];
    case "type":
      return [
        { value: "basic", label: "Basic" },
        { value: "standard", label: "Standard" },
        { value: "premium", label: "Premium" },
        { value: "custom", label: "Custom" },
      ];
    case "availability":
      return [
        { value: "available", label: "Available" },
        { value: "unavailable", label: "Unavailable" },
        { value: "limited", label: "Limited" },
      ];
    case "date":
      return [
        { value: "today", label: "Today" },
        { value: "this_week", label: "This Week" },
        { value: "this_month", label: "This Month" },
        { value: "custom", label: "Custom Range" },
      ];
    default:
      return [];
  }
}

// Updated SearchInputWithBadges component to use the enhanced getFilterOptions function
export function getEntitySpecificFilterOptions(
  filterType: string,
  entityType: string,
) {
  return getFilterOptions(filterType, entityType);
}
