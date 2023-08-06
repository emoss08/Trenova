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

import { Card, Flex, NavLink } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faBox,
  faBuildingColumns,
  faFileInvoiceDollar,
  faInbox,
  faTruckFast,
} from "@fortawesome/pro-duotone-svg-icons";
import { faRoad } from "@fortawesome/pro-duotone-svg-icons/faRoad";
import React, { lazy } from "react";
import { IconDefinition } from "@fortawesome/free-solid-svg-icons";
import { usePageStyles } from "@/styles/PageStyles";

const BillingControlContent = lazy(() => import("./BillingControl"));
const DispatchControlContent = lazy(() => import("./DispatchControl"));
const InvoiceControlContent = lazy(() => import("./InvoiceControl"));
const OrderControlContent = lazy(() => import("./OrderControl"));
const EmailControlContent = lazy(() => import("./EmailControl"));
const RouteControlContent = lazy(() => import("./RouteControl"));

export interface ControlFileData {
  icon: IconDefinition;
  label: string;
  description: string;
  component: React.ComponentType<any>;
}

interface NavBarProps {
  activeTab: ControlFileData;
  setActiveTab: (tab: ControlFileData) => void;
  navigate: (path: string) => void;
}

// TODO(Wolfred): Add permissions to control files to restrict access to certain users
export const controlFileData: ControlFileData[] = [
  {
    icon: faBuildingColumns,
    label: "Billing Controls",
    description: "Control and Monitor Billing Processes",
    component: BillingControlContent,
  },
  {
    icon: faTruckFast,
    label: "Dispatch Controls",
    description: "Manage and Oversee Dispatch Operations",
    component: DispatchControlContent,
  },
  {
    icon: faFileInvoiceDollar,
    label: "Invoice Controls",
    description: "Handle Invoicing and Payment Methods",
    component: InvoiceControlContent,
  },
  {
    icon: faBox,
    label: "Order Controls",
    description: "Administer and Manage Order Procedures",
    component: OrderControlContent,
  },
  {
    icon: faInbox,
    label: "Email Controls",
    description: "Supervise and Modify Email Settings",
    component: EmailControlContent,
  },
  {
    icon: faRoad,
    label: "Route Controls",
    description: "Manage and Optimize Delivery Routes",
    component: RouteControlContent,
  },
];

export function NavBar({ activeTab, setActiveTab, navigate }: NavBarProps) {
  const { classes } = usePageStyles();
  const items = controlFileData.map((item, index) => (
    <NavLink
      key={index}
      label={item.label}
      description={item.description}
      icon={<FontAwesomeIcon icon={item.icon} />}
      active={activeTab && item.label === activeTab.label}
      onClick={() => {
        setActiveTab(item);
        navigate(`#${item.label.toLowerCase().replace(/ /g, "-")}`);
      }}
    />
  ));
  return (
    <Flex>
      <Card className={classes.card} withBorder>
        {items}
      </Card>
    </Flex>
  );
}
