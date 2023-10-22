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
import { Box, Button, Card, Skeleton, Text, Title } from "@mantine/core";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getCustomerEmailProfile,
  getCustomerRuleProfile,
} from "@/services/CustomerRequestService";
import { customerStore as store } from "@/stores/CustomerStore";
import { CustomerEmailProfile, CustomerRuleProfile } from "@/types/customer";
import { CreateRuleProfileModal } from "./CreateRuleProfileModal";
import { usePageStyles } from "@/assets/styles/PageStyles";

type CustomerProfileTabProps = {
  customerId: string;
};

export function CustomerProfileTab({ customerId }: CustomerProfileTabProps) {
  const queryClient = useQueryClient();
  const { classes } = usePageStyles();

  const { data: emailProfile, isLoading: isEmailProfileLoading } = useQuery({
    queryKey: ["customerEmailProfile", customerId],
    queryFn: (): Promise<CustomerEmailProfile> =>
      getCustomerEmailProfile(customerId),
    enabled: store.get("activeTab") === "profile",
    initialData: () => queryClient.getQueryData("customerEmailProfiles"),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const { data: ruleProfile, isLoading: isRuleProfileLoading } = useQuery({
    queryKey: ["customerRuleProfile", customerId],
    queryFn: (): Promise<CustomerRuleProfile> =>
      getCustomerRuleProfile(customerId),
    enabled: store.get("activeTab") === "profile",
    initialData: () => queryClient.getQueryData("customerRuleProfile"),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    retry: false,
  });

  const isLoading = isEmailProfileLoading || isRuleProfileLoading;

  return isLoading ? (
    <>
      <Skeleton height={400} />
      <Skeleton height={300} mt={20} />
    </>
  ) : (
    <>
      {emailProfile ? (
        <p>Test</p>
      ) : (
        // <CustomerEmailProfileForm form={form} />
        <Card mb={20} className={classes.card}>
          <Box
            style={{
              textAlign: "center",
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
              flexDirection: "column",
            }}
          >
            <Title size="x-large"> No Email Profile Found</Title>
            <Text variant="dimmed" size="sm">
              Email Profile is used to define configurations that will be
              applied to the billing of the customer. Create a email profile to
              start billing the customer.
            </Text>
            <Button
              size="sm"
              color="blue"
              mt="xl"
              onClick={() => {
                store.set("createRuleProfileModalOpen", true);
              }}
            >
              Create Email Profile
            </Button>
            {/* <Image maw={200} src={image} className={classes.desktopImage} /> */}
          </Box>
        </Card>
      )}
      {ruleProfile ? (
        // <CustomerRuleProfileForm ruleProfile={ruleProfile} />
        <p>Test</p>
      ) : (
        <Card mb={20} className={classes.card}>
          <Box
            style={{
              textAlign: "center",
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
              flexDirection: "column",
            }}
          >
            <Title size="x-large">No Rule Profile Found</Title>
            <Text variant="dimmed" size="sm">
              Rule Profiles are used to define the rules that will be applied to
              the billing of the customer. Create a rule profile to start
              billing the customer.
            </Text>
            <Button
              size="sm"
              color="blue"
              mt="xl"
              onClick={() => {
                store.set("createRuleProfileModalOpen", true);
              }}
            >
              Create Rule Profile
            </Button>
            {/* <Image maw={200} src={image} className={classes.desktopImage} /> */}
          </Box>
        </Card>
      )}
      <CreateRuleProfileModal customerId={customerId} />
    </>
  );
}
