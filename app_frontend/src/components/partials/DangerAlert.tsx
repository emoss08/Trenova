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

import { MTSVG } from "@/components/elements/MTSVG";
import SvgGen050 from "@/components/svgs/SvgGen050";

export const DangerAlert = ({ message, title }: { title: string, message: string }) => {
  return (
    <div className="alert alert-danger d-flex align-items-center p-5 mb-10">
      <MTSVG icon={<SvgGen050 />} className={"svg-icon-2hx svg-icon-danger me-4"} />
      <div className="d-flex flex-column">
        <h4 className="mb-1 text-danger">
          {title}
        </h4>
        <span>
          {message}
        </span>
      </div>
    </div>
  );
};
