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

import React from 'react';
import ReactPaginate from 'react-paginate';

type Props = {
  handlePageChange: (selectedItem: { selected: number }) => void;
  totalCount: number;
  limit: number;
  page: number;
};

const MontaPagination: React.FC<Props> = ({ handlePageChange, totalCount, limit, page }) => {
  return (
    <div className="row">
      <div className="col-sm-12 col-md-5 d-flex align-items-center justify-content-center justify-content-md-start"></div>
      <div className="col-sm-12 col-md-7 d-flex align-items-center justify-content-center justify-content-md-end">
        <div className={'dataTables_paginate paging_simple_numbers'}>
          <ReactPaginate
            pageCount={Math.ceil(totalCount / limit)}
            marginPagesDisplayed={2}
            pageRangeDisplayed={5}
            onPageChange={handlePageChange}
            containerClassName="pagination"
            activeClassName="active"
            pageClassName="paginate_button page-item"
            pageLinkClassName="page-link"
            previousClassName="paginate_button page-item previous"
            previousLabel={<i className="previous"></i>}
            previousLinkClassName="page-link"
            nextClassName="page-item"
            nextLinkClassName="page-link"
            nextLabel={<i className="next"></i>}
            forcePage={page} // set the active page based on the current page state
          />
        </div>
      </div>
    </div>
  );
};

export default MontaPagination;
