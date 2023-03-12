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

import React, { useCallback, useEffect, useState } from "react";
import { UserModel } from "@/models/user";
import axios from "axios";
import { USER_URL } from "@/utils/_links";
import { isBefore, parseISO, subDays } from "date-fns";
import { UsersBodyContentLoader } from "@/components/admin/user-management/partials/UsersBodyContentLoader";
import Link from "next/link";
import { Dropdown } from "react-bootstrap";
import { SweetAlertResult } from "sweetalert2";
import { swalBs } from "@/components/partials/SwalBs";
import { formatDate } from "@/utils/helpers/DateHelpers";
import MontaTable from "@/components/partials/MontaTable";

const UsersCardBody = () => {
  const [users, setUsers] = useState<UserModel[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [limit, setLimit] = useState(20);
  const [page, setPage] = useState(0);
  const [totalCount, setTotalCount] = useState(0);

  const handlePageChange = (selectedItem: { selected: number }) => {
    setPage(selectedItem.selected);
  };

  const handleLimitChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    setLimit(Number(event.target.value));
    setPage(0); // Reset the page to 0 when the limit changes
  };

  const handleActionSelect = async (action: string, user: UserModel) => {
    console.log(`Selected action: ${action} for user with id ${user.id}`);
    if (action === "edit") {
      console.log("Edit user");
    } else if (action === "delete") {
      const handleDelete = async (): Promise<void> => {
        try {
          await axios.delete(`http://localhost:8000/api/users/${user.id}/`);
          swalBs
            .fire("Deleted!", "User has been deleted.", "success")
            .then(
              () => fetchUsers()
            );
        } catch (error) {
          console.error(error);
          await swalBs.fire("Error", "There was an error deleting the user.", "error");
        }
      };

      swalBs.fire({
        title: "Are you sure?",
        text: "You won't be able to revert this!",
        icon: "warning",
        showCancelButton: true,
        confirmButtonText: "Yes, delete it!",
        cancelButtonText: "No, cancel!",
        reverseButtons: true
      }).then((result: SweetAlertResult) => {
        if (result.isConfirmed) {
          handleDelete();
        } else {
          swalBs.fire("Cancelled", "Operation cancelled.", "info");
        }
      });
    }
  };

  const fetchUsers = useCallback(() => {
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

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const lastLoginClassName = (lastLogin: string | null) => {
    if (!lastLogin) {
      return "badge badge-light-danger fw-bold";
    }

    const lastLoginDate = parseISO(lastLogin);

    if (isBefore(lastLoginDate, subDays(new Date(), 7))) {
      return "badge badge-light-warning fw-bold";
    }

    return "badge badge-light-success fw-bold";
  };

  const TableColumns = () => {
    return (
      <>
        <th className={"min-w-125px sorting"}>User</th>
        <th className={"min-w-125px sorting"}>Role</th>
        <th className={"min-w-125px sorting"}>Last Login</th>
        <th className={"min-w-125px sorting"}>Date Joined</th>
        <th className={"text-end min-w-100px sorting_disabled"}>Actions</th>
      </>
    );
  };

  const TableRows = () => {
    return (
      <>
        {users.map((user) => (
          <tr key={user.id}>
            <td className="d-flex align-items-center">
              <div className="symbol symbol-circle symbol-50px overflow-hidden me-3">
                <Link href="#">
                  {/* Pick between random classes */}
                  <div
                    className={`symbol-label fs-3 ${Math.random() < 0.25 ? "bg-light-warning text-warning" : Math.random() < 0.5 ? "bg-light-primary text-primary" : Math.random() < 0.75 ? "bg-light-success text-success" : "bg-light-danger text-danger"}`}>
                    {user.profile?.first_name.charAt(0)}
                  </div>
                </Link>
              </div>
              <div className="d-flex flex-column">
                    <span
                      className="text-gray-800 text-hover-primary mb-1">{user.profile?.first_name} {user.profile?.last_name}</span>
                <span>{user?.email}</span>
              </div>
            </td>
            <td>{user.profile?.title_name}</td>
            <td>
              <div className={lastLoginClassName(user.last_login)}>
                {user.last_login ? formatDate(user.last_login) : "Never"}
              </div>
            </td>

            <td>{formatDate(user.date_joined)}</td>
            <td className={"text-end"}>
              <Dropdown>
                <Dropdown.Toggle
                  variant=""
                  className="btn btn-light btn-active-light-primary btn-sm"
                  id={"users-action-dropdown"}>
                  Actions
                </Dropdown.Toggle>
                <Dropdown.Menu>
                  <Dropdown.Item eventKey="list"
                                 onClick={() => handleActionSelect("list", user)}>
                    List
                  </Dropdown.Item>
                  <Dropdown.Item eventKey="edit"
                                 onClick={() => handleActionSelect("edit", user)}>
                    Edit
                  </Dropdown.Item>
                  <Dropdown.Item eventKey="delete"
                                 onClick={() => handleActionSelect("delete", user)}>
                    Delete
                  </Dropdown.Item>
                </Dropdown.Menu>
              </Dropdown>
            </td>
          </tr>
        ))}

      </>
    );
  };

  return (
    <div>
      <MontaTable
        limit={limit}
        page={page}
        totalCount={totalCount}
        isLoading={isLoading}
        handleLimitChange={handleLimitChange}
        handlePageChange={handlePageChange}
        PreLoader={<UsersBodyContentLoader />}
        tableColumns={<TableColumns />}
        tableRows={<TableRows />}
        tableName={"Users"} />
    </div>
  );
};

export default UsersCardBody;
