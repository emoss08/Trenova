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

/* eslint-disable react-hooks/exhaustive-deps */
import clsx from "clsx";
import Link from "next/link";
import { HeaderToolbar } from "./HeaderToolbar";
import { useLayout } from "@/utils/layout/LayoutProvider";
import Image from "next/image";
import defaultDark from "../../../../public/media/logos/default-dark.svg";
import { MTSVG } from "@/components/elements/MTSVG";
import SvgArr092 from "@/components/svgs/SvgArr092";
import SvgArr076 from "@/components/svgs/SvgArr076";
import SvgAbs015 from "@/components/svgs/SvgAbs015";

export function HeaderWrapper() {
  const { config, classes, attributes } = useLayout();
  const { aside } = config;

  return (
    <div
      id="mt_header"
      className={clsx("header", classes.header.join(" "), "align-items-stretch")}
      {...attributes.headerMenu}
    >
      {/* begin::Brand */}
      <div className="header-brand">
        {/* begin::Logo */}
        <Link href="/">
          <Image
            alt="Logo"
            src={defaultDark}
            className="h-25px h-lg-25px"
            style={{
              maxHeight: "125px",
              maxWidth: "125px",
            }}
          />
        </Link>
        {/* end::Logo */}

        {aside.minimize && (
          <div
            id="mt_aside_toggle"
            className="btn btn-icon w-auto px-0 btn-active-color-primary aside-minimize"
            data-mt-toggle="true"
            data-mt-toggle-state="active"
            data-mt-toggle-target="body"
            data-mt-toggle-name="aside-minimize"
          >
            <MTSVG icon={<SvgArr092 />} className="svg-icon-1 me-n1 minimize-default"
            />
            <MTSVG icon={<SvgArr076 />} className="svg-icon-1 minimize-active"
            />
          </div>
        )}

        {/* begin::Aside toggle */}
        <div className="d-flex align-items-center d-lg-none ms-n3 me-1" title="Show aside menu">
          <div
            className="btn btn-icon btn-active-color-primary w-30px h-30px"
            id="mt_aside_mobile_toggle"
          >
            <MTSVG icon={<SvgAbs015 />} className="svg-icon-1" />
          </div>
        </div>
        {/* end::Aside toggle */}
      </div>
      {/* end::Brand */}
      <HeaderToolbar />
    </div>
  );
}
