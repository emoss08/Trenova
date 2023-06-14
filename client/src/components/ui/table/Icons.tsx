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

import { MRT_Icons } from "mantine-react-table";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faArrowDownWideShort,
  faBars,
  faBarsStaggered,
  faColumns,
  faCompress,
  faEllipsisH,
  faEllipsisVertical,
  faExpand,
  faEyeSlash,
  faFilter,
  faFilterCircleXmark,
  faSearch,
  faSearchMinus,
  faSortDown,
  faThumbTack,
} from "@fortawesome/pro-duotone-svg-icons";
import React from "react";

export const montaTableIcons: Partial<MRT_Icons> = {
  // Override the default ``mantine-react-table`` icons with custom ones from FontAwesome
  IconArrowDown: (props: any) => (
    <FontAwesomeIcon icon={faSortDown} {...props} />
  ),

  IconClearAll: () => <FontAwesomeIcon icon={faBarsStaggered} />,

  IconTallymark1: () => <FontAwesomeIcon icon={faBars} />,

  IconTallymark2: () => <FontAwesomeIcon icon={faBars} />,

  IconTallymark3: () => <FontAwesomeIcon icon={faBars} />,

  IconTallymark4: () => <FontAwesomeIcon icon={faBars} />,

  IconTallymarks: () => <FontAwesomeIcon icon={faBars} />,

  IconFilter: (props: any) => <FontAwesomeIcon icon={faFilter} {...props} />,

  IconFilterOff: () => <FontAwesomeIcon icon={faFilterCircleXmark} />,

  IconMinimize: () => <FontAwesomeIcon icon={faCompress} />,

  IconMaximize: () => <FontAwesomeIcon icon={faExpand} />,

  IconSearch: (props: any) => <FontAwesomeIcon icon={faSearch} {...props} />,

  IconCircleOff: () => <FontAwesomeIcon icon={faSearchMinus} />,

  IconColumns: () => <FontAwesomeIcon icon={faColumns} />,

  IconDotsVertical: () => <FontAwesomeIcon icon={faEllipsisVertical} />,

  IconDots: () => <FontAwesomeIcon icon={faEllipsisH} />,

  IconArrowsSort: (props: any) => (
    <FontAwesomeIcon icon={faArrowDownWideShort} {...props} /> //props so that style rotation transforms are applied
  ),

  IconPinned: (props: any) => (
    <FontAwesomeIcon icon={faThumbTack} {...props} /> //props so that style rotation transforms are applied
  ),

  IconEyeOff: () => <FontAwesomeIcon icon={faEyeSlash} />,
};
