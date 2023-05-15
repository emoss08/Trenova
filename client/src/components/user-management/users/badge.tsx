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

import React from "react";

interface BadgeProps {
  active: string;
}

const Badge: React.FC<BadgeProps> = ({ active }) => {
  const badgeColor = active ? "fill-green-500" : "fill-rose-500";

  return (
    <>
      <span
        className="inline-flex items-center gap-x-1.5 rounded-md px-2 py-1 text-xs font-medium text-dark dark:text-white ring-1 ring-inset ring-gray-800">
        <svg className={`h-1.5 w-1.5 ${badgeColor}`} viewBox="0 0 6 6" aria-hidden="true">
          <circle cx={3} cy={3} r={3} />
        </svg>
        {active ? "Active" : "Inactive"}
      </span>
    </>
  );
};

export default Badge;