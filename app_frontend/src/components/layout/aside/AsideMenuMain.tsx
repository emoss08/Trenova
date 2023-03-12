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

/* eslint-disable react/jsx-no-target-blank */
import { AsideMenuItemWithSub } from "./AsideMenuItemWithSub";
import { AsideMenuItem } from "./AsideMenuItem";
import SvgGen005 from "@/components/svgs/SvgGen005";
import React from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faTractor,
  faFileInvoiceDollar,
  faUsers,
  faBookUser,
  faCalendarArrowUp,
  faDisplayCode,
  faFolders,
  faCarBurst,
  faTruckTow,
  faMoneyCheckPen,
  faMoneyBillTransfer
} from "@fortawesome/pro-duotone-svg-icons";


export function AsideMenuMain() {

  return (
    <>
      <AsideMenuItem
        to="/dashboard"
        icon={<SvgGen005 />}
        title={"Dashboard"}
      />

      {/* Order Management */}
      <div className="menu-item">
        <div className="menu-content pt-8 pb-2">
          <span className="menu-section text-muted text-uppercase fs-8 ls-1">Dispatch</span>
        </div>
      </div>

      {/* Dispatch */}
      <AsideMenuItemWithSub
        to="#"
        title="Dispatch"
        icon={<FontAwesomeIcon icon={faCalendarArrowUp} />}
      >

        {/* Dispatch Management */}
        <AsideMenuItem to="/dispatch" title="Dispatch Management" hasBullet={true} />

        {/* Driver Management */}
        <AsideMenuItem to="/dispatch/driver-management" title="Driver Management" hasBullet={true} />

        {/* Movements */}
        <AsideMenuItem to="/dispatch/movements" title="Movements" hasBullet={true} />

        {/* Service Incidents */}
        <AsideMenuItem to="/service-incidents" title="Service Incidents" hasBullet={true} />

      </AsideMenuItemWithSub>

      {/* Equipment */}
      <AsideMenuItemWithSub
        to="#"
        title="Equipment"
        icon={<FontAwesomeIcon icon={faTractor} />}
      >

        {/* Equipment Management */}
        <AsideMenuItem to="/equipment" title="Equipment Management" hasBullet={true} />

        {/* Equipment Maintenance Plans */}
        <AsideMenuItem to="/equipment" title="Equipment Maint. Plans" hasBullet={true} />
      </AsideMenuItemWithSub>

      {/* Dispatch Master Files */}
      <AsideMenuItemWithSub
        to="/#"
        title="Master Files"
        icon={<FontAwesomeIcon icon={faFolders} />}
      >

        {/* Comment Types */}
        <AsideMenuItem to="/dispatch/comment-types" title="Comment Types" hasBullet={true} />

        {/* Delay Codes */}
        <AsideMenuItem to="/dispatch/delay-codes" title="Delay Codes" hasBullet={true} />

        {/* Equipment Manufacturers */}
        <AsideMenuItem to="/dispatch/equipment-manufacturers" title="Equipment Manufacturers" hasBullet={true} />

        {/* Equipment Types */}
        <AsideMenuItem to="/equipment/equipment-types" title="Equipment Types" hasBullet={true} />

        {/* Fleet Codes */}
        <AsideMenuItem to="/dispatch/fleet-codes" title="Fleet Codes" hasBullet={true} />

        {/* Rates */}
        <AsideMenuItem to="/dispatch/rates" title="Rates" hasBullet={true} />

        {/* Qualifier Codes */}
        <AsideMenuItem to="/dispatch/qualifier-codes" title="Qualifier Codes" hasBullet={true} />

        {/* Locations */}
        <AsideMenuItem to="/dispatch/locations" title="Locations" hasBullet={true} />

        {/* Commodities */}
        <AsideMenuItem to="/dispatch/commodities" title="Commodities" hasBullet={true} />

        {/* Hazardous Material */}
        <AsideMenuItem to="/dispatch/hazardous-material" title="Hazardous Materials" hasBullet={true} />

        {/* Reason Codes */}
        <AsideMenuItem to="/dispatch/reason-codes" title="Reason Codes" hasBullet={true} />

        {/* Order Types */}
        <AsideMenuItem to="/dispatch/order-types" title="Order Types" hasBullet={true} />

        {/* Comment Types */}
        <AsideMenuItem to="/dispatch/comment-types" title="Comment Types" hasBullet={true} />

      </AsideMenuItemWithSub>

      {/* Billing & AR */}
      <div className="menu-item">
        <div className="menu-content pt-8 pb-2">
          <span className="menu-section text-muted text-uppercase fs-8 ls-1">Billing & AR</span>
        </div>
      </div>
      {/* Billing */}
      <AsideMenuItemWithSub
        to="#"
        title="Billing"
        icon={<FontAwesomeIcon icon={faFileInvoiceDollar} />}
      >
        {/* Billing Workflow */}
        <AsideMenuItem to="/billing/transfer" title="Billing Workflow" hasBullet={true} />

        {/* Transfer to Billing */}
        <AsideMenuItem to="/billing/transfer" title="Transfer to Billing" hasBullet={true} />

        {/* Billing Processing */}
        <AsideMenuItem to="/billing/processing" title="Billing Processing" hasBullet={true} />

        {/* Billing History */}
        <AsideMenuItem to="/billing/history" title="Billing History" hasBullet={true} />

        {/* Billing Exceptions */}
        <AsideMenuItem to="/billing/exceptions" title="Billing Exceptions" hasBullet={true} />

        {/* Billing Transfer Logs */}
        <AsideMenuItem to="/billing/transfer-logs" title="Billing Transfer Logs" hasBullet={true} />
      </AsideMenuItemWithSub>

      {/* Customers */}
      <AsideMenuItemWithSub
        to="#"
        title="Customers"
        icon={<FontAwesomeIcon icon={faBookUser} />}
      >
        {/* Customers */}
        <AsideMenuItem to="/customers" title="Customers" hasBullet={true} />

        {/* Customer Billing Profiles */}
        <AsideMenuItem to="/customer/billing-profiles" title="Customer Billing Profiles" hasBullet={true} />

        {/* Customer Email Profiles */}
        <AsideMenuItem to="/customers/email-profiles" title="Customer Email Profiles" hasBullet={true} />

        {/* Customer Fuel Profiles */}
        <AsideMenuItem to="/customers/fuel-profiles" title="Customer Fuel Profiles" hasBullet={true} />

        {/* Customer Fuel Tables */}
        <AsideMenuItem to="/customers/fuel-tables" title="Customer Fuel Tables" hasBullet={true} />

        {/* Customer Rule Profiles */}
        <AsideMenuItem to="/customers/rule-profiles" title="Customer Rule Profiles" hasBullet={true} />
      </AsideMenuItemWithSub>
      {/* Billing & AR Master Files */}
      <AsideMenuItemWithSub
        to="/#"
        title="Master Files"
        icon={<FontAwesomeIcon icon={faFolders} />}
      >
        {/* Charge Types */}
        <AsideMenuItem to="/billing/charge-types" title="Charge Types" hasBullet={true} />

        {/* Document Classifications */}
        <AsideMenuItem to="/billing/document-class" title="Document Classifications" hasBullet={true} />

        {/* Other Charges */}
        <AsideMenuItem to="/billing/other-charges" title="Other Charges" hasBullet={true} />
      </AsideMenuItemWithSub>

      {/* General Ledger */}
      <div className="menu-item">
        <div className="menu-content pt-8 pb-2">
          <span className="menu-section text-muted text-uppercase fs-8 ls-1">General Ledger</span>
        </div>
      </div>

      {/* General Journal */}
      <AsideMenuItem
        to="/gl/general-journal"
        title="General Journal"
        icon={<FontAwesomeIcon icon={faMoneyCheckPen} />}
      />

      {/* Recurring Journal */}
      <AsideMenuItem
        to="/gl/general-journal"
        title="Recurring Journal"
        icon={<FontAwesomeIcon icon={faMoneyBillTransfer} />}
      />

      {/* General Ledger Master Files */}
      <AsideMenuItemWithSub
        to="/#"
        title="Master Files"
        icon={<FontAwesomeIcon icon={faFolders} />}
      >
        {/* Division Codes */}
        <AsideMenuItem to="/gl/division-codes" title="Division Codes" hasBullet={true} />

        {/* General Ledger Accounts */}
        <AsideMenuItem to="/gl/gl-accounts" title="General Ledger Accounts" hasBullet={true} />

        {/* Revenue Codes */}
        <AsideMenuItem to="/gl/revenue-codes" title="Revenue Codes" hasBullet={true} />
      </AsideMenuItemWithSub>

      {/* Safety & Fuel Tax */}
      <div className="menu-item">
        <div className="menu-content pt-8 pb-2">
          <span className="menu-section text-muted text-uppercase fs-8 ls-1">Safety & Fuel Tax</span>
        </div>
      </div>
      {/* Accidents & Incidents */}
      <AsideMenuItemWithSub
        to="/#"
        title="Accidents & Incidents"
        icon={<FontAwesomeIcon icon={faTruckTow} />}
      >
        {/* Vehicle Accidents */}
        <AsideMenuItem to="/safety/vehicle-accidents" title="Vehicle Accidents" hasBullet={true} />

        {/* Policy Holder */}
        <AsideMenuItem to="/safety/policy-holder" title="Policy Holder" hasBullet={true} />

        {/* Accident Register */}
        <AsideMenuItem to="/safety/accident-register" title="Accident Register" hasBullet={true} />
      </AsideMenuItemWithSub>

      {/* Safety Management */}
      <AsideMenuItemWithSub
        to="/#"
        title="Safety Management"
        icon={<FontAwesomeIcon icon={faCarBurst} />}
      >
        {/* Vehicle Accidents */}
        <AsideMenuItem to="/safety/inspections" title="Vehicle Accidents" hasBullet={true} />

        {/* Policy Holder */}
        <AsideMenuItem to="/safety/policy-holder" title="Policy Holder" hasBullet={true} />

        {/* Accident Register */}
        <AsideMenuItem to="/safety/accident-register" title="Accident Register" hasBullet={true} />
      </AsideMenuItemWithSub>

      {/* Safety & Fuel Tax Master Files */}
      <AsideMenuItemWithSub
        to="/#"
        title="Master Files"
        icon={<FontAwesomeIcon icon={faFolders} />}
      >
        {/* Violation Codes */}
        <AsideMenuItem to="/safety/violation-codes" title="Violation Codes" hasBullet={true} />

        {/* Safety Codes */}
        <AsideMenuItem to="/safety/safety-codes" title="Safety Codes" hasBullet={true} />
      </AsideMenuItemWithSub>

      {/* Administration */}
      <div className="menu-item">
        <div className="menu-content pt-8 pb-2">
          <span className="menu-section text-muted text-uppercase fs-8 ls-1">Administration</span>
        </div>
      </div>

      {/* User Management */}
      <AsideMenuItemWithSub
        to="/admin/users-management/"
        title="User Management"
        icon={<FontAwesomeIcon icon={faUsers} />}
      >
        {/* Manage Users */}
        <AsideMenuItem to="/admin/user-management" title="Manage Users" hasBullet={true} />

        {/* Manage Job Titles */}
        <AsideMenuItem to="/admin/job-titles" title="Manage Job Titles" hasBullet={true} />

        {/* Manage Roles */}
        <AsideMenuItem to="/admin/role-management" title="Manage Roles" hasBullet={true} />

        {/* Manage Permissions */}
        <AsideMenuItem to="/admin/permission-management" title="Permissions" hasBullet={true} />

        {/* Manage Tokens */}
        <AsideMenuItem to="/admin/token-management" title="Web Tokens" hasBullet={true} />
      </AsideMenuItemWithSub>

      {/* System Management */}
      <AsideMenuItem
        to="/admin/system-management"
        icon={<FontAwesomeIcon icon={faDisplayCode} />}
        title="System Management"
      />
      <div className="menu-item">
        <div className="menu-content">
          <div className="separator mx-1 my-4"></div>
        </div>
      </div>

      {/* Documentation */}
      <div className="menu-item">
        <a
          target="_blank"
          className="menu-link"
          href={process.env.REACT_APP_PREVIEW_DOCS_URL + "/docs/changelog"}
        >
          <span className="menu-icon">
            <SvgGen005 />
          </span>
          <span className="menu-title">Changelog {process.env.NEXT_PUBLIC_APP_VERSION}</span>
        </a>
      </div>
    </>
  );
}
