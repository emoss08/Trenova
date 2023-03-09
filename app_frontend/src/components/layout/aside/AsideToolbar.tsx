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

import { authStore } from "@/utils/providers/AuthGuard";
import { HeaderUserMenu, Search } from "@/components/partials";
import Image from "next/image";
import avatar3001 from "../../../../public/media/avatars/300-1.jpg";
import { MTSVG } from "@/components/elements/MTSVG";
import SvgCod001 from "@/components/svgs/SvgCod001";

/* eslint-disable jsx-a11y/anchor-is-valid */
const AsideToolbar = () => {
  const [user] = authStore.use("user");
  return (
    <>
      <div className="aside-user d-flex align-items-sm-center justify-content-center py-5">
        <div className="symbol symbol-50px">
          <Image src={avatar3001} alt="" />
        </div>

        <div className="aside-user-info flex-row-fluid flex-wrap ms-5">
          <div className="d-flex">
            <div className="flex-grow-1 me-2">
              <a href="#" className="text-white text-hover-primary fs-6 fw-bold">
                {user?.first_name} {user?.last_name}
              </a>

              <span className="text-gray-600 fw-bold d-block fs-8 mb-1">{user?.job_title}</span>

              <div className="d-flex align-items-center text-success fs-9">
                <span className="bullet bullet-dot bg-success me-1"></span>online
              </div>
            </div>

            <div className="me-n2">
              <a
                href="#"
                className="btn btn-icon btn-sm btn-active-color-primary mt-n2"
                data-mt-menu-trigger="click"
                data-mt-menu-placement="bottom-start"
                data-mt-menu-overflow="false"
              >
                <MTSVG icon={<SvgCod001 />} className="svg-icon-muted svg-icon-12" />
              </a>

              <HeaderUserMenu />
            </div>
          </div>
        </div>
      </div>

      <div className="aside-search py-5">
        <Search />
      </div>
    </>
  );
};

export { AsideToolbar };
