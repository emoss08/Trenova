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
  Button,
  Container,
  createStyles,
  Image,
  rem,
  SimpleGrid,
  Text,
  Title,
} from "@mantine/core";
import { WebSocketManager } from "@/utils/websockets";
import { STEPS } from "@/pages/billing/BillingClient";
import { billingClientStore } from "@/stores/BillingStores";

interface Props {
  websocketManager: WebSocketManager;
}

import image from "@/assets/images/profit.png";

const useStyles = createStyles((theme) => {
  const BREAKPOINT = theme.fn.smallerThan("sm");

  return {
    text: {
      fontSize: BREAKPOINT ? theme.fontSizes.xl : theme.fontSizes.xxl,
      fontWeight: 700,
      color: theme.colorScheme === "dark" ? "white" : "black",
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
  };
});

export const GettingStarted: React.FC<Props> = ({ websocketManager }) => {
  const [websocketMessage] = billingClientStore.use("websocketMessage");
  const { classes } = useStyles();

  const getStarted = () => {
    // send message to websocket to get started
    websocketManager.sendJsonMessage("billing_client", {
      action: STEPS[0], // or "get_started"
    });
  };

  return (
    <>
      <Container className={classes.root}>
        <SimpleGrid
          spacing={80}
          cols={2}
          breakpoints={[{ maxWidth: "sm", cols: 1, spacing: 40 }]}
        >
          <Image src={image} className={classes.mobileImage} />
          <div>
            <Title className={classes.title}>
              🚀 Introducing the Monta Billing Client
            </Title>

            <Text color="dimmed" size="lg">
              {websocketMessage?.payload?.message
                ? (websocketMessage?.payload?.message as string)
                : "The Monta Billing Client is your efficient partner for end-to-end billing management." +
                  " This ingenious software client for interacting with the billing API offers a streamlined" +
                  " approach for handling your financial transactions."}
            </Text>

            {!websocketMessage?.payload?.message && (
              <Button
                variant="outline"
                size="md"
                mt="xl"
                className={classes.control}
                onClick={getStarted}
              >
                Get Started
              </Button>
            )}
          </div>
          <Image src={image} className={classes.desktopImage} />
        </SimpleGrid>
      </Container>
    </>
  );
};
