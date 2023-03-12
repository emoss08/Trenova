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

import { useLayout } from "@/utils/layout/LayoutProvider";
import { FC } from "react";
import { AsideToolbar } from "./AsideToolbar";
import { MTSVG } from "@/components/elements/MTSVG";
import SvgGen005 from "@/components/svgs/SvgGen005";
import { AsideMenu } from "./AsideMenu";

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
