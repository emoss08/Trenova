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
import SvgGen031 from "@/components/svgs/SvgGen031";
import SvgArr078 from "@/components/svgs/SvgArr078";
import UsersAddModal from "@/components/admin/user-management/partials/UsersAddModal"

export default function UsersCardToolbar() {
  return (
    <>
      <div className="card-toolbar">
        <div className="d-flex justify-content-end" data-kt-user-table-toolbar="base">
          <button type="button"
                  className="btn btn-light-primary me-3"
                  data-kt-menu-trigger="click"
                  data-kt-menu-placement="bottom-end">
            <MTSVG icon={<SvgGen031 />} className={"svg-icon-2"} />
            Filter
          </button>
          <button type="button" className="btn btn-light-primary me-3" data-bs-toggle="modal"
                  data-bs-target="#kt_modal_export_users">
            <MTSVG icon={<SvgArr078 />} className={"svg-icon-2"} />
            Export
          </button>
        </div>
        <UsersAddModal />
      </div>
    </>
  );
}