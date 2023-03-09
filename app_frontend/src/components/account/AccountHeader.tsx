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

/* eslint-disable jsx-a11y/anchor-is-valid */
import React from "react";
import { Dropdown1 } from "@/components/partials/";
import { useRouter } from "next/router";
import Link from "next/link";
import avatar3001 from "../../../public/media/avatars/300-1.jpg";
import Image from "next/image";
import { MTSVG } from "../elements/MTSVG";
import SvgGen026 from "@/components/svgs/SvgGen026";
import SvgCom006 from "@/components/svgs/SvgCom006";
import SvgGen018 from "@/components/svgs/SvgGen018";
import SvgCom011 from "../svgs/SvgCom011";
import SvgArr012 from "../svgs/SvgArr012";
import { authStore } from "@/utils/providers/AuthGuard";

const AccountHeader: React.FC = () => {
  const router = useRouter();
  const [user] = authStore.use("user");

  return (
    <div className="card mb-5 mb-xl-10">
      <div className="card-body pt-9 pb-0">
        <div className="d-flex flex-wrap flex-sm-nowrap mb-3">
          <div className="me-7 mb-4">
            <div className="symbol symbol-100px symbol-lg-160px symbol-fixed position-relative">
              <Image src={avatar3001} alt="Metronic" />
              <div
                className="position-absolute translate-middle bottom-0 start-100 bg-success rounded-circle border border-4 border-light h-20px w-20px"></div>
            </div>
          </div>

          <div className="flex-grow-1">
            <div className="d-flex justify-content-between align-items-start flex-wrap mb-2">
              <div className="d-flex flex-column">
                <div className="d-flex align-items-center mb-2">
                  <a href="#" className="text-gray-800 text-hover-primary fs-2 fw-bolder me-1">
                    {user?.full_name}
                  </a>
                  <a href="#">

                    <MTSVG
                      icon={<SvgGen026 />}
                      className="svg-icon-1 svg-icon-primary"
                    />
                  </a>
                  <a
                    href="#"
                    className="btn btn-sm btn-light-success fw-bolder ms-2 fs-8 py-1 px-3"
                    data-bs-toggle="modal"
                    data-bs-target="#kt_modal_upgrade_plan"
                  >
                    Upgrade to Pro
                  </a>
                </div>

                <div className="d-flex flex-wrap fw-bold fs-6 mb-4 pe-2">
                  <a
                    href="#"
                    className="d-flex align-items-center text-gray-400 text-hover-primary me-5 mb-2"
                  >
                    <MTSVG
                      icon={<SvgCom006 />}
                      className="svg-icon-4 me-1"
                    />
                    {user?.job_title}
                  </a>
                  <a
                    href="#"
                    className="d-flex align-items-center text-gray-400 text-hover-primary me-5 mb-2"
                  >
                    <MTSVG
                      icon={<SvgGen018 />}
                      className="svg-icon-4 me-1"
                    />
                    {user?.full_address}
                  </a>
                  <a
                    href="#"
                    className="d-flex align-items-center text-gray-400 text-hover-primary mb-2"
                  >
                    <MTSVG
                      icon={<SvgCom011 />}
                      className="svg-icon-4 me-1"
                    />
                    {user?.email}
                  </a>
                </div>
              </div>

              <div className="d-flex my-4">
                <a href="#" className="btn btn-sm btn-light me-2" id="kt_user_follow_button">
                  <MTSVG
                    icon={<SvgArr012 />}
                    className="svg-icon-3 d-none"
                  />

                  <span className="indicator-label">Follow</span>
                  <span className="indicator-progress">
                    Please wait...
                    <span className="spinner-border spinner-border-sm align-middle ms-2"></span>
                  </span>
                </a>
                <a
                  href="#"
                  className="btn btn-sm btn-primary me-3"
                  data-bs-toggle="modal"
                  data-bs-target="#kt_modal_offer_a_deal"
                >
                  Hire Me
                </a>
                <div className="me-0">
                  <button
                    className="btn btn-sm btn-icon btn-bg-light btn-active-color-primary"
                    data-kt-menu-trigger="click"
                    data-kt-menu-placement="bottom-end"
                    data-kt-menu-flip="top-end"
                  >
                    <i className="bi bi-three-dots fs-3"></i>
                  </button>
                  <Dropdown1 />
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="d-flex overflow-auto h-55px">
          <ul className="nav nav-stretch nav-line-tabs nav-line-tabs-2x border-transparent fs-5 fw-bolder flex-nowrap">
            <li className="nav-item">
              <Link
                className={
                  `nav-link text-active-primary me-6 ` +
                  (router.pathname === "/crafted/account/overview" && "active")
                }
                href="/crafted/account/overview"
              >
                Overview
              </Link>
            </li>
            <li className="nav-item">
              <Link
                className={
                  `nav-link text-active-primary me-6 ` +
                  (router.pathname === "/account/settings" && "active")
                }
                href="/account/settings"
              >
                Settings
              </Link>
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export { AccountHeader };
