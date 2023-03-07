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

import React, { useEffect } from "react";
import { WithChildren } from "@/utils/types";
import Image from "next/image";
import darkLogo from "../../../public/logos/default-dark.svg";
import defaultLogo from "../../../public/logos/default.svg";
import authBgIcon from "../../../public/media/illustrations/sketchy-1/14.png";
import { useTheme } from "next-themes";

export default function AuthLayout({ children }: WithChildren) {

  const { resolvedTheme } = useTheme();

  useEffect(() => {
    document.body.style.backgroundImage = "none";
    return () => {
      document.body.style.backgroundImage = "";
    };
  }, []);

  return (
    <>
      <div
        className="d-flex flex-column flex-column-fluid bgi-position-y-bottom position-x-center bgi-no-repeat bgi-size-contain bgi-attachment-fixed"
        style={{
          backgroundImage: `url(${authBgIcon.src})`,
          minHeight: "100vh",
        }}
      >
        {/* begin::Content */}
        <div className="d-flex flex-center flex-column flex-column-fluid p-10 pb-lg-20">
          {/* begin::Logo */}
          <a href="#" className="mb-12">
            {resolvedTheme === "dark" ? (
              <Image
                alt="Dark Logo"
                src={darkLogo}
                className="h-45px"
              />
            ) : (
              <Image
                alt="Light Logo"
                src={defaultLogo}
                className="h-45px"
              />
            )}
          </a>
          {/* end::Logo */}
          {/* begin::Wrapper */}
          <div className="w-lg-500px bg-body rounded shadow-sm p-10 p-lg-15 mx-auto">
            {children}
          </div>
          {/* end::Wrapper */}
        </div>
        {/* end::Content */}
        {/* begin::Footer */}
        <div className="d-flex flex-center flex-column-auto p-10">
          <div className="d-flex align-items-center fw-semibold fs-6">
            <a href="#" className="text-muted text-hover-primary px-2">
              About
            </a>

            <a href="#" className="text-muted text-hover-primary px-2">
              Contact
            </a>

            <a href="#" className="text-muted text-hover-primary px-2">
              Contact Us
            </a>
          </div>
        </div>
        {/* end::Footer */}
      </div>
    </>
  );
}
