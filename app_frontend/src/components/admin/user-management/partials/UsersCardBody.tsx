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
import React, { useEffect, useState } from "react";
import { UserModel } from "@/models/user";
import axios from "axios";
import { USER_URL } from "@/utils/_links";
import { formatDistanceToNow, parseISO } from "date-fns";
import MontaPagination from "@/components/partials/MontaPagination";

const UsersCardBody = () => {
  const [users, setUsers] = useState<UserModel[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [limit, setLimit] = useState(20);
  const [page, setPage] = useState(0);
  const [totalCount, setTotalCount] = useState(0);

  useEffect(() => {
    setIsLoading(true);

    axios
      .get(USER_URL, {
        params: {
          limit,
          offset: page * limit
        }
      })
      .then((response) => {
        setUsers(response.data.results);
        setTotalCount(response.data.count);
        setIsLoading(false);
      })
      .catch((error) => {
        console.error(error);
        setIsLoading(false);
      });
  }, [limit, page]);

  const formatDate = (dateString: string) => {
    const date = parseISO(dateString);
    return formatDistanceToNow(date, { addSuffix: true });
  };

  const handlePageChange = (selectedItem: { selected: number }) => {
    setPage(selectedItem.selected);
  };

  const handleLimitChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    setLimit(Number(event.target.value));
    setPage(0); // Reset the page to 0 when the limit changes
  };

  return (
    <div>
      <div className="d-flex justify-content-between align-items-center">
        <h3 className="mb-0">Users ({totalCount})</h3>
        <div className="form-group mb-0">
          <label htmlFor="limitSelect">Limit:</label>
          <select
            id="limitSelect"
            className="form-select form-select-solid fw-bold"
            value={limit}
            onChange={handleLimitChange}
          >
            <option value={20}>20</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </select>
        </div>
      </div>
      {isLoading ? (
        <div>Loading...</div>
      ) : (
        <>
          <table className="table">
            <thead>
            <tr>
              <th>Full Name</th>
              <th>Username</th>
              <th>Email</th>
              <th>Date Joined</th>
            </tr>
            </thead>
            <tbody>
            {users.map((user) => (
              <tr key={user.id}>
                <td>
                  {user.profile?.first_name} {user.profile?.last_name}
                </td>
                <td>{user.username}</td>
                <td>{user.email}</td>
                <td>{formatDate(user.date_joined)}</td>
              </tr>
            ))}
            </tbody>
          </table>
          <MontaPagination handlePageChange={handlePageChange} totalCount={totalCount} limit={limit} page={page} />
        </>
      )}
    </div>
  );
};

export default UsersCardBody;
