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
import { Card, Skeleton } from "@mantine/core";
import { useQuery, useQueryClient } from "react-query";
import { faShieldCheck } from "@fortawesome/pro-duotone-svg-icons";
import {
  getCustomerEmailProfile,
  getCustomerRuleProfile,
} from "@/requests/CustomerRequestFactory";
import { customerStore as store } from "@/stores/CustomerStore";
import { CustomerEmailProfileForm } from "./profile_forms/CustomerEmailProfileForm";
import { CustomerRuleProfileForm } from "./profile_forms/CustomerRuleProfileForm";
import {
  CustomerEmailProfile,
  CustomerRuleProfile,
} from "@/types/apps/customer";
import { usePageStyles } from "@/styles/PageStyles";
import { Alert } from "@/components/ui/Alert";
import { CreateRuleProfileModal } from "./CreateRuleProfileModal";

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

  const isLoading: boolean = isEmailProfileLoading || isRuleProfileLoading;

  return isLoading ? (
    <>
      <Skeleton height={400} />
      <Skeleton height={300} mt={20} />
    </>
  ) : (
    <>
      {emailProfile && <CustomerEmailProfileForm emailProfile={emailProfile} />}
      {ruleProfile ? (
        <CustomerRuleProfileForm ruleProfile={ruleProfile} />
      ) : (
        <Card className={classes.card} mt={20}>
          <Alert
            color="yellow"
            icon={faShieldCheck}
            title="No Rule Profile Found"
            message="Rule Profiles are used to define the rules that will be applied to the billing of the customer. Create a rule profile to start billing the customer."
            buttonText="Create"
            withButton
            onClick={() => {
              store.set("createRuleProfileModalOpen", true);
            }}
            withIcon
          />
        </Card>
      )}
      <CreateRuleProfileModal customerId={customerId} />
    </>
  );
}
