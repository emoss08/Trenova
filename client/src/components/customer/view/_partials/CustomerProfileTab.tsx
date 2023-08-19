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
import { Card, Flex, Skeleton } from "@mantine/core";
import { useQuery, useQueryClient } from "react-query";
import { usePageStyles } from "@/styles/PageStyles";
import { getCustomerEmailProfile } from "@/requests/CustomerRequestFactory";
import { customerStore as store } from "@/stores/CustomerStore";
import { CustomerEmailProfileForm } from "./profile_forms/CustomerEmailProfileForm";

type CustomerProfileTabProps = {
  customerId: string;
};

export function CustomerProfileTab({ customerId }: CustomerProfileTabProps) {
  const { classes } = usePageStyles();
  const queryClient = useQueryClient();

  const { data: emailProfile, isLoading: isEmailProfileLoading } = useQuery({
    queryKey: ["customerEmailProfile", customerId],
    queryFn: () => getCustomerEmailProfile(customerId),
    enabled: store.get("activeTab") === "profile",
    initialData: () => queryClient.getQueryData("customerEmailProfiles"),
    staleTime: Infinity,
  });

  const isLoading = isEmailProfileLoading;

  return (
    <Flex>
      {isLoading ? (
        <Skeleton height={200} />
      ) : (
        <Card className={classes.card}>
          {emailProfile && (
            <CustomerEmailProfileForm emailProfile={emailProfile} />
          )}
        </Card>
      )}
    </Flex>
  );
}
