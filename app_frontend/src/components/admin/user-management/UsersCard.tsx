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
import SvgGen021 from "@/components/svgs/SvgGen021";

export default function UsersCard() {
  return (
    <>
      <div className="card shadow-sm">
        <div className="card-header">
          <div className="card-title">
            <div className="d-flex align-items-center position-relative my-1">
              <span className="svg-icon svg-icon-1 position-absolute ms-6">
                <MTSVG icon={<SvgGen021 />} />
              </span>
              <input
                type="text"
                data-kt-user-table-filter="search"
                className="form-control form-control-solid w-250px ps-14"
                placeholder="Search user" />
            </div>
          </div>
          <div className="card-toolbar">
            <button type="button" className="btn btn-sm btn-light">
              Action
            </button>
          </div>
        </div>
        <div className="card-body">
          Lorem Ipsum is simply dummy text...
        </div>
        <div className="card-footer">
          Footer
        </div>
      </div>


    </>
  );
}