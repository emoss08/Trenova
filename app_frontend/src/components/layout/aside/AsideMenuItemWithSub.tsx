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

import React, { PropsWithChildren } from "react";
import clsx from "clsx";
import { useRouter } from "next/router";
import { checkIsActive } from "@/utils/RouteHelpers";
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
