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
  faAnglesDown,
  faBars,
  faBarsStaggered,
  faChevronDown,
  faCircleXmark,
  faColumns,
  faCompress,
  faEdit,
  faEllipsisH,
  faEllipsisV,
  faExpand,
  faEyeSlash,
  faFilter,
  faFilterCircleXmark,
  faFloppyDisk,
  faGrip,
  faLayerGroup,
  faSearch,
  faSearchMinus,
  faSort,
  faSortDown,
  faSortUp,
  faTextWidth,
  faThumbTack,
  faX,
} from "@fortawesome/pro-duotone-svg-icons";
import React from "react";

export const montaTableIcons: Partial<MRT_Icons> = {
  // Override the default ``mantine-react-table`` icons with custom ones from FontAwesome
  IconArrowAutofitContent: (props: any) => (
    <FontAwesomeIcon icon={faTextWidth} {...props} />
  ),

  IconArrowsSort: (props: any) => <FontAwesomeIcon icon={faSort} {...props} />,

  IconBoxMultiple: (props: any) => (
    <FontAwesomeIcon icon={faLayerGroup} {...props} />
  ),

  IconChevronDown: (props: any) => (
    <FontAwesomeIcon icon={faChevronDown} {...props} />
  ),

  IconChevronsDown: (props: any) => (
    <FontAwesomeIcon icon={faAnglesDown} {...props} />
  ),

  IconCircleX: (props: any) => (
    <FontAwesomeIcon icon={faCircleXmark} {...props} />
  ),

  IconClearAll: (props: any) => (
    <FontAwesomeIcon icon={faBarsStaggered} {...props} />
  ),

  IconColumns: (props: any) => <FontAwesomeIcon icon={faColumns} {...props} />,

  IconDeviceFloppy: (props: any) => (
    <FontAwesomeIcon icon={faFloppyDisk} {...props} />
  ),

  IconDots: (props: any) => <FontAwesomeIcon icon={faEllipsisH} {...props} />,

  IconDotsVertical: (props: any) => (
    <FontAwesomeIcon icon={faEllipsisV} {...props} />
  ),

  IconEdit: (props: any) => <FontAwesomeIcon icon={faEdit} {...props} />,

  IconEyeOff: (props: any) => <FontAwesomeIcon icon={faEyeSlash} {...props} />,

  IconFilter: (props: any) => <FontAwesomeIcon icon={faFilter} {...props} />,

  IconFilterOff: (props: any) => (
    <FontAwesomeIcon icon={faFilterCircleXmark} {...props} />
  ),

  IconGripHorizontal: (props: any) => (
    <FontAwesomeIcon icon={faGrip} {...props} />
  ),

  IconMaximize: (props: any) => <FontAwesomeIcon icon={faExpand} {...props} />,

  IconMinimize: (props: any) => (
    <FontAwesomeIcon icon={faCompress} {...props} />
  ),

  IconPinned: (props: any) => <FontAwesomeIcon icon={faThumbTack} {...props} />,

  IconPinnedOff: (props: any) => (
    <FontAwesomeIcon icon={faThumbTack} {...props} />
  ),

  IconSearch: (props: any) => <FontAwesomeIcon icon={faSearch} {...props} />,

  IconSearchOff: (props: any) => (
    <FontAwesomeIcon icon={faSearchMinus} {...props} />
  ),

  IconSortAscending: (props: any) => (
    <FontAwesomeIcon icon={faSortUp} {...props} />
  ),

  IconSortDescending: (props: any) => (
    <FontAwesomeIcon icon={faSortDown} {...props} />
  ),

  IconBaselineDensityLarge: (props: any) => (
    <FontAwesomeIcon icon={faBars} {...props} />
  ),

  IconBaselineDensityMedium: (props: any) => (
    <FontAwesomeIcon icon={faBars} {...props} />
  ),

  IconBaselineDensitySmall: (props: any) => (
    <FontAwesomeIcon icon={faBars} {...props} />
  ),

  IconX: (props: any) => <FontAwesomeIcon icon={faX} {...props} />,
};
