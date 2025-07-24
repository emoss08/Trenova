/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { queries } from "@/lib/queries";
import { useIsAuthenticated, useUser } from "@/stores/user-store";
import { useQuery } from "@tanstack/react-query";
import { Helmet } from "@dr.pogodin/react-helmet";

export function MetaTags({
  title,
  description,
}: {
  title: string;
  description?: string;
}) {
  const user = useUser();
  const isAuthenticated = useIsAuthenticated();
  const userOrganization = useQuery({
    ...queries.organization.getOrgById(user?.currentOrganizationId ?? ""),
    enabled: !!user?.currentOrganizationId && isAuthenticated,
  });

  const defaultTitle = "Trenova";
  const organization = userOrganization.data?.name;
  const formattedTitle = title
    ? organization
      ? `${title} | ${organization} | ${defaultTitle}`
      : `${title} | ${defaultTitle}`
    : defaultTitle;

  return (
    <Helmet>
      <title>{formattedTitle}</title>
      {description && <meta name="description" content={description} />}
      <meta property="og:type" content="website" />
      <meta name="robots" content="index, follow" />
      <meta httpEquiv="X-UA-Compatible" content="IE=edge" />
      <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    </Helmet>
  );
}
