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
import { formatDistanceToNow, isBefore, parseISO, subDays } from "date-fns";
import MontaPagination from "@/components/partials/MontaPagination";
import { UsersBodyContentLoader } from "@/components/admin/user-management/partials/UsersBodyContentLoader";
import Table from "react-bootstrap/Table";
import Link from "next/link";
import { Dropdown } from "react-bootstrap";
import Swal, { SweetAlertResult } from "sweetalert2";


const UsersCardBody = () => {
  const [users, setUsers] = useState<UserModel[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [limit, setLimit] = useState(20);
  const [page, setPage] = useState(0);
  const [totalCount, setTotalCount] = useState(0);


  const formatDate = (dateString: string) => {
    const date = parseISO(dateString);
    return formatDistanceToNow(date, { addSuffix: true });
  };

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

  const handlePageChange = (selectedItem: { selected: number }) => {
    setPage(selectedItem.selected);
  };

  const handleLimitChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    setLimit(Number(event.target.value));
    setPage(0); // Reset the page to 0 when the limit changes
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

  const handleActionSelect = (action: string, userId: string) => {
    console.log(`Selected action: ${action} for user with id ${userId}`);
    if (action === "edit") {
      console.log("Edit user");
    } else if (action === "delete") {
      const handleDelete = async (): Promise<void> => {
        try {
          await axios.delete(`http://localhost:8000/api/users/${userId}/`);
          Swal
            .fire("Deleted!", "User has been deleted.", "success")
            .then(
            () => fetchUsers()
          );
        } catch (error) {
          console.error(error);
          Swal.fire("Error", "There was an error deleting the user.", "error");
        }
      };

      Swal.fire({
        title: "Are you sure?",
        text: "You won't be able to revert this!",
        icon: "warning",
        showCancelButton: true,
        confirmButtonText: "Yes, delete it!",
        cancelButtonText: "No, cancel!",
        reverseButtons: true,
        customClass: {
          confirmButton: "btn btn-danger",
          cancelButton: "btn btn-outline-secondary ms-1"
        }
      }).then((result: SweetAlertResult) => {
        if (result.isConfirmed) {
          handleDelete();
        } else {
          Swal.fire("Cancelled", "Operation cancelled.", "info");
        }
      });
    }
  };

  return (
    <div>

      {isLoading ? (
        <UsersBodyContentLoader />
      ) : (
        <>
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
                <option value={totalCount}>All</option>
              </select>
            </div>
          </div>
          <Table className={"table align-middle table-row-dashed fs-6 gy-5 dataTable no-footer"} responsive>
            <thead>
            <tr className={"text-start text-muted fw-bold fs-7 text-uppercase gs-0"}>
              <th className={"min-w-125px sorting"}>User</th>
              <th className={"min-w-125px sorting"}>Role</th>
              <th className={"min-w-125px sorting"}>Last Login</th>
              <th className={"min-w-125px sorting"}>Date Joined</th>
              <th className={"text-end min-w-100px sorting_disabled"}>Actions</th>
            </tr>
            </thead>
            <tbody className={"text-gray-600 fw-semibold"}>
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
                    <Dropdown.Toggle variant="" className="btn btn-light btn-active-light-primary btn-sm"
                                     id={"users-action-dropdown"}>
                      Actions
                    </Dropdown.Toggle>

                    <Dropdown.Menu>
                      <Dropdown.Item eventKey="list"
                                     onClick={() => handleActionSelect("list", user.id)}>List</Dropdown.Item>
                      <Dropdown.Item eventKey="edit"
                                     onClick={() => handleActionSelect("edit", user.id)}>Edit</Dropdown.Item>
                      <Dropdown.Item eventKey="delete"
                                     onClick={() => handleActionSelect("delete", user.id)}>Delete</Dropdown.Item>
                    </Dropdown.Menu>
                  </Dropdown>
                </td>
              </tr>
            ))}
            </tbody>
          </Table>
          <MontaPagination handlePageChange={handlePageChange} totalCount={totalCount} limit={limit} page={page} />
        </>
      )}
    </div>
  );
};

export default UsersCardBody;
