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

import { Skeleton } from "@/components/ui/skeleton";
import { getUserOrganizationId } from "@/lib/auth";
import { getOrganizationDetails } from "@/services/OrganizationRequestService";
import { QueryKeys } from "@/types";
import { Organization } from "@/types/organization";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { useTheme } from "../ui/theme-provider";

export function Logo() {
  const queryClient = useQueryClient();
  const { theme } = useTheme();

  // Get User organization data
  const organizationId = getUserOrganizationId() || "";

  const { data: organizationData, isLoading } = useQuery({
    queryKey: ["organization", organizationId] as QueryKeys[],
    queryFn: async () => {
      if (!organizationId) {
        return Promise.resolve(null);
      }
      return getOrganizationDetails(organizationId);
    },
    initialData: (): Organization | undefined =>
      queryClient.getQueryData(["organization", organizationId]),
    staleTime: Infinity,
  });

  if (isLoading) {
    return <Skeleton className="h-10 w-40" />;
  }

  if (organizationData && organizationData.logo) {
    const logoSource =
      theme === "light"
        ? organizationData.logo
        : organizationData.darkLogo || organizationData.logo;

    return <LogoLink src={logoSource} alt="Organization Logo" />;
  }

  return (
    <Link
      className="mr-5 max-w-[250px] truncate text-xl font-semibold text-accent-foreground"
      to="/"
      title={organizationData?.name}
    >
      {organizationData?.name}
    </Link>
  );
}

function LogoLink({ src, alt }: { src: string; alt: string }) {
  return (
    <Link to="/" style={{ textDecoration: "none" }}>
      <img className="h-[60px] object-contain" src={src} alt={alt} />
    </Link>
  );
}
