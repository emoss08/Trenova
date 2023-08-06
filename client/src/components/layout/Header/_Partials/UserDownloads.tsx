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
  createStyles,
  Divider,
  Menu,
  rem,
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
import { useHeaderStyles } from "@/styles/HeaderStyles";
import { getUserReports } from "@/requests/UserRequestFactory";
import { getUserId } from "@/lib/utils";

const useStyles = createStyles((theme) => ({
  mainLinks: {
    paddingLeft: `calc(${theme.spacing.md} - ${theme.spacing.xs})`,
    paddingRight: `calc(${theme.spacing.md} - ${theme.spacing.xs})`,
  },

  mainLink: {
    fontWeight: 500,
    display: "flex",
    alignItems: "center",
    width: "100%",
    fontSize: theme.fontSizes.xs,
    padding: `${rem(8)} ${theme.spacing.xs}`,
    borderRadius: theme.radius.sm,
    "& svg": {
      color:
        theme.colorScheme === "dark"
          ? theme.colors.dark[2]
          : theme.colors.gray[6],
    },
    color:
      theme.colorScheme === "dark" ? theme.colors.dark[0] : theme.colors.black,

    "&:hover": {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[6]
          : theme.colors.gray[0],
      color: theme.colorScheme === "dark" ? theme.white : theme.black,
    },
    "&:hover svg": {
      color: theme.colorScheme === "dark" ? theme.colors.gray[0] : theme.black,
    },
  },

  mainLinkInner: {
    display: "flex",
    alignItems: "center",
    fontWeight: 500,
    flex: 1,
  },

  mainLinkIcon: {
    marginRight: theme.spacing.sm,
    color:
      theme.colorScheme === "dark"
        ? theme.colors.dark[2]
        : theme.colors.gray[6],
  },

  mainLinkBadge: {
    padding: 0,
    width: rem(20),
    height: rem(20),
    pointerEvents: "none",
  },
}));

export const UserDownloads: React.FC = () => {
  const [downloadMenuOpen] = useNavbarStore.use("downloadMenuOpen");
  const { classes } = useStyles();
  const { classes: headerClasses } = useHeaderStyles();
  const userId = getUserId() || "";
  const queryClient = useQueryClient();

  // No stale time on this we want it to always be up-to-date
  const { data: userReportData, isLoading: isUserReportDataLoading } = useQuery(
    {
      queryKey: ["userReport", userId],
      queryFn: () => getUserReports(),
      initialData: () => queryClient.getQueryData(["userReport", userId]),
    }
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
            <Badge
              size="sm"
              variant="filled"
              className={classes.mainLinkBadge}
            >
              {userReportData?.count || 0}
            </Badge>
          </UnstyledButton>
        </div>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Label>Downloads</Menu.Label>
        <Divider />
        <>
          {isUserReportDataLoading ? (
            <Skeleton width={220} height={250} />
          ) : (
            <ScrollArea h={250} scrollbarSize={5} offsetScrollbars>
              {userReportData && <UserReports reportData={userReportData} />}
            </ScrollArea>
          )}
        </>
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
};
