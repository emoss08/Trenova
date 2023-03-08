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

import clsx from "clsx";
import Link from "next/link";
import Image from "next/image";
import gen062 from "../../../../../public/media/icons/duotune/general/gen062.svg";
import gen061 from "../../../../../public/media/icons/duotune/general/gen061.svg";
import gen060 from "../../../../../public/media/icons/duotune/general/gen060.svg";
import React, { forwardRef, Ref } from "react";
import { systemMode, ThemeModeType, useThemeMode } from "@/utils/providers/ThemeProvider";
/* eslint-disable jsx-a11y/anchor-is-valid */
type Props = {
  toggleBtnClass?: string
  toggleBtnIconClass?: string
  menuPlacement?: string
  menuTrigger?: string
}

const ThemeModeSwitcher = ({
                             toggleBtnClass = "",
                             toggleBtnIconClass = "svg-icon-2",
                             menuPlacement = "bottom-end",
                             menuTrigger = "{default: 'click', lg: 'hover'}"
                           }: Props) => {
  const { mode, menuMode, updateMode, updateMenuMode } = useThemeMode();
  const calculatedMode = mode === "system" ? systemMode : mode;
  const switchMode = (_mode: ThemeModeType) => {
    console.info("switchMode", _mode);
    updateMenuMode(_mode);
    updateMode(_mode);
  };

  type Props = {
    onClick?: (event: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => void
    href?: string
  }

  const ThemeSwitcherButton = forwardRef(
    ({ onClick, href }: Props, ref: Ref<HTMLAnchorElement>) => {
      return (
        <a href={href} onClick={onClick} ref={ref}>
          Click Me
        </a>
      )
    }
  )
  ThemeSwitcherButton.displayName = 'ThemeSwitcherButton'

  return (
    <>
      {/* begin::Menu toggle */}
      <a
        className={clsx("btn btn-icon ", toggleBtnClass)}
        data-kt-menu-trigger={menuTrigger}
        data-kt-menu-attach="parent"
        data-kt-menu-placement={menuPlacement}
        href={"#"}
      >
        {calculatedMode === "dark" && (
          <Image
            src={gen061}
            className={clsx("theme-light-hide", toggleBtnIconClass)}
            alt={"img"}
          />
        )}

        {calculatedMode === "light" && (
          <Image
            src={gen060}
            className={clsx("theme-dark-hide", toggleBtnIconClass)}
            alt={"img"}
          />
        )}
      </a>
      {/* begin::Menu toggle */}

      {/* begin::Menu */}
      <div
        className="menu menu-sub menu-sub-dropdown menu-column menu-rounded menu-title-gray-700 menu-icon-muted menu-active-bg menu-state-primary fw-semibold py-4 fs-base w-175px"
        data-kt-menu="true"
      >
        {/* begin::Menu item */}
        <div className="menu-item px-3 my-0">
          <a
            href="#"
            className={clsx("menu-link px-3 py-2", { active: menuMode === "light" })}
            onClick={() => switchMode("light")}
          >
          <span className="menu-icon" data-kt-element="icon">
            <img src="/media/icons/duotune/general/gen060.svg" className="svg-icon-3" />
          </span>
            <span className="menu-title">Light</span>
          </a>
        </div>
        {/* end::Menu item */}

        {/* begin::Menu item */}
        <div className="menu-item px-3 my-0">
          <a
            href="#"
            className={clsx("menu-link px-3 py-2", { active: menuMode === "dark" })}
            onClick={() => switchMode("dark")}
          >
          <span className="menu-icon" data-kt-element="icon">
            <Image src={gen062} className="svg-icon-3"  alt={"img"}/>
          </span>
            <span className="menu-title">Dark</span>
          </a>
        </div>
        {/* end::Menu item */}

        {/* begin::Menu item */}
        <div className="menu-item px-3 my-0">
          <a
            href="#"
            className={clsx("menu-link px-3 py-2", { active: menuMode === "system" })}
            onClick={() => switchMode("system")}
          >
          <span className="menu-icon" data-kt-element="icon">
            <Image src={gen061} className="svg-icon-3"  alt={"img"}/>
          </span>
            <span className="menu-title">System</span>
          </a>
        </div>
        {/* end::Menu item */}
      </div>
      {/* end::Menu */}
    </>
  );
};

export { ThemeModeSwitcher };
