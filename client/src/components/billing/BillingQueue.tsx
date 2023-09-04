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
import { createStyles, Divider, Skeleton } from "@mantine/core";
import { useQuery, useQueryClient } from "react-query";
import Typed from "typed.js";
import { WebSocketManager } from "@/helpers/websockets";
import { getBillingQueue } from "@/services/BillingRequestService";
import { BillingQueueTable } from "@/components/billing/_partials/BillingQueueTable";

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
  };
});

const BillingQueue: React.FC<Props> = ({ websocketManager }) => {
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
      typed.destroy();
    };
  }, []);

  const { data: billingQueueData, isLoading: isBillingQueueDataLoading } =
    useQuery({
      queryKey: ["billingQueueData"],
      queryFn: () => getBillingQueue(),
      initialData: () => queryClient.getQueryData(["billingQueueData"]),
    });

  return (
    <>
      <span ref={el} className={classes.text} />
      <Divider my={10} />
      {isBillingQueueDataLoading ? (
        <Skeleton height={500} />
      ) : (
        billingQueueData && (
          <BillingQueueTable
            data={billingQueueData}
            websocketManager={websocketManager}
          />
        )
      )}
    </>
  );
};

export default BillingQueue;
