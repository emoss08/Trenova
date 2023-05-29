/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */
import React from "react";
import { ColumnDef } from "@tanstack/react-table";
import { User } from "@/types/user";
import Badge from "./badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Menu, User as UserIcon, UserCog, UserMinus } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { formatDate, formatDateToHumanReadable } from "@/utils/date";
import { ViewUserDialog } from "@/components/user-management/view-user-dialog";
import { EditUserDialog } from "@/components/user-management/edit-user-dialog";

export const usersColumn: ColumnDef<User>[] = [
  {
    id: "user-details",
    header: "User Details",
    cell: ({ row }) => {
      const user = row.original as User;
      const firstName = user.profile?.first_name ?? "-";
      const lastName = user.profile?.last_name ?? "-";
      const usernameInitial = user.username
        ? user.username.charAt(0).toUpperCase()
        : "";
      return (
        <div className="flex items-center">
          <div>
            <Avatar>
              <AvatarImage src={user.profile?.profile_picture} />
              <AvatarFallback>{usernameInitial}</AvatarFallback>
            </Avatar>
          </div>
          <div className="ml-2 flex flex-col">
            <p className="text-sm font-medium">{`${firstName} ${lastName}`}</p>
            <p className="text-sm text-muted-foreground">{user.username}</p>
          </div>
        </div>
      );
    },
  },
  {
    id: "is_active",
    header: "Status",
    cell: ({ row }) => {
      const user = row.original as User;
      return <Badge active={user.is_active} />;
    },
  },
  {
    accessorKey: "email",
    header: "Email",
  },
  {
    id: "date_joined",
    header: "Date Joined",
    cell: ({ row }) => {
      const user = row.original as User;
      const dateJoined = user.date_joined;

      if (dateJoined) {
        const formattedDate = formatDate(dateJoined);
        return <p className="text-sm text-muted-foreground">{formattedDate}</p>;
      }
    },
  },
  {
    id: "last_login",
    header: "Last Login",
    cell: ({ row }) => {
      const user = row.original as User;
      const lastLogin = user.last_login;

      if (lastLogin) {
        const formattedDate = formatDateToHumanReadable(lastLogin);
        const tooltipDate = formatDate(lastLogin);

        return (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <p className="text-sm text-muted-foreground">{formattedDate}</p>
              </TooltipTrigger>
              <TooltipContent>
                <p>{tooltipDate}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        );
      } else {
        return <p className="text-sm text-muted-foreground">-</p>;
      }
    },
  },
  {
    id: "actions",
    header: "Actions",
    cell: ({ row }) => {
      const user = row.original as User;
      const [isViewUserDialogOpen, setIsViewUserDialogOpen] =
        React.useState(false);
      const [isEditUserDialogOpen, setIsEditUserDialogOpen] =
        React.useState(false);
      return (
        <>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm">
                <Menu className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-56">
              <DropdownMenuLabel>User Actions</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem onClick={() => setIsViewUserDialogOpen(true)}>
                  <UserIcon className="h-4 w-4 mr-2" />
                  <span>View User</span>
                  <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setIsEditUserDialogOpen(true)}>
                  <UserCog className="h-4 w-4 mr-2" />
                  <span>Edit User</span>
                  <DropdownMenuShortcut>⇧⌘B</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => console.log(`deleting user ${user.username}`)}
                >
                  <UserMinus className="h-4 w-4 mr-2" />
                  <span>Delete User</span>
                  <DropdownMenuShortcut>⇧⌘X</DropdownMenuShortcut>
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
          <ViewUserDialog
            user={user}
            isOpen={isViewUserDialogOpen}
            onClose={() => setIsViewUserDialogOpen(false)}
          />
          <EditUserDialog
            user={user}
            isOpen={isEditUserDialogOpen}
            onClose={() => setIsEditUserDialogOpen(false)}
          />
        </>
      );
    },
  },
];
