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

export const ProfileDetailsContentLoader = () => (
  <ContentLoader
    viewBox='0 0 400 110'
    backgroundOpacity={0.1}
    foregroundOpacity={0.2}
    speed={2.0}
    backgroundColor='#ffffff'
    foregroundColor='#5c5c5c'
  >
    {/* Only SVG shapes */}
    <rect x='10' y='5' rx='3' ry='3' width='100' height='13' />
    <rect x='150' y='5' rx='3' ry='3' width='100' height='13' />
    <rect x='260' y='5' rx='3' ry='3' width='100' height='13' />
    <rect x='10' y='25' rx='3' ry='3' width='100' height='13' />
    <rect x='150' y='25' rx='3' ry='3' width='210' height='13' />
    <rect x='10' y='45' rx='3' ry='3' width='100' height='13' />
    <rect x='150' y='45' rx='3' ry='3' width='210' height='13' />
    <rect x='10' y='65' rx='3' ry='3' width='100' height='13' />
    <rect x='150' y='65' rx='3' ry='3' width='210' height='13' />
    <rect x='10' y='85' rx='3' ry='3' width='100' height='13' />
    <rect x='150' y='85' rx='3' ry='3' width='210' height='13' />
  </ContentLoader>
)
