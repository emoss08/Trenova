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

import ContentLoader from "react-content-loader";

export const UsersBodyContentLoader = () => {
  return (
    <ContentLoader viewBox="0 0 500 160"
                   backgroundColor={"#636363"}
                   foregroundColor={"#a19d9d"}
                   backgroundOpacity={0.1}
                   foregroundOpacity={0.3}>
      <circle cx="30" cy="20" r="15" />
      <rect x="50" y="10" rx="4" ry="4" width="375" height="8" />
      <rect x="50" y="20" rx="3" ry="3" width="425" height="10" />

      <circle cx="30" cy="60" r="15" />
      <rect x="50" y="50" rx="4" ry="4" width="375" height="8" />
      <rect x="50" y="60" rx="3" ry="3" width="425" height="10" />

      <circle cx="30" cy="100" r="15" />
      <rect x="50" y="90" rx="4" ry="4" width="375" height="8" />
      <rect x="50" y="100" rx="3" ry="3" width="425" height="10" />

      <circle cx="30" cy="140" r="15" />
      <rect x="50" y="130" rx="4" ry="4" width="375" height="8" />
      <rect x="50" y="140" rx="3" ry="3" width="425" height="10" />

    </ContentLoader>
  );
};