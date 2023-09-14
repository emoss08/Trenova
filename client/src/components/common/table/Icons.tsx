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
import {
  FontAwesomeIcon,
  FontAwesomeIconProps,
} from "@fortawesome/react-fontawesome";
import {
  faArrowDownShortWide,
  faArrowUpShortWide,
  faBars,
  faBarsSort,
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
  faTextWidth,
  faThumbTack,
  faX,
} from "@fortawesome/pro-duotone-svg-icons";
import React from "react";

type Props = Omit<FontAwesomeIconProps, "icon">;

export const montaTableIcons: Partial<MRT_Icons> = {
  // Override the default ``mantine-react-table`` icons with custom ones from FontAwesome
  IconArrowAutofitContent: (props: Props) => (
    <FontAwesomeIcon icon={faTextWidth} {...props} />
  ),

  IconArrowsSort: (props: Props) => (
    <FontAwesomeIcon icon={faBarsSort} size="sm" {...props} />
  ),

  IconBoxMultiple: (props: Props) => (
    <FontAwesomeIcon icon={faLayerGroup} size="sm" {...props} />
  ),

  IconChevronDown: (props: any) => (
    <FontAwesomeIcon icon={faChevronDown} {...props} />
  ),

  IconCircleX: (props: Props) => (
    <FontAwesomeIcon icon={faCircleXmark} {...props} />
  ),

  IconClearAll: (props: Props) => (
    <FontAwesomeIcon icon={faBarsStaggered} {...props} />
  ),

  IconColumns: (props: Props) => (
    <FontAwesomeIcon icon={faColumns} {...props} />
  ),

  IconDeviceFloppy: (props: Props) => (
    <FontAwesomeIcon icon={faFloppyDisk} {...props} />
  ),

  IconDots: (props: Props) => <FontAwesomeIcon icon={faEllipsisH} {...props} />,

  IconDotsVertical: (props: Props) => (
    <FontAwesomeIcon icon={faEllipsisV} {...props} />
  ),

  IconEdit: (props: Props) => <FontAwesomeIcon icon={faEdit} {...props} />,

  IconEyeOff: (props: Props) => (
    <FontAwesomeIcon icon={faEyeSlash} {...props} />
  ),

  IconFilter: (props: Props) => <FontAwesomeIcon icon={faFilter} {...props} />,

  IconFilterOff: (props: Props) => (
    <FontAwesomeIcon icon={faFilterCircleXmark} {...props} />
  ),

  IconGripHorizontal: (props: Props) => (
    <FontAwesomeIcon icon={faGrip} {...props} />
  ),

  IconMaximize: (props: Props) => (
    <FontAwesomeIcon icon={faExpand} {...props} />
  ),

  IconMinimize: (props: Props) => (
    <FontAwesomeIcon icon={faCompress} {...props} />
  ),

  IconPinned: (props: Props) => (
    <FontAwesomeIcon icon={faThumbTack} {...props} />
  ),

  IconPinnedOff: (props: Props) => (
    <FontAwesomeIcon icon={faThumbTack} {...props} />
  ),

  IconSearch: (props: Props) => <FontAwesomeIcon icon={faSearch} {...props} />,

  IconSearchOff: (props: Props) => (
    <FontAwesomeIcon icon={faSearchMinus} {...props} />
  ),

  IconSortAscending: (props: Props) => (
    <FontAwesomeIcon icon={faArrowUpShortWide} size="sm" {...props} />
  ),

  IconSortDescending: (props: Props) => (
    <FontAwesomeIcon icon={faArrowDownShortWide} size="sm" {...props} />
  ),

  IconBaselineDensityLarge: (props: Props) => (
    <FontAwesomeIcon icon={faBars} {...props} />
  ),

  IconBaselineDensityMedium: (props: Props) => (
    <FontAwesomeIcon icon={faBars} {...props} />
  ),

  IconBaselineDensitySmall: (props: Props) => (
    <FontAwesomeIcon icon={faBars} {...props} />
  ),

  IconX: (props: Props) => <FontAwesomeIcon icon={faX} {...props} />,
};
