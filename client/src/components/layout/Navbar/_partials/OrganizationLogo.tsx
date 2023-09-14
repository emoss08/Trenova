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
import { useQuery, useQueryClient } from "react-query";
import { Image, rem, Skeleton, Text } from "@mantine/core";
import { Link } from "react-router-dom";
import { getUserOrganizationId } from "@/helpers/auth";
import { getOrganizationDetails } from "@/services/OrganizationRequestService";
import { useHeaderStyles } from "@/assets/styles/HeaderStyles";

export function OrganizationLogo() {
  const queryClient = useQueryClient();
  const { classes } = useHeaderStyles();

  // Get User organization data
  const organizationId = getUserOrganizationId() || "";
  const { data: organizationData, isLoading: isOrganizationDataLoading } =
    useQuery({
      queryKey: ["organization", organizationId],
      queryFn: () => {
        if (!organizationId) {
          return Promise.resolve(null);
        }
        return getOrganizationDetails(organizationId);
      },
      initialData: () =>
        queryClient.getQueryData(["organization", organizationId]),
      staleTime: Infinity, // never refetch
    });

  if (isOrganizationDataLoading) {
    return <Skeleton width={rem(190)} height={rem(30)} />;
  }

  if (organizationData && organizationData.logo) {
    return (
      <Link to="/" style={{ textDecoration: "none" }}>
        <Image
          radius="md"
          width={rem(120)}
          height={rem(40)}
          maw={rem(150)}
          src={organizationData?.logo}
          alt="Organization Logo"
        />
      </Link>
    );
  }

  return (
    <Link to="/" style={{ textDecoration: "none" }}>
      <Text size="lg" fw={600} className={classes.logoText}>
        {organizationData?.name}
      </Text>
    </Link>
  );
}
