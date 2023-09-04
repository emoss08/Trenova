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
import { Grid } from "@mantine/core";
import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "react-query";
import { getCustomerDetailsWithMetrics } from "@/services/CustomerRequestService";

export default function EditCustomer() {
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const { data: customerData, isLoading: isCustomerDataLoading } = useQuery({
    queryKey: ["customer", id],
    queryFn: () => {
      if (!id) {
        return Promise.resolve(null);
      }
      return getCustomerDetailsWithMetrics(id);
    },
    initialData: () => queryClient.getQueryData(["customerWithMetrics", id]),
    staleTime: Infinity,
  });

  return (
    <Grid>
      <Grid.Col span={12} sm={6} md={4} lg={3} xl={3}>
        <p>TEST</p>
      </Grid.Col>
    </Grid>
  );
}
