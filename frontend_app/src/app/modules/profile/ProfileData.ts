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

import {PageLink} from '../../../_monta/layout/core'

export const profileSubmenu: Array<PageLink> = [
  {
    title: 'Overview',
    path: '/crafted/pages/profile/overview',
    isActive: true,
  },
  {
    title: 'Separator',
    path: '/crafted/pages/profile/overview',
    isActive: true,
    isSeparator: true,
  },
  {
    title: 'Account',
    path: '/crafted/pages/profile/account',
    isActive: false,
  },
  {
    title: 'Account',
    path: '/crafted/pages/profile/account',
    isActive: false,
    isSeparator: true,
  },
  {
    title: 'Settings',
    path: '/crafted/pages/profile/settings',
    isActive: false,
  },
  {
    title: 'Settings',
    path: '/crafted/pages/profile/settings',
    isActive: false,
    isSeparator: true,
  },
]
