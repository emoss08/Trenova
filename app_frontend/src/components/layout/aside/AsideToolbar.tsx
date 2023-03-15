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

import { authStore } from "@/utils/providers/AuthGuard";
import { HeaderUserMenu, Search } from "@/components/partials";
import { MTSVG } from "@/components/elements/MTSVG";
import SvgCod001 from "@/components/svgs/SvgCod001";
import { MontaSymbol } from "@/components/partials/MontaSymbol";

/* eslint-disable jsx-a11y/anchor-is-valid */
const AsideToolbar = () => {
  const [user] = authStore.use("user");
  
  return (
    <>
      <div className="aside-user d-flex align-items-sm-center justify-content-center py-5">
        <MontaSymbol colorClass={'bg-primary text-inverse-primary'} text={user?.first_name[0]} />

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
                <MTSVG icon={<SvgCod001 />} className="svg-icon-muted svg-icon-1" />
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
