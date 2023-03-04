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

import ContentLoader from 'react-content-loader'

export const AsideMenuContentLoader = () => (
  <ContentLoader viewBox='0 0 380 110' backgroundOpacity={0.1} foregroundOpacity={0.2}>
    {/* Only SVG shapes */}
    <rect x='0' y='0' rx='10' ry='10' width='100' height='100' />
    <rect x='120' y='17' rx='5' ry='5' width='250' height='30' />
    <rect x='120' y='60' rx='5' ry='5' width='150' height='20' />
  </ContentLoader>
)
