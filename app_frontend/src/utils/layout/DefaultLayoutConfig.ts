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

import {ILayout} from '@/models/layout'

export const DefaultLayoutConfig: ILayout = {
  main: {
    type: 'default',
    primaryColor: '#009EF7',
    darkSkinEnabled: true,
  },
  loader: {
    display: true,
    type: 'default', // Set default|spinner-message|spinner-logo to hide or show page loader
  },
  scrolltop: {
    display: true,
  },
  header: {
    width: 'fluid', // Set fixed|fluid to change width type
    fixed: {
      desktop: false,
      tabletAndMobile: true, // Set true|false to set fixed Header for tablet and mobile modes
    },
  },
  megaMenu: {
    display: false, // Set true|false to show or hide Mega Menu
  },
  aside: {
    minimized: false,
    minimize: true,
  },
  content: {
    width: 'fixed', // Set fixed|fluid to change width
  },
  footer: {
    width: 'fluid', // Set fixed|fluid to change width type
  },
}
