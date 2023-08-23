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
import {
  Skeleton,
  createStyles,
  rem,
  Image,
  Title,
  Text,
  Card,
  Divider,
  Button,
} from "@mantine/core";
import { useQuery, useQueryClient } from "react-query";
import image from "../../../../assets/images/notfound.png";
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
import { Alert } from "@/components/ui/Alert";
import { CreateRuleProfileModal } from "./CreateRuleProfileModal";

type CustomerProfileTabProps = {
  customerId: string;
};

const useStyles = createStyles((theme) => ({
  card: {
    width: "100%",
    // maxWidth: "100%",
    // height: "auto",
    "@media (max-width: 576px)": {
      height: "auto",
      maxHeight: "none",
    },
  },
  root: {
    paddingTop: rem(80),
    paddingBottom: rem(80),
  },

  title: {
    fontWeight: 900,
    fontSize: rem(34),
    marginBottom: theme.spacing.md,
    fontFamily: `Greycliff CF, ${theme.fontFamily}`,

    [theme.fn.smallerThan("sm")]: {
      fontSize: rem(32),
    },
  },

  control: {
    [theme.fn.smallerThan("sm")]: {
      width: "100%",
    },
  },

  mobileImage: {
    [theme.fn.largerThan("sm")]: {
      display: "none",
    },
  },

  desktopImage: {
    [theme.fn.smallerThan("sm")]: {
      display: "none",
    },
  },
}));

export function CustomerProfileTab({ customerId }: CustomerProfileTabProps) {
  const queryClient = useQueryClient();
  const { classes } = useStyles();

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
        <CustomerEmailProfileForm emailProfile={emailProfile} />
      ) : (
        <Card>
          <div
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
            <Image maw={240} src={image} className={classes.desktopImage} />
            {/* <Text> */}
            {/*   Email Profile is used to define the email rules that will be */}
            {/*   applied to the billing of the customer. Create a email profile to */}
            {/*   start billing the customer. */}
            {/* </Text> */}
            <Button
              size="md"
              color="yellow"
              mt="xl"
              className={classes.control}
              onClick={() => {
                store.set("createRuleProfileModalOpen", true);
              }}
            >
              Create Email Profile
            </Button>
          </div>
        </Card>
        // <Container className={classes.root}>
        //   <SimpleGrid
        //     spacing={80}
        //     cols={2}
        //     breakpoints={[{ maxWidth: "sm", cols: 1, spacing: 40 }]}
        //   >
        //     <Image src={image} className={classes.mobileImage} />
        //     <div>
        //       <Title className={classes.title}>
        //         🚀 Introducing the Monta Billing Client
        //       </Title>
        //
        //       <Text color="dimmed" size="lg">
        //         Email Profile is used to define the email rules that will be
        //         applied to the billing of the customer. Create a email profile
        //         to start billing the customer.
        //       </Text>
        //     </div>
        //     <Image src={image} className={classes.desktopImage} />
        //   </SimpleGrid>
        // </Container>
        // <Alert
        //   color="yellow"
        //   title="No Email Profile Found"
        //   message="Email Profile is used to define the email rules that will be applied to the billing of the customer. Create a email profile to start billing the customer."
        //   buttonText="Create"
        //   withButton
        //   mb={20}
        //   onClick={() => {
        //     store.set("createRuleProfileModalOpen", true);
        //   }}
        // />
      )}
      {ruleProfile ? (
        <CustomerRuleProfileForm ruleProfile={ruleProfile} />
      ) : (
        <Alert
          color="yellow"
          title="No Rule Profile Found"
          message="Rule Profiles are used to define the rules that will be applied to the billing of the customer. Create a rule profile to start billing the customer."
          buttonText="Create"
          withButton
          onClick={() => {
            store.set("createRuleProfileModalOpen", true);
          }}
        />
      )}
      <CreateRuleProfileModal customerId={customerId} />
    </>
  );
}
