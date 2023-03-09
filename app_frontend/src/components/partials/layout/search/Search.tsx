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

import { SearchComponent } from "@/utils/assets/ts/components";
import { FC, useEffect, useRef, useState } from "react";
import { toAbsoluteUrl } from "@/utils/AssetHelpers";
import { MTSVG } from "@/components/elements/MTSVG";
import SvgGen021 from "@/components/svgs/SvgGen021";

const Search: FC = () => {
  const [searchVal, setSearchVal] = useState<string>("");
  const [menuState, setMenuState] = useState<"main" | "advanced" | "preferences">("main");
  const element = useRef<HTMLDivElement | null>(null);
  const wrapperElement = useRef<HTMLDivElement | null>(null);
  const resultsElement = useRef<HTMLDivElement | null>(null);
  const suggestionsElement = useRef<HTMLDivElement | null>(null);
  const emptyElement = useRef<HTMLDivElement | null>(null);

  const processs = (search: SearchComponent) => {
    setTimeout(function() {
      const number = Math.floor(Math.random() * 6) + 1;

      // Hide recently viewed
      suggestionsElement.current!.classList.add("d-none");

      if (number === 3) {
        // Hide results
        resultsElement.current!.classList.add("d-none");
        // Show empty message
        emptyElement.current!.classList.remove("d-none");
      } else {
        // Show results
        resultsElement.current!.classList.remove("d-none");
        // Hide empty message
        emptyElement.current!.classList.add("d-none");
      }

      // Complete search
      search.complete();
    }, 1500);
  };

  const clear = (search: SearchComponent) => {
    // Show recently viewed
    suggestionsElement.current!.classList.remove("d-none");
    // Hide results
    resultsElement.current!.classList.add("d-none");
    // Hide empty message
    emptyElement.current!.classList.add("d-none");
  };

  useEffect(() => {
    // Initialize search handler
    const searchObject = SearchComponent.createInsance("#mt_header_search");

    // Search handler
    searchObject!.on("kt.search.process", processs);

    // Clear handler
    searchObject!.on("kt.search.clear", clear);
  }, []);

  return (
    <>
      <div
        id="mt_header_search"
        className="header-search d-flex align-items-center w-100"
        data-mt-search-keypress="true"
        data-mt-search-min-length="2"
        data-mt-search-enter="enter"
        data-mt-search-layout="menu"
        data-mt-search-responsive="false"
        data-mt-menu-trigger="auto"
        data-mt-menu-permanent="true"
        data-mt-menu-placement="bottom-start"
        data-mt-search="true"
        ref={element}
      >
        <form data-mt-search-element="form" className="w-100 position-relative" autoComplete="off">
          <MTSVG icon={<SvgGen021 />}
                 className="svg-icon-2 search-icon position-absolute top-50 translate-middle-y ms-4" />
          <input
            type="text"
            className="search-input form-control ps-13 fs-7 h-40px"
            name="search"
            value={searchVal}
            onChange={(e) => setSearchVal(e.target.value)}
            placeholder="Quick Search"
            data-mt-search-element="input"
          />
        </form>

        <div
          data-mt-search-element="content"
          className="menu menu-sub menu-sub-dropdown p-7 w-325px w-md-375px"
        >
          <div
            className={`${menuState === "main" ? "" : "d-none"}`}
            ref={wrapperElement}
            data-mt-search-element="wrapper"
          >
            <form
              data-mt-search-element="form"
              className="w-100 position-relative mb-3"
              autoComplete="off"
            >
              <MTSVG icon={<SvgGen021 />}
                     className="svg-icon-2 svg-icon-lg-1 svg-icon-gray-500 position-absolute top-50 translate-middle-y ms-0" />
              <input
                type="text"
                className="form-control form-control-flush ps-10"
                name="search"
                placeholder="Search..."
                data-mt-search-element="input"
              />

              <span
                className="position-absolute top-50 end-0 translate-middle-y lh-0 d-none me-1"
                data-mt-search-element="spinner"
              >
                <span className="spinner-border h-15px w-15px align-middle text-gray-400" />
              </span>

              <span
                className="btn btn-flush btn-active-color-primary position-absolute top-50 end-0 translate-middle-y lh-0 d-none"
                data-mt-search-element="clear"
              >
                <img
                  src="/media/icons/duotune/arrows/arr061.svg"
                  className="svg-icon-2 svg-icon-lg-1 me-0"
                />
              </span>

              <div
                className="position-absolute top-50 end-0 translate-middle-y"
                data-mt-search-element="toolbar"
              >
                <div
                  data-mt-search-element="preferences-show"
                  className="btn btn-icon w-20px btn-sm btn-active-color-primary me-1"
                  data-bs-toggle="tooltip"
                  onClick={() => {
                    setMenuState("preferences");
                  }}
                  title="Show search preferences"
                >
                  <img src="/media/icons/duotune/coding/cod001.svg" className="svg-icon-1" />
                </div>

                <div
                  data-mt-search-element="advanced-options-form-show"
                  className="btn btn-icon w-20px btn-sm btn-active-color-primary"
                  data-bs-toggle="tooltip"
                  onClick={() => {
                    setMenuState("advanced");
                  }}
                  title="Show more search options"
                >
                  <img src="/media/icons/duotune/arrows/arr072.svg" className="svg-icon-2" />
                </div>
              </div>
            </form>

            <div ref={resultsElement} data-mt-search-element="results" className="d-none">
              <div className="scroll-y mh-200px mh-lg-350px">
                <h3 className="fs-5 text-muted m-0 pb-5" data-mt-search-element="category-title">
                  Users
                </h3>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <img src={toAbsoluteUrl("/media/avatars/300-6.jpg")} alt="" />
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Karina Clark</span>
                    <span className="fs-7 fw-bold text-muted">Marketing Manager</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <img src={toAbsoluteUrl("/media/avatars/300-2.jpg")} alt="" />
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Olivia Bold</span>
                    <span className="fs-7 fw-bold text-muted">Software Engineer</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <img src={toAbsoluteUrl("/media/avatars/300-9.jpg")} alt="" />
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Ana Clark</span>
                    <span className="fs-7 fw-bold text-muted">UI/UX Designer</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <img src={toAbsoluteUrl("/media/avatars/300-14.jpg")} alt="" />
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Nick Pitola</span>
                    <span className="fs-7 fw-bold text-muted">Art Director</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <img src={toAbsoluteUrl("/media/avatars/300-11.jpg")} alt="" />
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Edward Kulnic</span>
                    <span className="fs-7 fw-bold text-muted">System Administrator</span>
                  </div>
                </a>

                <h3
                  className="fs-5 text-muted m-0 pt-5 pb-5"
                  data-mt-search-element="category-title"
                >
                  Customers
                </h3>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        className="w-20px h-20px"
                        src={toAbsoluteUrl("/media/svg/brand-logos/volicity-9.svg")}
                        alt=""
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Company Rbranding</span>
                    <span className="fs-7 fw-bold text-muted">UI Design</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        className="w-20px h-20px"
                        src={toAbsoluteUrl("/media/svg/brand-logos/tvit.svg")}
                        alt=""
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Company Re-branding</span>
                    <span className="fs-7 fw-bold text-muted">Web Development</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        className="w-20px h-20px"
                        src={toAbsoluteUrl("/media/svg/misc/infography.svg")}
                        alt=""
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Business Analytics App</span>
                    <span className="fs-7 fw-bold text-muted">Administration</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        className="w-20px h-20px"
                        src={toAbsoluteUrl("/media/svg/brand-logos/leaf.svg")}
                        alt=""
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">EcoLeaf App Launch</span>
                    <span className="fs-7 fw-bold text-muted">Marketing</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        className="w-20px h-20px"
                        src={toAbsoluteUrl("/media/svg/brand-logos/tower.svg")}
                        alt=""
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column justify-content-start fw-bold">
                    <span className="fs-6 fw-bold">Tower Group Website</span>
                    <span className="fs-7 fw-bold text-muted">Google Adwords</span>
                  </div>
                </a>

                <h3
                  className="fs-5 text-muted m-0 pt-5 pb-5"
                  data-mt-search-element="category-title"
                >
                  Projects
                </h3>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/general/gen005.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <span className="fs-6 fw-bold">Si-Fi Project by AU Themes</span>
                    <span className="fs-7 fw-bold text-muted">#45670</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/general/gen032.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <span className="fs-6 fw-bold">Shopix Mobile App Planning</span>
                    <span className="fs-7 fw-bold text-muted">#45690</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/communication/com012.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <span className="fs-6 fw-bold">Finance Monitoring SAAS Discussion</span>
                    <span className="fs-7 fw-bold text-muted">#21090</span>
                  </div>
                </a>

                <a
                  href="/#"
                  className="d-flex text-dark text-hover-primary align-items-center mb-5"
                >
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/communication/com006.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <span className="fs-6 fw-bold">Dashboard Analitics Launch</span>
                    <span className="fs-7 fw-bold text-muted">#34560</span>
                  </div>
                </a>
              </div>
            </div>

            <div ref={suggestionsElement} className="mb-4" data-mt-search-element="main">
              <div className="d-flex flex-stack fw-bold mb-4">
                <span className="text-muted fs-6 me-2">Recently Searched:</span>
              </div>

              <div className="scroll-y mh-200px mh-lg-325px">
                <div className="d-flex align-items-center mb-5">
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/electronics/elc004.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <a href="/#" className="fs-6 text-gray-800 text-hover-primary fw-bold">
                      BoomApp by Keenthemes
                    </a>
                    <span className="fs-7 text-muted fw-bold">#45789</span>
                  </div>
                </div>

                <div className="d-flex align-items-center mb-5">
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/graphs/gra001.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <a href="/#" className="fs-6 text-gray-800 text-hover-primary fw-bold">
                      Kept API Project Meeting
                    </a>
                    <span className="fs-7 text-muted fw-bold">#84050</span>
                  </div>
                </div>

                <div className="d-flex align-items-center mb-5">
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/graphs/gra006.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <a href="/#" className="fs-6 text-gray-800 text-hover-primary fw-bold">
                      KPI Monitoring App Launch
                    </a>
                    <span className="fs-7 text-muted fw-bold">#84250</span>
                  </div>
                </div>

                <div className="d-flex align-items-center mb-5">
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/graphs/gra002.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <a href="/#" className="fs-6 text-gray-800 text-hover-primary fw-bold">
                      Project Reference FAQ
                    </a>
                    <span className="fs-7 text-muted fw-bold">#67945</span>
                  </div>
                </div>

                <div className="d-flex align-items-center mb-5">
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/communication/com010.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <a href="/#" className="fs-6 text-gray-800 text-hover-primary fw-bold">
                      FitPro App Development
                    </a>
                    <span className="fs-7 text-muted fw-bold">#84250</span>
                  </div>
                </div>

                <div className="d-flex align-items-center mb-5">
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/finance/fin001.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <a href="/#" className="fs-6 text-gray-800 text-hover-primary fw-bold">
                      Shopix Mobile App
                    </a>
                    <span className="fs-7 text-muted fw-bold">#45690</span>
                  </div>
                </div>

                <div className="d-flex align-items-center mb-5">
                  <div className="symbol symbol-40px me-4">
                    <span className="symbol-label bg-light">
                      <img
                        src="/media/icons/duotune/graphs/gra002.svg"
                        className="svg-icon-2 svg-icon-primary"
                      />
                    </span>
                  </div>

                  <div className="d-flex flex-column">
                    <a href="/#" className="fs-6 text-gray-800 text-hover-primary fw-bold">
                      `&quot;`Landing UI Design`&quot;` Launch
                    </a>
                    <span className="fs-7 text-muted fw-bold">#24005</span>
                  </div>
                </div>
              </div>
            </div>

            <div ref={emptyElement} data-mt-search-element="empty" className="text-center d-none">
              <div className="pt-10 pb-10">
                <img
                  src="/media/icons/duotune/files/fil024.svg"
                  className="svg-icon-4x opacity-50"
                />
              </div>

              <div className="pb-15 fw-bold">
                <h3 className="text-gray-600 fs-5 mb-2">No result found</h3>
                <div className="text-muted fs-7">Please try again with a different query</div>
              </div>
            </div>
          </div>

          <form className={`pt-1 ${menuState === "advanced" ? "" : "d-none"}`}>
            <h3 className="fw-bold text-dark mb-7">Advanced Search</h3>

            <div className="mb-5">
              <input
                type="text"
                className="form-control form-control-sm form-control-solid"
                placeholder="Contains the word"
                name="query"
              />
            </div>

            <div className="mb-5">
              <div className="nav-group nav-group-fluid">
                <label>
                  <input
                    type="radio"
                    className="btn-check"
                    name="type"
                    value="has"
                    defaultChecked
                  />
                  <span className="btn btn-sm btn-color-muted btn-active btn-active-primary">
                    All
                  </span>
                </label>

                <label>
                  <input type="radio" className="btn-check" name="type" value="users" />
                  <span className="btn btn-sm btn-color-muted btn-active btn-active-primary px-4">
                    Users
                  </span>
                </label>

                <label>
                  <input type="radio" className="btn-check" name="type" value="orders" />
                  <span className="btn btn-sm btn-color-muted btn-active btn-active-primary px-4">
                    Orders
                  </span>
                </label>

                <label>
                  <input type="radio" className="btn-check" name="type" value="projects" />
                  <span className="btn btn-sm btn-color-muted btn-active btn-active-primary px-4">
                    Projects
                  </span>
                </label>
              </div>
            </div>

            <div className="mb-5">
              <input
                type="text"
                name="assignedto"
                className="form-control form-control-sm form-control-solid"
                placeholder="Assigned to"
              />
            </div>

            <div className="mb-5">
              <input
                type="text"
                name="collaborators"
                className="form-control form-control-sm form-control-solid"
                placeholder="Collaborators"
              />
            </div>

            <div className="mb-5">
              <div className="nav-group nav-group-fluid">
                <label>
                  <input
                    type="radio"
                    className="btn-check"
                    name="attachment"
                    value="has"
                    defaultChecked
                  />
                  <span className="btn btn-sm btn-color-muted btn-active btn-active-primary">
                    Has attachment
                  </span>
                </label>

                <label>
                  <input type="radio" className="btn-check" name="attachment" value="any" />
                  <span className="btn btn-sm btn-color-muted btn-active btn-active-primary px-4">
                    Any
                  </span>
                </label>
              </div>
            </div>

            <div className="mb-5">
              <select
                name="timezone"
                aria-label="Select a Timezone"
                data-control="select2"
                data-placeholder="date_period"
                className="form-select form-select-sm form-select-solid"
              >
                <option value="next">Within the next</option>
                <option value="last">Within the last</option>
                <option value="between">Between</option>
                <option value="on">On</option>
              </select>
            </div>

            <div className="row mb-8">
              <div className="col-6">
                <input
                  type="number"
                  name="date_number"
                  className="form-control form-control-sm form-control-solid"
                  placeholder="Lenght"
                />
              </div>

              <div className="col-6">
                <select
                  name="date_typer"
                  aria-label="Select a Timezone"
                  data-control="select2"
                  data-placeholder="Period"
                  className="form-select form-select-sm form-select-solid"
                >
                  <option value="days">Days</option>
                  <option value="weeks">Weeks</option>
                  <option value="months">Months</option>
                  <option value="years">Years</option>
                </select>
              </div>
            </div>

            <div className="d-flex justify-content-end">
              <button
                onClick={(e) => {
                  e.preventDefault();
                  setMenuState("main");
                }}
                className="btn btn-sm btn-light fw-bolder btn-active-light-primary me-2"
              >
                Cancel
              </button>

              <a
                href="/#"
                className="btn btn-sm fw-bolder btn-primary"
                data-mt-search-element="advanced-options-form-search"
              >
                Search
              </a>
            </div>
          </form>

          <form className={`pt-1 ${menuState === "preferences" ? "" : "d-none"}`}>
            <h3 className="fw-bold text-dark mb-7">Search Preferences</h3>

            <div className="pb-4 border-bottom">
              <label className="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack">
                <span className="form-check-label text-gray-700 fs-6 fw-bold ms-0 me-2">
                  Projects
                </span>

                <input className="form-check-input" type="checkbox" value="1" defaultChecked />
              </label>
            </div>

            <div className="py-4 border-bottom">
              <label className="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack">
                <span className="form-check-label text-gray-700 fs-6 fw-bold ms-0 me-2">
                  Targets
                </span>
                <input className="form-check-input" type="checkbox" value="1" defaultChecked />
              </label>
            </div>

            <div className="py-4 border-bottom">
              <label className="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack">
                <span className="form-check-label text-gray-700 fs-6 fw-bold ms-0 me-2">
                  Affiliate Programs
                </span>
                <input className="form-check-input" type="checkbox" value="1" />
              </label>
            </div>

            <div className="py-4 border-bottom">
              <label className="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack">
                <span className="form-check-label text-gray-700 fs-6 fw-bold ms-0 me-2">
                  Referrals
                </span>
                <input className="form-check-input" type="checkbox" value="1" defaultChecked />
              </label>
            </div>

            <div className="py-4 border-bottom">
              <label className="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack">
                <span className="form-check-label text-gray-700 fs-6 fw-bold ms-0 me-2">Users</span>
                <input className="form-check-input" type="checkbox" />
              </label>
            </div>

            <div className="d-flex justify-content-end pt-7">
              <button
                onClick={(e) => {
                  e.preventDefault();
                  setMenuState("main");
                }}
                className="btn btn-sm btn-light fw-bolder btn-active-light-primary me-2"
              >
                Cancel
              </button>
              <button className="btn btn-sm fw-bolder btn-primary">Save Changes</button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
};

export { Search };
