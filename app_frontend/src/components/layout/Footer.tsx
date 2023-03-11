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
import { FC } from "react";
import { useLayout } from "@/utils/layout/LayoutProvider";

const Footer: FC = () => {
  const { classes } = useLayout();
  return (
    <div className="footer py-4 d-flex flex-lg-column" id="mt_footer">
      <div
        className={` container-fluid  d-flex flex-column flex-md-row align-items-center justify-content-between`}
      >
        <div className="text-dark order-2 order-md-1">
          <span className="text-muted fw-bold me-2">{new Date().getFullYear()} &copy;</span>
          <a href="#" className="text-gray-800 text-hover-primary">
            Monta LLC.
          </a>
        </div>

        <ul className="menu menu-gray-600 menu-hover-primary fw-bold order-1">
          <li className="menu-item">
            <a href="#" className="menu-link ps-0 pe-2">
              About
            </a>
          </li>
          <li className="menu-item">
            <a href="#" className="menu-link pe-0 pe-2">
              Contact
            </a>
          </li>
          <li className="menu-item">
            <a href="#" className="menu-link pe-0">
              Purchase
            </a>
          </li>
        </ul>
      </div>
    </div>
  );
};

export { Footer };
