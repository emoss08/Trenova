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
  Badge,
  Divider,
  Menu,
  ScrollArea,
  Skeleton,
  UnstyledButton,
} from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faDownload } from "@fortawesome/pro-duotone-svg-icons";
import { Link } from "react-router-dom";
import { useQuery, useQueryClient } from "react-query";
import { UserReports } from "@/components/layout/Header/_Partials/UserReports";
import { useNavbarStore } from "@/stores/HeaderStore";
import { useHeaderStyles } from "@/assets/styles/HeaderStyles";
import { getUserReports } from "@/services/UserRequestService";
import { getUserId } from "@/helpers/constants";
import { useAsideStyles } from "@/assets/styles/AsideStyles";

export function UserDownloads(): React.ReactElement {
  const [downloadMenuOpen] = useNavbarStore.use("downloadMenuOpen");
  const { classes } = useAsideStyles();
  const { classes: headerClasses } = useHeaderStyles();
  const userId = getUserId() || "";
  const queryClient = useQueryClient();

  // No stale time on this we want it to always be up-to-date
  const { data: userReportData, isLoading: isUserReportDataLoading } = useQuery(
    {
      queryKey: ["userReport", userId],
      queryFn: () => getUserReports(),
      initialData: () => queryClient.getQueryData(["userReport", userId]),
    },
  );

  return (
    <Menu
      position="right-start"
      width={230}
      opened={downloadMenuOpen}
      onChange={(changeEvent) => {
        useNavbarStore.set("downloadMenuOpen", changeEvent);
      }}
      withinPortal
      withArrow
      arrowSize={5}
    >
      <Menu.Target>
        <div className={classes.mainLinks}>
          <UnstyledButton className={classes.mainLink}>
            <div className={classes.mainLinkInner}>
              <FontAwesomeIcon
                size="lg"
                icon={faDownload}
                className={classes.mainLinkIcon}
              />
              <span>Downloads</span>
            </div>
            <Badge size="sm" variant="filled" className={classes.mainLinkBadge}>
              {userReportData?.count || 0}
            </Badge>
          </UnstyledButton>
        </div>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Label>Downloads</Menu.Label>
        <Divider />
        {isUserReportDataLoading ? (
          <Skeleton width={220} height={250} />
        ) : (
          <ScrollArea h={250} scrollbarSize={5} offsetScrollbars>
            {userReportData && <UserReports reportData={userReportData} />}
          </ScrollArea>
        )}
        <Divider mb={2} mt={10} />
        <Link
          to="#"
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            marginTop: "5px",
          }}
          className={headerClasses.link}
        >
          View all{" "}
          <FontAwesomeIcon
            icon={faArrowRight}
            size="sm"
            style={{
              marginLeft: "5px",
              marginTop: "2px",
            }}
          />
        </Link>
      </Menu.Dropdown>
    </Menu>
  );
}
