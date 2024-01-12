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

import {
  faBox,
  faBuildingColumns,
  faFileInvoiceDollar,
  faInbox,
  faMoneyBillTransfer,
  faTruckClock,
  faTruckFast,
} from "@fortawesome/pro-duotone-svg-icons";
import { lazy } from "react";
import { faRoad } from "@fortawesome/pro-duotone-svg-icons/faRoad";
import { NavLinks } from "@/components/common/NavBar";

const BillingControlContent = lazy(
  () => import("../../../components/control-files/BillingControl"),
);
const DispatchControlContent = lazy(
  () => import("../../../components/control-files/DispatchControl"),
);
const InvoiceControlContent = lazy(
  () => import("../../../components/control-files/InvoiceControl"),
);
const ShipmentControlContent = lazy(
  () => import("../../../components/control-files/ShipmentControl"),
);
const EmailControlContent = lazy(
  () => import("../../../components/control-files/EmailControl"),
);
const RouteControlContent = lazy(
  () => import("../../../components/control-files/RouteControl"),
);
const FeasibilityToolControlContent = lazy(
  () => import("../../../components/control-files/FeasibilityControl"),
);
const AccountingControlContent = lazy(
  () => import("../../../components/control-files/AccountingControl"),
);

// TODO(Wolfred): Add permissions to control files to restrict access to certain users
export const controlFileData: NavLinks[] = [
  {
    icon: faBuildingColumns,
    label: "Accounting Controls",
    description: "Control and Oversee Accounting Processes",
    component: AccountingControlContent,
  },
  {
    icon: faMoneyBillTransfer,
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
    label: "Shipment Controls",
    description: "Administer and Manage Shipment Procedures",
    component: ShipmentControlContent,
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
  {
    icon: faTruckClock,
    label: "Feasibility Tool Controls",
    description: "Control and Optimize Feasibility Tool",
    component: FeasibilityToolControlContent,
  },
];
