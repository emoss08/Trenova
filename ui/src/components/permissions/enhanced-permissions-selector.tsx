/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { PermissionSchema } from "@/lib/schemas/user-schema";
import { cn } from "@/lib/utils";
import {
  AlertTriangle,
  BarChart3,
  Calendar,
  Check,
  ChevronDown,
  ChevronRight,
  CreditCard,
  Database,
  Download,
  Edit,
  Eye,
  FileText,
  Filter,
  MapPin,
  Package,
  Plus,
  Search,
  Settings,
  Shield,
  Trash2,
  Truck,
  Upload,
  Users,
  X,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";

interface EnhancedPermissionsSelectorProps {
  permissions: PermissionSchema[];
  selectedPermissions: PermissionSchema[];
  onPermissionsChange: (permissions: PermissionSchema[]) => void;
  isLoading?: boolean;
}

// Icons for different resources - mapping transportation management concepts
const getResourceIcon = (resource: string) => {
  const iconMap: Record<string, any> = {
    shipment: Truck,
    customer: Users,
    document: FileText,
    location: MapPin,
    equipment: Package,
    billing: CreditCard,
    analytics: BarChart3,
    system: Settings,
    user: Users,
    audit: Shield,
    organization: Database,
    tractor: Truck,
    trailer: Package,
    worker: Users,
    commodity: Package,
    fleet: Truck,
    hazmat: AlertTriangle,
    integration: Settings,
    assignment: Calendar,
  };

  const IconComponent = iconMap[resource.toLowerCase()] || FileText;
  return <IconComponent className="h-4 w-4" />;
};

// Action icons and colors
const getActionInfo = (action: string) => {
  const actionMap: Record<string, { icon: any; color: string; label: string }> =
    {
      view: { icon: Eye, color: "text-blue-600", label: "View" },
      create: { icon: Plus, color: "text-green-600", label: "Create" },
      update: { icon: Edit, color: "text-yellow-600", label: "Edit" },
      delete: { icon: Trash2, color: "text-red-600", label: "Delete" },
      manage: { icon: Shield, color: "text-purple-600", label: "Full Access" },
      export: { icon: Download, color: "text-indigo-600", label: "Export" },
      import: { icon: Upload, color: "text-indigo-600", label: "Import" },
    };

  return (
    actionMap[action.toLowerCase()] || {
      icon: FileText,
      color: "text-gray-600",
      label: action?.toUpperCase() || "UNKNOWN",
    }
  );
};

export function EnhancedPermissionsSelector({
  permissions,
  selectedPermissions,
  onPermissionsChange,
  isLoading = false,
}: EnhancedPermissionsSelectorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [filterResource, setFilterResource] = useState<string>("all");
  const [filterAction, setFilterAction] = useState<string>("all");
  const [activeTab, setActiveTab] = useState("browse");
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
    new Set(),
  );

  // Organize permissions by resource
  const groupedPermissions = useMemo(() => {
    if (!permissions?.length) return {};

    return permissions.reduce(
      (acc, permission) => {
        const resource = permission.resource || "Other";
        if (!acc[resource]) {
          acc[resource] = [];
        }
        acc[resource].push(permission);
        return acc;
      },
      {} as Record<string, PermissionSchema[]>,
    );
  }, [permissions]);

  // Get unique resources and actions for filtering
  const uniqueResources = useMemo(
    () => Array.from(new Set(permissions.map((p) => p.resource || "Other"))),
    [permissions],
  );

  const uniqueActions = useMemo(
    () => Array.from(new Set(permissions.map((p) => p.action).filter(Boolean))),
    [permissions],
  );

  // Filter permissions based on search and filters
  const filteredGroupedPermissions = useMemo(() => {
    const filtered: Record<string, PermissionSchema[]> = {};

    Object.entries(groupedPermissions).forEach(([resource, perms]) => {
      const filteredPerms = perms.filter((permission) => {
        const matchesSearch =
          searchQuery === "" ||
          permission.action
            ?.toLowerCase()
            .includes(searchQuery.toLowerCase()) ||
          permission.description
            ?.toLowerCase()
            .includes(searchQuery.toLowerCase()) ||
          permission.resource
            ?.toLowerCase()
            .includes(searchQuery.toLowerCase());

        const matchesResource =
          filterResource === "all" || permission.resource === filterResource;
        const matchesAction =
          filterAction === "all" || permission.action === filterAction;

        return matchesSearch && matchesResource && matchesAction;
      });

      if (filteredPerms.length > 0) {
        filtered[resource] = filteredPerms;
      }
    });

    return filtered;
  }, [groupedPermissions, searchQuery, filterResource, filterAction]);

  // Helper functions
  const isPermissionSelected = useCallback(
    (permissionId: string) => {
      return selectedPermissions.some((p) => p.id === permissionId);
    },
    [selectedPermissions],
  );

  const togglePermission = useCallback(
    (permission: PermissionSchema, checked: boolean) => {
      if (checked) {
        onPermissionsChange([...selectedPermissions, permission]);
      } else {
        onPermissionsChange(
          selectedPermissions.filter((p) => p.id !== permission.id),
        );
      }
    },
    [selectedPermissions, onPermissionsChange],
  );

  const toggleCategoryPermissions = useCallback(
    (categoryPerms: PermissionSchema[], allSelected: boolean) => {
      if (allSelected) {
        // Remove all permissions in this category
        const categoryIds = categoryPerms.map((p) => p.id);
        onPermissionsChange(
          selectedPermissions.filter((p) => !categoryIds.includes(p.id)),
        );
      } else {
        // Add all permissions in this category
        const newPerms = categoryPerms.filter(
          (p) => !isPermissionSelected(p.id!),
        );
        onPermissionsChange([...selectedPermissions, ...newPerms]);
      }
    },
    [selectedPermissions, onPermissionsChange, isPermissionSelected],
  );

  const toggleManagePermission = useCallback(
    (categoryPerms: PermissionSchema[]) => {
      const managePermission = categoryPerms.find((p) => p.action === "manage");
      if (!managePermission) return;

      const isCurrentlySelected = isPermissionSelected(managePermission.id!);

      if (isCurrentlySelected) {
        // Remove manage permission
        onPermissionsChange(
          selectedPermissions.filter((p) => p.id !== managePermission.id),
        );
      } else {
        // Add manage permission and remove other permissions in this category
        const categoryIds = categoryPerms.map((p) => p.id);
        const filteredSelected = selectedPermissions.filter(
          (p) => !categoryIds.includes(p.id),
        );
        onPermissionsChange([...filteredSelected, managePermission]);
      }
    },
    [selectedPermissions, onPermissionsChange, isPermissionSelected],
  );

  const removePermission = useCallback(
    (permissionId: string) => {
      onPermissionsChange(
        selectedPermissions.filter((p) => p.id !== permissionId),
      );
    },
    [selectedPermissions, onPermissionsChange],
  );

  const clearAllPermissions = useCallback(() => {
    onPermissionsChange([]);
  }, [onPermissionsChange]);

  const toggleCategoryExpansion = useCallback(
    (resource: string) => {
      const newExpanded = new Set(expandedCategories);
      if (newExpanded.has(resource)) {
        newExpanded.delete(resource);
      } else {
        newExpanded.add(resource);
      }
      setExpandedCategories(newExpanded);
    },
    [expandedCategories],
  );

  const getCategoryStats = useCallback(
    (categoryPerms: PermissionSchema[]) => {
      const selectedCount = categoryPerms.filter((p) =>
        isPermissionSelected(p.id!),
      ).length;
      const totalCount = categoryPerms.length;
      const hasManage = categoryPerms.some(
        (p) => p.action === "manage" && isPermissionSelected(p.id!),
      );

      return {
        selectedCount,
        totalCount,
        hasManage,
        allSelected: selectedCount === totalCount,
      };
    },
    [isPermissionSelected],
  );

  // Group selected permissions by resource for review tab
  const selectedPermissionsByResource = useMemo(() => {
    const grouped: Record<string, PermissionSchema[]> = {};
    selectedPermissions.forEach((permission) => {
      const resource = permission.resource || "Other";
      if (!grouped[resource]) {
        grouped[resource] = [];
      }
      grouped[resource].push(permission);
    });
    return grouped;
  }, [selectedPermissions]);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Loading Permissions...
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Role Permissions
          </CardTitle>
          <div className="flex items-center gap-2">
            <Badge variant="secondary">
              {selectedPermissions.length} selected
            </Badge>
            {selectedPermissions.length > 0 && (
              <Button
                variant="outline"
                size="sm"
                onClick={clearAllPermissions}
                className="text-xs"
              >
                <X className="h-3 w-3 mr-1" />
                Clear All
              </Button>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="browse">Browse & Select</TabsTrigger>
            <TabsTrigger value="review">Review Selected</TabsTrigger>
          </TabsList>

          <TabsContent value="browse" className="space-y-4 mt-4">
            {/* Search and Filter Controls */}
            <div className="flex flex-col sm:flex-row gap-3">
              <div className="relative flex-1">
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search permissions..."
                  className="pl-8"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
              <Select value={filterResource} onValueChange={setFilterResource}>
                <SelectTrigger className="w-full sm:w-[200px]">
                  <Filter className="h-4 w-4 mr-2" />
                  <SelectValue placeholder="All Resources" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Resources</SelectItem>
                  {uniqueResources.map((resource) => (
                    <SelectItem key={resource} value={resource}>
                      {resource
                        .replace(/_/g, " ")
                        .replace(/\b\w/g, (l) => l.toUpperCase())}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={filterAction} onValueChange={setFilterAction}>
                <SelectTrigger className="w-full sm:w-[160px]">
                  <SelectValue placeholder="All Actions" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Actions</SelectItem>
                  {uniqueActions.map((action) => (
                    <SelectItem key={action} value={action || "all"}>
                      {getActionInfo(action || "all").label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Permissions Grid */}
            <ScrollArea className="h-[400px] rounded-md border">
              <div className="p-4 space-y-4">
                {Object.entries(filteredGroupedPermissions).map(
                  ([resource, categoryPerms]) => {
                    const {
                      selectedCount,
                      totalCount,
                      hasManage,
                      allSelected,
                    } = getCategoryStats(categoryPerms);
                    const isExpanded = expandedCategories.has(resource);
                    const managePermission = categoryPerms.find(
                      (p) => p.action === "manage",
                    );

                    return (
                      <div key={resource} className="overflow-hidden">
                        <Collapsible
                          open={isExpanded}
                          onOpenChange={() => toggleCategoryExpansion(resource)}
                        >
                          <CollapsibleTrigger asChild>
                            <div className="cursor-pointer hover:bg-muted/50 transition-colors p-3 rounded-md">
                              <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                  {getResourceIcon(resource)}
                                  <div>
                                    <h4 className="font-semibold">
                                      {resource
                                        .replace(/_/g, " ")
                                        .replace(/\b\w/g, (l) =>
                                          l.toUpperCase(),
                                        )}
                                    </h4>
                                    <p className="text-sm text-muted-foreground">
                                      {selectedCount}/{totalCount} permissions
                                      selected
                                    </p>
                                  </div>
                                </div>
                                <div className="flex items-center gap-2">
                                  {hasManage && (
                                    <Badge
                                      variant="default"
                                      className="text-xs"
                                    >
                                      Full Access
                                    </Badge>
                                  )}
                                  <Badge
                                    variant={
                                      selectedCount > 0
                                        ? "default"
                                        : "secondary"
                                    }
                                    className="text-xs"
                                  >
                                    {selectedCount}/{totalCount}
                                  </Badge>
                                  {isExpanded ? (
                                    <ChevronDown className="h-4 w-4" />
                                  ) : (
                                    <ChevronRight className="h-4 w-4" />
                                  )}
                                </div>
                              </div>
                            </div>
                          </CollapsibleTrigger>
                          <CollapsibleContent>
                            <div className="p-3">
                              {/* Category Actions */}
                              <div className="flex flex-wrap gap-2 mb-4">
                                {managePermission && (
                                  <Button
                                    type="button"
                                    variant={hasManage ? "default" : "outline"}
                                    size="sm"
                                    onClick={() =>
                                      toggleManagePermission(categoryPerms)
                                    }
                                    className="text-xs"
                                  >
                                    <Shield className="h-3 w-3 mr-1" />
                                    Full Access
                                  </Button>
                                )}
                                <Button
                                  type="button"
                                  variant="outline"
                                  size="sm"
                                  onClick={() =>
                                    toggleCategoryPermissions(
                                      categoryPerms,
                                      allSelected,
                                    )
                                  }
                                  className="text-xs"
                                >
                                  {allSelected ? (
                                    <>
                                      <X className="h-3 w-3 mr-1" />
                                      Deselect All
                                    </>
                                  ) : (
                                    <>
                                      <Check className="h-3 w-3 mr-1" />
                                      Select All
                                    </>
                                  )}
                                </Button>
                              </div>

                              <Separator className="mb-4" />

                              {/* Individual Permissions */}
                              <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                                {categoryPerms
                                  .filter((p) => p.action !== "manage")
                                  .map((permission) => {
                                    const actionInfo = getActionInfo(
                                      permission.action || "",
                                    );
                                    const Icon = actionInfo.icon;
                                    const isSelected = isPermissionSelected(
                                      permission.id!,
                                    );

                                    return (
                                      <div
                                        key={permission.id}
                                        className={cn(
                                          "flex items-start gap-3 p-3 rounded-lg border transition-all",
                                          "hover:bg-accent/50 cursor-pointer",
                                          isSelected &&
                                            "border-blue-600 outline-hidden ring-4 ring-blue-600/20",
                                          hasManage && "opacity-50",
                                        )}
                                        onClick={() =>
                                          !hasManage &&
                                          togglePermission(
                                            permission,
                                            !isSelected,
                                          )
                                        }
                                      >
                                        <Checkbox
                                          checked={isSelected}
                                          disabled={hasManage}
                                          onChange={() => {}}
                                          className="mt-0.5"
                                        />
                                        <div className="flex-1 space-y-1">
                                          <div className="flex items-center gap-2">
                                            <Icon
                                              className={cn(
                                                "h-4 w-4",
                                                actionInfo.color,
                                              )}
                                            />
                                            <span className="font-medium text-sm">
                                              {actionInfo.label}
                                            </span>
                                          </div>
                                          <p className="text-xs text-muted-foreground">
                                            {permission.description ||
                                              `${actionInfo.label} access to ${resource.toLowerCase()}`}
                                          </p>
                                        </div>
                                      </div>
                                    );
                                  })}
                              </div>
                            </div>
                          </CollapsibleContent>
                        </Collapsible>
                      </div>
                    );
                  },
                )}
              </div>
            </ScrollArea>
          </TabsContent>

          <TabsContent value="review" className="mt-4">
            {selectedPermissions.length === 0 ? (
              <div className="text-center py-12">
                <Shield className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <h3 className="text-lg font-semibold mb-2">
                  No permissions selected
                </h3>
                <p className="text-muted-foreground mb-4">
                  Switch to the Browse tab to select permissions for this role.
                </p>
                <Button
                  variant="outline"
                  onClick={() => setActiveTab("browse")}
                >
                  Browse Permissions
                </Button>
              </div>
            ) : (
              <ScrollArea className="h-[500px]">
                <div className="space-y-6">
                  {Object.entries(selectedPermissionsByResource).map(
                    ([resource, perms]) => (
                      <Card key={resource}>
                        <CardHeader className="pb-3">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              {getResourceIcon(resource)}
                              <h4 className="font-semibold">
                                {resource
                                  .replace(/_/g, " ")
                                  .replace(/\b\w/g, (l) => l.toUpperCase())}
                              </h4>
                            </div>
                            <Badge variant="outline">{perms.length}</Badge>
                          </div>
                        </CardHeader>
                        <CardContent className="pt-0">
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                            {perms.map((permission) => {
                              const actionInfo = getActionInfo(
                                permission.action || "",
                              );
                              const Icon = actionInfo.icon;

                              return (
                                <div
                                  key={permission.id}
                                  className="flex items-center justify-between p-3 rounded-lg border bg-muted/20"
                                >
                                  <div className="flex items-center gap-2">
                                    <Icon
                                      className={cn(
                                        "h-4 w-4",
                                        actionInfo.color,
                                      )}
                                    />
                                    <div>
                                      <span className="font-medium text-sm">
                                        {actionInfo.label}
                                      </span>
                                      {permission.description && (
                                        <p className="text-xs text-muted-foreground">
                                          {permission.description}
                                        </p>
                                      )}
                                    </div>
                                  </div>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() =>
                                      removePermission(permission.id!)
                                    }
                                    className="h-8 w-8 p-0"
                                  >
                                    <X className="h-4 w-4" />
                                  </Button>
                                </div>
                              );
                            })}
                          </div>
                        </CardContent>
                      </Card>
                    ),
                  )}
                </div>
              </ScrollArea>
            )}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
