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

import React from "react";
import Table from "react-bootstrap/Table";
import MontaPagination from "@/components/partials/MontaPagination";

type TableProps = {
  limit: number,
  page: number,
  totalCount: number,
  isLoading: boolean,
  handleLimitChange: (event: React.ChangeEvent<HTMLSelectElement>) => void,
  handlePageChange: (selectedItem: { selected: number }) => void;
  PreLoader: JSX.Element,
  tableColumns: JSX.Element,
  tableRows: JSX.Element,
  tableName: string,

}

const MontaTable: React.FC<TableProps> = (
  {
    limit,
    page,
    totalCount,
    isLoading,
    handleLimitChange,
    handlePageChange,
    PreLoader,
    tableColumns,
    tableRows,
    tableName,
  }
) => {
  return (
    <div>
      {isLoading ? (
        PreLoader
      ) : (
        <>
          <div className="d-flex justify-content-between align-items-center">
            <h3 className="mb-0">{tableName} ({totalCount})</h3>
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
            <tr className="text-start text-gray-400 fw-bolder fs-7 text-uppercase gs-0">
              {tableColumns}
            </tr>
            </thead>
            <tbody className={"text-gray-600 fw-semibold"}>
            {tableRows}
            </tbody>
          </Table>
          <MontaPagination
            handlePageChange={handlePageChange}
            totalCount={totalCount}
            limit={limit}
            page={page} />
        </>
      )}
    </div>
  );
};

export default MontaTable;