import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { listRoles } from "@/lib/role-api";
import { cn } from "@/lib/utils";
import type { Role } from "@/types/role";
import { useQuery } from "@tanstack/react-query";
import { LockIcon, SearchIcon, ShieldCheckIcon } from "lucide-react";
import { useMemo, useState } from "react";

type UserRolesSectionProps = {
  selectedRoleIds: string[];
  onRoleIdsChange: (roleIds: string[]) => void;
};

export function UserRolesSection({
  selectedRoleIds,
  onRoleIdsChange,
}: UserRolesSectionProps) {
  const [searchQuery, setSearchQuery] = useState("");

  const { data: rolesResponse, isLoading } = useQuery({
    queryKey: ["roles-for-user"],
    queryFn: () => listRoles({ limit: 100, includeSystem: true }),
  });

  const roles = useMemo(() => rolesResponse?.results ?? [], [rolesResponse]);

  const filteredRoles = useMemo(() => {
    if (!searchQuery.trim()) return roles;
    const query = searchQuery.toLowerCase();
    return roles.filter(
      (role) =>
        role.name.toLowerCase().includes(query) ||
        role.description?.toLowerCase().includes(query),
    );
  }, [roles, searchQuery]);

  const handleToggleRole = (roleId: string) => {
    if (selectedRoleIds.includes(roleId)) {
      onRoleIdsChange(selectedRoleIds.filter((id) => id !== roleId));
    } else {
      onRoleIdsChange([...selectedRoleIds, roleId]);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col gap-4">
      <div className="relative">
        <SearchIcon className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search roles..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="h-9 pl-9 text-sm"
        />
      </div>

      {selectedRoleIds.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {selectedRoleIds.map((roleId) => {
            const role = roles.find((r) => r.id === roleId);
            if (!role) return null;
            return (
              <Badge key={roleId} variant="secondary" className="gap-1 pr-1.5">
                {role.name}
                <button
                  type="button"
                  onClick={() => handleToggleRole(roleId)}
                  className="ml-1 rounded-full p-0.5 hover:bg-muted-foreground/20"
                >
                  <span className="sr-only">Remove</span>
                  <svg
                    className="size-3"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </Badge>
            );
          })}
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-auto rounded-lg border">
        {filteredRoles.length === 0 ? (
          <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
            {searchQuery ? "No roles found" : "No roles available"}
          </div>
        ) : (
          <div className="divide-y">
            {filteredRoles.map((role) => (
              <RoleRow
                key={role.id}
                role={role}
                isSelected={selectedRoleIds.includes(role.id!)}
                onToggle={() => handleToggleRole(role.id!)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

type RoleRowProps = {
  role: Role;
  isSelected: boolean;
  onToggle: () => void;
};

function RoleRow({ role, isSelected, onToggle }: RoleRowProps) {
  return (
    <label
      className={cn(
        "flex cursor-pointer items-start gap-3 p-3 transition-colors hover:bg-muted/50",
        isSelected && "bg-primary/5",
      )}
    >
      <Checkbox
        checked={isSelected}
        onCheckedChange={onToggle}
        className="mt-0.5"
      />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{role.name}</span>
          {role.isSystem && (
            <Tooltip>
              <TooltipTrigger>
                <Badge variant="outline" className="gap-1 text-[10px]">
                  <LockIcon className="size-2.5" />
                  System
                </Badge>
              </TooltipTrigger>
              <TooltipContent>System-managed role</TooltipContent>
            </Tooltip>
          )}
          {role.isOrgAdmin && (
            <Tooltip>
              <TooltipTrigger>
                <Badge className="gap-1 bg-amber-600 text-[10px] hover:bg-amber-700">
                  <ShieldCheckIcon className="size-2.5" />
                  Admin
                </Badge>
              </TooltipTrigger>
              <TooltipContent>Organization administrator</TooltipContent>
            </Tooltip>
          )}
        </div>
        {role.description && (
          <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
            {role.description}
          </p>
        )}
      </div>
    </label>
  );
}
