/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  HoverCardTimestamp,
  RandomColoredBadge,
} from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { LazyImage } from "@/components/ui/image";
import type { RoleSchema, UserSchema } from "@/lib/schemas/user-schema";
import { RoleType } from "@/types/roles-permissions";
import { type ColumnDef } from "@tanstack/react-table";

export function getUserColumns(): ColumnDef<UserSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <StatusBadge status={status} />;
      },
    },
    {
      id: "fullName",
      accessorKey: "name",
      header: "Full Name",
      minSize: 150,
      cell: ({ row }) => {
        const { profilePicUrl, name, username } = row.original;
        return (
          <div className="flex items-center gap-2">
            <LazyImage
              src={profilePicUrl || `https://avatar.vercel.sh/${name}.svg`}
              alt={name}
              className="size-6 rounded-full"
            />
            <div className="flex flex-col text-left leading-tight">
              <span className="text-sm font-medium">{name}</span>
              <span className="text-2xs text-muted-foreground">{username}</span>
            </div>
          </div>
        );
      },
    },
    {
      accessorKey: "emailAddress",
      header: "Email",
      cell: ({ row }) => {
        // We should do mailto: link here
        const { emailAddress } = row.original;
        return (
          <a
            href={`mailto:${emailAddress}`}
            className="text-sm text-muted-foreground underline hover:text-foreground"
          >
            {emailAddress}
          </a>
        );
      },
    },
    {
      accessorKey: "roles",
      header: "Roles",
      cell: ({ row }) => {
        const { roles } = row.original;
        if (!roles) return <p>-</p>;

        return (
          <div className="flex flex-wrap gap-1">
            {roles.map((role) => (
              <RandomColoredBadge key={role.id}>{role.name}</RandomColoredBadge>
            ))}
          </div>
        );
      },
    },
    {
      id: "lastLoginAt",
      accessorKey: "lastLoginAt",
      minSize: 150,
      header: "Last Login",
      cell: ({ row }) => {
        const { lastLoginAt } = row.original;
        return <HoverCardTimestamp timestamp={lastLoginAt} />;
      },
    },
  ];
}

export function getRoleColumns(): ColumnDef<RoleSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <StatusBadge status={status} />;
      },
    },
    {
      accessorKey: "roleType",
      header: "Type",
      cell: ({ row }) => {
        const { roleType } = row.original;
        return (
          <Badge variant={roleType === RoleType.System ? "indigo" : "orange"}>
            {roleType}
          </Badge>
        );
      },
    },
    {
      accessorKey: "name",
      header: "Name",
    },
    {
      accessorKey: "description",
      header: "Description",
    },
  ];
}
