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
import { createStyles, Divider, rem, Skeleton } from "@mantine/core";
import { WebSocketManager } from "@/utils/websockets";
import { OrdersReadyTable } from "@/components/billing/_partials/OrdersReadyTable";
import { useQuery, useQueryClient } from "react-query";
import { getOrdersReadyToBill } from "@/requests/BillingRequestFactory";
import Typed from "typed.js";

interface Props {
  websocketManager: WebSocketManager;
}

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

export const OrdersReady: React.FC<Props> = ({ websocketManager }) => {
  const queryClient = useQueryClient();
  const { classes } = useStyles();
  const el = React.useRef(null);

  React.useEffect(() => {
    const typed = new Typed(el.current, {
      strings: [
        "Validate orders ready to be billed.",
        "Click on the order to bill it.",
      ],
      typeSpeed: 50,
      loop: true,
    });

    return () => {
      // Destroy Typed instance during cleanup to stop animation
      typed.destroy();
    };
  }, []);

  const { data: readyOrdersData, isLoading: isReadyOrdersDataLoading } =
    useQuery({
      queryKey: ["readyOrdersData"],
      queryFn: () => getOrdersReadyToBill(),
      initialData: () => {
        return queryClient.getQueryData(["readyOrdersData"]);
      },
    });

  return (
    <>
      <span ref={el} className={classes.text} />
      <Divider my={10} />
      {isReadyOrdersDataLoading ? (
        <Skeleton height={500}></Skeleton>
      ) : (
        readyOrdersData && (
          <OrdersReadyTable
            data={readyOrdersData}
            websocketManager={websocketManager}
          />
        )
      )}
    </>
  );
};
