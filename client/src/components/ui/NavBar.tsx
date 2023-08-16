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
import { Card, Flex, NavLink } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { IconDefinition } from "@fortawesome/free-solid-svg-icons";
import { MantineColor } from "@mantine/styles";
import { usePageStyles } from "@/styles/PageStyles";

export interface NavLinks {
  icon: IconDefinition;
  label: string;
  description: string;
  component: React.ComponentType<any>;
  id?: string;
  color?: MantineColor;
}

interface NavBarProps {
  activeTab: NavLinks;
  setActiveTab: (tab: NavLinks) => void;
  navigate: (path: string) => void;
  data: NavLinks[];
}

export function NavBar({
  activeTab,
  setActiveTab,
  navigate,
  data,
}: NavBarProps) {
  const { classes } = usePageStyles();
  const items = data.map((item) => (
    <NavLink
      key={`${item.id}-${activeTab}-${item.label}`}
      label={item.label}
      description={item.description}
      icon={<FontAwesomeIcon icon={item.icon} />}
      color={item.color || "blue"}
      active={activeTab && item.label === activeTab.label}
      onClick={() => {
        setActiveTab(item);
        navigate(`#${item.label.toLowerCase().replace(/ /g, "-")}`);
      }}
      variant="light"
    />
  ));
  return (
    <Flex>
      <Card className={classes.card}>{items}</Card>
    </Flex>
  );
}
