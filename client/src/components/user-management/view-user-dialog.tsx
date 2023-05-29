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
import { User } from "@/types/user";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Loader2 } from "lucide-react";
import {
  getUserDepartment,
  getUserJobTitle,
  getUserOrganization,
} from "@/requests/UserRequestFactory";
import { useQuery } from "react-query";
import { formatDate } from "@/utils/date";
import { Separator } from "@/components/ui/separator";

interface ViewUserDialogProps {
  user: User;
  isOpen: boolean;
  onClose: () => void;
}

export const ViewUserDialog: React.FC<ViewUserDialogProps> = ({
  user,
  isOpen,
  onClose,
}) => {
  const { data: jobTitleData, isLoading: isJobTitleLoading } = useQuery(
    ["jobTitle", user.profile?.job_title],
    () => getUserJobTitle(user),
    {
      enabled: isOpen && !!user.profile?.job_title,
    }
  );

  const { data: organizationData, isLoading: isOrganizationLoading } = useQuery(
    ["organization", user.profile?.organization],
    () => getUserOrganization(user),
    {
      enabled: isOpen,
    }
  );

  const { data: departmentData, isLoading: isDepartmentLoading } = useQuery(
    ["department", user?.department],
    () => getUserDepartment(user),
    {
      enabled: isOpen && !!user?.department,
    }
  );

  const isLoading =
    isJobTitleLoading || isOrganizationLoading || isDepartmentLoading;
  const jobTitle = jobTitleData?.name ?? "-";
  const organization = organizationData?.name ?? "-";
  const department = departmentData?.name ?? "-";

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>
            {user.profile?.first_name ?? "-"} {user.profile?.last_name ?? "-"}
          </DialogTitle>
          <DialogDescription>
            You are currently viewing the profile of{" "}
            {user.profile?.first_name ?? "-"} {user.profile?.last_name ?? "-"} (
            {user.username}).
          </DialogDescription>
        </DialogHeader>
        {isLoading ? (
          <>
            <div className="mt-2 inline-flex items-center justify-center">
              <Loader2 className="mr-2 h-10 w-10 animate-spin" />
            </div>
          </>
        ) : (
          <div className="mt-4">
            <div className="flex flex-col sm:flex-row">
              <div className="flex-1">
                <div className="text-sm font-medium text-gray-500">
                  Username
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.username}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">Email</div>
                <div className="mt-1 text-sm text-gray-900">{user.email}</div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Organization
                </div>
                <div className="mt-1 text-sm text-gray-900">{organization}</div>
              </div>
            </div>
            <div className="flex flex-col sm:flex-row mt-4">
              <div className="flex-1">
                <div className="text-sm font-medium text-gray-500">
                  Department
                </div>
                <div className="mt-1 text-sm text-gray-900">{department}</div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Super User?
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.is_superuser ? "Yes" : "No"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Staff Member?
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.is_staff ? "Yes" : "No"}
                </div>
              </div>
            </div>
            <div className="flex flex-col sm:flex-row mt-4">
              <div className="flex-1">
                <div className="text-sm font-medium text-gray-500">
                  Is Active?
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.is_active ? "Yes" : "No"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Last Login Date
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.last_login ? formatDate(user.last_login) : "-"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4"></div>
            </div>
            <p className="mt-4 mb-1 text-center">Profile Details</p>
            <Separator />
            <div className="flex flex-col sm:flex-row mt-4">
              <div className="flex-1">
                <div className="text-sm font-medium text-gray-500">
                  First Name
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.profile?.first_name ?? "-"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Last Name
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.profile?.last_name ?? "-"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Job Title
                </div>
                <div className="mt-1 text-sm text-gray-900">{jobTitle}</div>
              </div>
            </div>
            <div className="flex flex-col sm:flex-row mt-4">
              <div className="flex-1">
                <div className="text-sm font-medium text-gray-500">
                  Address Line 1
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.profile?.address_line_1 ?? "-"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Address Line 2
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.profile?.address_line_2 ?? "-"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">City</div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.profile?.city ?? "-"}
                </div>
              </div>
            </div>
            <div className="flex flex-col sm:flex-row mt-4">
              <div className="flex-1">
                <div className="text-sm font-medium text-gray-500">State</div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.profile?.state ?? "-"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Zip Code
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.profile?.zip_code ?? "-"}
                </div>
              </div>
              <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                <div className="text-sm font-medium text-gray-500">
                  Phone Number
                </div>
                <div className="mt-1 text-sm text-gray-900">
                  {user.profile?.phone_number ?? "-"}
                </div>
              </div>
            </div>
            <div className="mt-4">
              <div className="text-sm font-medium text-gray-500">
                Phone Number Verified?
              </div>
              <div className="mt-1 text-sm text-gray-900">
                {user.profile?.is_phone_verified ? "Yes" : "No"}
              </div>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};
