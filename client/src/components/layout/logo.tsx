/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
import { Link } from "react-router-dom";
import { useTheme } from "../ui/theme-provider";
import { useUserOrganization } from "@/hooks/useQueries";

export function Logo() {
  const { theme } = useTheme();
  const { userOrganizationData, userOrganizationLoading } =
    useUserOrganization();

  if (userOrganizationLoading) {
    return <Skeleton className="h-10 w-40" />;
  }

  if (userOrganizationData && userOrganizationData.logo) {
    const logoSource =
      theme === "light"
        ? userOrganizationData.logo
        : userOrganizationData.darkLogo || userOrganizationData.logo;

    return <LogoLink src={logoSource} alt="Organization Logo" />;
  }

  return (
    <Link
      className="mr-5 max-w-[250px] truncate text-xl font-semibold text-accent-foreground"
      to="/"
      title={userOrganizationData?.name}
    >
      {userOrganizationData?.name}
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
