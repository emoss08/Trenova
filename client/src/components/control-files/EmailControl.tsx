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

import { usePageStyles } from "@/styles/PageStyles";
import { useQuery, useQueryClient } from "react-query";
import {
  getEmailControl,
  getEmailProfiles,
} from "@/requests/OrganizationRequestFactory";
import { Card, Divider, Skeleton, Text } from "@mantine/core";
import React from "react";
import { EmailControlForm } from "@/components/control-files/_partials/EmailControlForm";
import { EmailProfile } from "@/types/apps/organization";

const EmailControlPage = () => {
  const { classes } = usePageStyles();
  const queryClient = useQueryClient();

  const { data: emailProfileData, isLoading: isEmailProfilesLoading } =
    useQuery({
      queryKey: ["emailProfiles"],
      queryFn: () => getEmailProfiles(),
      initialData: () => {
        return queryClient.getQueryData(["emailProfiles"]);
      },
      staleTime: Infinity,
    });

  const selectEmailProfileData =
    emailProfileData?.map((emailProfile: EmailProfile) => ({
      value: emailProfile.id,
      label: emailProfile.name,
    })) || [];

  const { data: emailControlData, isLoading: isEmailControlDataLoading } =
    useQuery({
      queryKey: ["emailControl"],
      queryFn: () => getEmailControl(),
      initialData: () => {
        return queryClient.getQueryData(["emailControl"]);
      },
      staleTime: Infinity,
    });

  // Store first element of dispatchControlData in variable
  const emailControlDataArray = emailControlData?.[0];

  const isLoading = isEmailControlDataLoading || isEmailProfilesLoading;

  return (
    <>
      {isLoading ? (
        <Skeleton height={400}></Skeleton>
      ) : (
        <Card className={classes.card} withBorder>
          <Text fz="xl" fw={700} className={classes.text}>
            Email Controls
          </Text>

          <Divider my={10} />
          {emailControlDataArray && (
            <EmailControlForm
              emailControl={emailControlDataArray}
              selectEmailProfileData={selectEmailProfileData}
            />
          )}
        </Card>
      )}
    </>
  );
};

export default EmailControlPage;
