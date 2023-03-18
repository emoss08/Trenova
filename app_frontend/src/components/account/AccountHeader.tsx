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
import Button from 'react-bootstrap/Button';

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
                    data-bs-target="#mt_modal_upgrade_plan"
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
                <a href="#" className="btn btn-sm btn-light me-2" id="mt_user_follow_button">
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
                  data-bs-target="#mt_modal_offer_a_deal"
                >
                  Hire Me
                </a>
                <div className="me-0">
                  <Button
                    className="btn btn-sm btn-icon btn-bg-light btn-active-color-primary"
                    data-kt-menu-trigger="click"
                    data-kt-menu-placement="bottom-end"
                    data-kt-menu-flip="top-end"
                  >
                    <i className="bi bi-three-dots fs-3"></i>
                  </Button>
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
