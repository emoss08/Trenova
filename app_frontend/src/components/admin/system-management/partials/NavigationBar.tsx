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

import { MTSVG } from "@/components/elements/MTSVG";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faMoneyCheckPen
} from "@fortawesome/pro-duotone-svg-icons";
import React from "react";

export const NavigationBar = () => {
  return (
    <>
      <div className={"flex-column flex-lg-row-auto w-100 w-lg-200px w-xl-300px mb-10"}>
        <div className="card card-flush">
          <div className="card-header">
            <div className="card-title">
              <h2 className="mb-0">Navigation</h2>
            </div>
          </div>
          <div className="card-body pt-0">
            <div className="d-flex flex-column text-gray-600">
              <ul
                className="nav nav-tabs nav-pills flex-row border-0 flex-md-column me-5 mb-3 mb-md-0 fs-6 min-w-lg-200px">
                <li className="nav-item w-100 me-0 mb-md-2">
                  <a className="nav-link w-100 btn btn-flex btn-active-light-success" data-bs-toggle="tab"
                     href="#kt_vtab_pane_4">
                    <span className="svg-icon svg-icon-2">
                      <MTSVG icon={<FontAwesomeIcon icon={faMoneyCheckPen} />} />
                    </span>
                    <span className="d-flex flex-column align-items-start">
                      <span className="fs-4 fw-bold">Billing Controls</span>
                      <span className="fs-7"></span>
                    </span>
                  </a>
                </li>
                <li className="nav-item w-100 me-0 mb-md-2">
                  <a className="nav-link w-100 btn btn-flex btn-active-light-info" data-bs-toggle="tab"
                     href="#kt_vtab_pane_5">
                    <span className="svg-icon svg-icon-2">
                      <MTSVG icon={<FontAwesomeIcon icon={faMoneyCheckPen} />} />
                    </span>
                    <span className="d-flex flex-column align-items-start">
                      <span className="fs-4 fw-bold">Link 2</span>
                      <span className="fs-7">Description</span>
                    </span>
                  </a>
                </li>
                <li className="nav-item w-100">
                  <a className="nav-link w-100 btn btn-flex btn-active-light-danger" data-bs-toggle="tab"
                     href="#kt_vtab_pane_6">
                    <span className="svg-icon svg-icon-2"><svg>...</svg></span>
                    <span className="d-flex flex-column align-items-start">
                      <span className="fs-4 fw-bold">Link 3</span>
                      <span className="fs-7">Description</span>
                    </span>
                  </a>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </>
  );
};