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

import { ActionIcon, Divider, Menu, ScrollArea } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faDownload } from "@fortawesome/pro-duotone-svg-icons";
import { UserReports } from "@/components/layout/Header/_Partials/UserReports";
import React from "react";
import { useHeaderStore } from "@/stores/HeaderStore";
import { useHeaderStyles } from "@/styles/HeaderStyles";
import { Link } from "react-router-dom";

export const UserDownloads: React.FC = () => {
  const [downloadMenuOpen] = useHeaderStore.use("downloadMenuOpen");
  const { classes } = useHeaderStyles();

  return (
    <>
      <Menu
        position="bottom-end"
        width={230}
        opened={downloadMenuOpen}
        onChange={(changeEvent) => {
          useHeaderStore.set("downloadMenuOpen", changeEvent);
        }}
        withinPortal
        withArrow
        arrowSize={5}
      >
        <Menu.Target>
          <ActionIcon className={classes.hoverEffect}>
            <FontAwesomeIcon icon={faDownload} />
          </ActionIcon>
        </Menu.Target>
        <Menu.Dropdown>
          <Menu.Label>Downloads</Menu.Label>
          <ScrollArea h={250} scrollbarSize={4}>
            <UserReports />
          </ScrollArea>
          <Divider mb={2} mt={10} />
          <Link
            to="#"
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              marginTop: "5px",
            }}
            className={classes.link}
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
    </>
  );
};
