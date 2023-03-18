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

import React, { FC } from "react";
import Link from "next/link";

type Breadcrumb = {
  label: string;
  url?: string;
}

type ToolbarProps = {
  pageTitle: string;
  breadcrumbs: Breadcrumb[]
}

export const MTToolbar: FC<ToolbarProps> = ({ pageTitle, breadcrumbs }) => {
  return (
    <>
      <div id="mt_app_toolbar" className="app-toolbar py-3 py-lg-6 ">
        <div id="mt_app_toolbar_container" className="app-container container-xxl d-flex flex-stack ">
          <div className="page-title d-flex flex-column justify-content-center flex-wrap me-3 ">
            <h1 className="page-heading d-flex text-dark fw-bold fs-3 flex-column justify-content-center my-0">
              {pageTitle}
            </h1>
            <ul className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 my-0 pt-1">
              {breadcrumbs.map((breadcrumb, index) => (
                <React.Fragment key={index}>
                  <li className="breadcrumb-item text-muted">
                    <Link href={breadcrumb.url || "#"} className="text-muted text-hover-primary">
                      {breadcrumb.label}
                    </Link>
                  </li>
                  {index < breadcrumbs.length - 1 && (
                    <li className="breadcrumb-item">
                      <span className="bullet bg-gray-400 w-5px h-2px"></span>
                    </li>
                  )}
                </React.Fragment>
              ))}
            </ul>
          </div>
          <div className="d-flex align-items-center gap-2 gap-lg-3">
            <div className="m-0">
              <a href="../admin/system-management/partials#" className="btn btn-sm btn-flex bg-body btn-color-gray-700 btn-active-color-primary fw-bold"
                 data-kt-menu-trigger="click" data-kt-menu-placement="bottom-end">
                Filter
              </a>
            </div>
            <a href="../admin/system-management/partials#" className="btn btn-sm fw-bold btn-primary" data-bs-toggle="modal"
               data-bs-target="#mt_modal_create_app">
              Create
            </a>
          </div>
        </div>
      </div>
    </>
  );
};