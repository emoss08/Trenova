/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { useUserOrganization } from "@/hooks/useQueries";
import { faChevronDown } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";
import { Skeleton } from "../ui/skeleton";

export function OrganizationLogo() {
  const { data, isLoading } = useUserOrganization();

  if (isLoading) {
    return <Skeleton className="h-14" />;
  }

  const initial = data?.name?.charAt(0);

  const cityState = `${data?.city}, ${data?.state?.abbreviation}`;

  return (
    <div className="hover:bg-muted group col-span-full flex w-full items-center gap-x-4 rounded-lg p-1 hover:cursor-pointer">
      <Avatar className="bg-muted/50 group-hover:bg-muted-foreground/20 border-muted size-14 flex-none rounded-lg border">
        <AvatarImage
          src={data?.logoUrl || ""}
          alt={"Trenova Logo"}
          className="size-14 flex-none rounded-lg p-2"
        />
        <AvatarFallback className="rounded-none" delayMs={600}>
          {initial}
        </AvatarFallback>
      </Avatar>
      <div className="flex flex-1 flex-col">
        <div className="flex items-center">
          <h2 className="text-foreground mr-2 w-36 truncate text-lg font-semibold leading-7">
            {data?.name || ""}
          </h2>
          <FontAwesomeIcon
            icon={faChevronDown}
            className="text-muted-foreground size-3"
          />
        </div>
        <p className="text-muted-foreground text-sm">{cityState}</p>
      </div>
    </div>
  );
}

export function MiniOrganizationLogo() {
  const { data, isLoading } = useUserOrganization();

  if (isLoading) {
    return <Skeleton className="h-14" />;
  }

  const initial = data?.name?.charAt(0);

  return (
    <div className="hover:bg-muted rounded-lghover:cursor-pointer group col-span-full flex w-full items-center gap-x-4">
      <Avatar className="bg-muted/50 group-hover:bg-muted-foreground/20 border-muted size-9 flex-none rounded-lg border">
        <AvatarImage
          src={data?.logoUrl || ""}
          alt={"Trenova Logo"}
          className="size-9 flex-none rounded-lg p-2"
        />
        <AvatarFallback className="rounded-none" delayMs={600}>
          {initial}
        </AvatarFallback>
      </Avatar>
    </div>
  );
}
