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

import { useLayout } from "@/utils/layout/LayoutProvider";
import { FC } from "react";
import { AsideMenu } from "./AsideMenu";
import { AsideToolbar } from "./AsideToolbar";
import { MTSVG } from "@/components/elements/MTSVG";
import SvgGen005 from "@/components/svgs/SvgGen005";

const AsideDefault: FC = () => {
  const { classes } = useLayout();

  return (
    <div
      id="mt_aside"
      className="aside"
      data-mt-drawer="true"
      data-mt-drawer-name="aside"
      data-mt-drawer-activate="{default: true, lg: false}"
      data-mt-drawer-overlay="true"
      data-mt-drawer-width="{default:'200px', '300px': '250px'}"
      data-mt-drawer-direction="start"
      data-mt-drawer-toggle="#mt_aside_mobile_toggle"
    >
      <div className="aside-toolbar flex-column-auto" id="mt_aside_toolbar">
        <AsideToolbar />
      </div>
      <div className="aside-menu flex-column-fluid">
        <AsideMenu asideMenuCSSClasses={classes.asideMenu} />
      </div>
      <div className="aside-footer flex-column-auto py-5" id="mt_aside_footer">
        <a
          className="btn btn-custom btn-primary w-100"
          target="_blank"
          href={process.env.REACT_APP_PREVIEW_DOCS_URL}
          data-bs-toggle="tooltip"
          data-bs-trigger="hover"
          data-bs-dismiss-="click"
          title="Check out the complete documentation with over 100 components"
        >
          <span className="btn-label">Docs & Components</span>
          <span className="svg-icon btn-icon svg-icon-2">
            <MTSVG icon={<SvgGen005 />} />
          </span>
        </a>
      </div>
    </div>
  );
};

export { AsideDefault };
