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

import React, { PropsWithChildren } from "react";
import clsx from "clsx";
import { useRouter } from "next/router";
import { checkIsActive } from "@/utils/helpers/RouteHelpers";
import { MTSVG } from "@/components/elements/MTSVG";

type Props = {
  to: string
  title: string
  icon?: React.ReactNode,
  fontIcon?: string
  hasBullet?: boolean
}

const AsideMenuItemWithSub: React.FC<Props & PropsWithChildren> = (
  {
    children,
    to,
    title,
    icon,
    fontIcon,
    hasBullet
  }) => {
  const router = useRouter();
  const isActive = checkIsActive(router.pathname, to);

  return (
    <div
      className={clsx("menu-item", { "here show": isActive }, "menu-accordion")}
      data-mt-menu-trigger="click"
    >
      <span className="menu-link">
        {hasBullet && (
          <span className="menu-bullet">
            <span className="bullet bullet-dot"></span>
          </span>
        )}
        {icon && (
          <span className="menu-icon">
            <MTSVG icon={icon} className="svg-icon-2" />
          </span>
        )}
        {fontIcon && <i className={clsx("bi fs-3", fontIcon)}></i>}
        <span className="menu-title">{title}</span>
        <span className="menu-arrow"></span>
      </span>
      <div className={clsx("menu-sub menu-sub-accordion", { "menu-active-bg": isActive })}>
        {children}
      </div>
    </div>
  );
};

export { AsideMenuItemWithSub };
