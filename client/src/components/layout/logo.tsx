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
import { useUserOrganization } from "@/hooks/useQueries";
import { Image } from "@unpic/react";
import { Link } from "react-router-dom";

export function Logo() {
  const { userOrganizationData, userOrganizationLoading } =
    useUserOrganization();

  if (userOrganizationLoading) {
    return <Skeleton className="h-10 w-40" />;
  }

  if (userOrganizationData && userOrganizationData.logo) {
    return <LogoLink src={userOrganizationData.logo} alt="Organization Logo" />;
  }

  return (
    <span className=" max-w-[200px] truncate">
      <Link
        className="text-accent-foreground text-xl font-semibold"
        to="/"
        title={userOrganizationData?.name}
      >
        {userOrganizationData?.name}
      </Link>
    </span>
  );
}

export function OrganizationNameLogo() {
  const { userOrganizationData, userOrganizationLoading } =
    useUserOrganization();

  if (userOrganizationLoading) {
    return <Skeleton className="h-10 w-40" />;
  }

  return (
    <span className=" max-w-[200px] truncate">
      <Link
        className="text-accent-foreground text-xl font-bold"
        to="/"
        title={userOrganizationData?.name}
      >
        {userOrganizationData?.name}
      </Link>
    </span>
  );
}

function LogoLink({ src, alt }: { src: string; alt: string }) {
  return (
    <Link to="/" style={{ textDecoration: "none" }}>
      <Image
        src={src}
        layout="constrained"
        className="object-contain"
        width={40}
        height={40}
        alt={alt}
      />
    </Link>
  );
}
