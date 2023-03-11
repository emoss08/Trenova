/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

import { MTSVG } from "@/components/elements/MTSVG";
import SvgGen031 from "@/components/svgs/SvgGen031";
import SvgArr078 from "@/components/svgs/SvgArr078";
import UsersAddModal from "@/components/admin/user-management/partials/UsersAddModal";

export default function UsersCardToolbar() {
  return (
    <>
      <div className="card-toolbar">
        <div className="d-flex justify-content-end" data-kt-user-table-toolbar="base">
          <button type="button" className="btn btn-light-primary me-3" data-kt-menu-trigger="click"
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