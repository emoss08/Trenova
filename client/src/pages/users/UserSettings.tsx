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

import { getUserDetails } from "@/requests/UserRequestFactory";
import React from "react";
import { useQuery } from "react-query";
import { useNavigate, useParams } from "react-router-dom";
import { getJobTitleDetails } from "@/requests/OrganizationRequestFactory";
import UserProfileDetails from "@/components/users/UserProfileDetails";
import EditUserProfileDetails from "@/components/users/EditUserProfileDetails";
import { Skeleton, Stack } from "@mantine/core";
import { SignInMethod } from "@/components/users/SignInMethod";

const UserSettings: React.FC = () => {
  const { userId } = useParams<{ userId: string }>();
  const navigate = useNavigate();

  if (!userId) {
    throw new Error("No user ID provided");
  }

  const { data: userDetails, isLoading: isUserDetailsLoading } = useQuery(
    ["userDetails", userId],
    () => getUserDetails(userId),
    {
      onError: () => navigate("/error"),
    }
  );
  const { data: jobTitleData, isLoading: isJobTitlesLoading } = useQuery(
    ["job_title"],
    () => getJobTitleDetails(userDetails.profile.job_title),
    {
      enabled: !!userDetails,
    }
  );

  const isLoading = isUserDetailsLoading || isJobTitlesLoading;
  // const isLoading = true;

  return (
    <>
      {isLoading ? (
        <Stack>
          <Skeleton height={250} />
          <Skeleton height={500} />
          <Skeleton height={100} />
        </Stack>
      ) : (
        <>
          <Stack>
            <UserProfileDetails user={userDetails} jobTitle={jobTitleData} />
            <EditUserProfileDetails user={userDetails} />
            <SignInMethod user={userDetails} />
          </Stack>
        </>
      )}
    </>
  );
};

export default UserSettings;
