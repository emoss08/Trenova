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
import { useState, useEffect } from "react";
import noUiSlider from "nouislider";
import { DefaultTitle } from "./page-title/DefaultTitle";
import { useLayout } from "@/utils/layout/LayoutProvider";
import Link from "next/link";
import { ThemeModeSwitcher } from "@/components/partials/layout/theme-mode/ThemeModeSwitcher";
import { MTSVG } from "@/components/elements/MTSVG";
import SvgFil010 from "@/components/svgs/SvgFil010";
import SvgFil003 from "@/components/svgs/SvgFil003";
import SvgFil005 from "@/components/svgs/SvgFil005";


const HeaderToolbar = () => {
  const { classes } = useLayout();
  const [status, setStatus] = useState<string>("1");

  useEffect(() => {
    const rangeSlider = document.querySelector("#mt_toolbar_slider");
    const rangeSliderValueElement = document.querySelector("#mt_toolbar_slider_value");

    if (!rangeSlider || !rangeSliderValueElement) {
      return;
    }

    // @ts-ignore
    noUiSlider.create(rangeSlider, {
      start: [5],
      connect: [true, false],
      step: 1,
      format: {
        to: function(value) {
          const val = +value;
          return Math.round(val).toString();
        },
        from: function(value) {
          return value;
        }
      },
      range: {
        min: [1],
        max: [10]
      }
    });

    // @ts-ignore
    rangeSlider.noUiSlider.on("update", function(values, handle) {
      rangeSliderValueElement.innerHTML = values[handle];
    });

    const handle = rangeSlider.querySelector(".noUi-handle");
    if (handle) {
      handle.setAttribute("tabindex", "0");
    }

    // @ts-ignore
    handle.addEventListener("click", function() {
      // @ts-ignore
      this.focus();
    });

    // @ts-ignore
    handle.addEventListener("keydown", function(event) {
      // @ts-ignore
      const value = Number(rangeSlider.noUiSlider.get());
      // @ts-ignore
      switch (event.which) {
        case 37:
          // @ts-ignore
          rangeSlider.noUiSlider.set(value - 1);
          break;
        case 39:
          // @ts-ignore
          rangeSlider.noUiSlider.set(value + 1);
          break;
      }
    });
    return () => {
      // @ts-ignore
      rangeSlider.noUiSlider.destroy();
    };
  }, []);

  return (
    <div className="toolbar d-flex align-items-stretch">
      <div
        className={`${classes.headerContainer.join(
          " "
        )} py-6 py-lg-0 d-flex flex-column flex-lg-row align-items-lg-stretch justify-content-lg-between`}
      >
        <DefaultTitle />
        <div className="d-flex align-items-stretch overflow-auto pt-3 pt-lg-0">
          <div className="d-flex align-items-center">
            <span className="fs-7 fw-bolder text-gray-700 pe-4 text-nowrap d-none d-xxl-block">
              Sort By:
            </span>

            <select
              className="form-select form-select-sm form-select-solid w-100px w-xxl-125px"
              data-control="select2"
              data-placeholder="Latest"
              data-hide-search="true"
              defaultValue={status}
              onChange={(e) => setStatus(e.target.value)}
            >
              <option value=""></option>
              <option value="1">Latest</option>
              <option value="2">In Progress</option>
              <option value="3">Done</option>
            </select>
          </div>

          <div className="d-flex align-items-center">
            <div className="bullet bg-secondary h-35px w-1px mx-5"></div>

            <span className="fs-7 text-gray-700 fw-bolder d-none d-sm-block">
              Impact <span className="d-none d-xxl-inline">Level</span>:
            </span>

            <div className="d-flex align-items-center ps-4" id="mt_toolbar">
              <div
                id="mt_toolbar_slider"
                className="noUi-target noUi-target-primary w-75px w-xxl-150px noUi-sm noUi-ltr noUi-horizontal noUi-txt-dir-ltr"
              ></div>

              <span
                id="mt_toolbar_slider_value"
                className="d-flex flex-center bg-light-primary rounded-circle w-35px h-35px ms-4 fs-7 fw-bolder text-primary"
                data-bs-toggle="tooltip"
                data-bs-placement="top"
                title="Set impact level"
              ></span>
            </div>

            <div className="bullet bg-secondary h-35px w-1px mx-5"></div>
          </div>

          <div className="d-flex align-items-center">
            <span className="fs-7 text-gray-700 fw-bolder pe-3 d-none d-xxl-block">
              Quick Tools:
            </span>

            <div className="d-flex">
              <a
                href="#"
                className="btn btn-sm btn-icon btn-icon-muted btn-active-icon-primary"
                data-bs-toggle="modal"
                data-bs-target="#mt_modal_invite_friends"
              >
                <MTSVG icon={<SvgFil003 />} className={"svg-icon-1"} />
              </a>

              <div className="d-flex align-items-center">
                <Link href="#" className="btn btn-sm btn-icon btn-icon-muted btn-active-icon-primary">
                  <MTSVG icon={<SvgFil005 />} className={"svg-icon-1"} />
                </Link>
              </div>

              <div className="d-flex align-items-center">
                <Link href="#" className="btn btn-sm btn-icon btn-icon-muted btn-active-icon-primary">
                  <MTSVG icon={<SvgFil010 />} className={"svg-icon-1"} />
                </Link>
              </div>

              <div className="d-flex align-items-center">
                <ThemeModeSwitcher toggleBtnClass="btn btn-sm btn-icon btn-icon-muted btn-active-icon-primary" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export { HeaderToolbar };
