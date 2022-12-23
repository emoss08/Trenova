import { useMemo } from 'react';

// material-ui
import { Chip, Table, TableBody, TableCell, TableHead, TableRow } from '@mui/material';

// third-party
import { useTable, useSortBy } from 'react-table';

// project import
import MainCard from 'components/MainCard';
import ScrollX from 'components/ScrollX';
import LinearWithLabel from 'components/@extended/Progress/LinearWithLabel';
import { HeaderSort } from 'components/third-party/ReactTable';
import makeData from 'data/react-table';

// ==============================|| REACT TABLE ||============================== //

function ReactTable({ columns, data, getHeaderProps }: any) {
  const { getTableProps, getTableBodyProps, headerGroups, rows, prepareRow } = useTable(
    {
      columns,
      data
    },
    useSortBy
  );

  const sortingRow = rows.slice(0, 9);

  return (
    <Table {...getTableProps()}>
      <TableHead>
        {headerGroups.map((headerGroup) => (
          <TableRow {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map((column: any) => (
              <TableCell {...column.getHeaderProps([{ className: column.className }, getHeaderProps(column)])}>
                <HeaderSort column={column} />
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableHead>
      <TableBody {...getTableBodyProps()}>
        {sortingRow.map((row, i) => {
          prepareRow(row);
          return (
            <TableRow {...row.getRowProps()}>
              {row.cells.map((cell: any) => (
                <TableCell {...cell.getCellProps([{ className: cell.column.className }])}>{cell.render('Cell')}</TableCell>
              ))}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

// ==============================|| REACT TABLE - SORTING ||============================== //

const SortingTable = () => {
  const data = useMemo(() => makeData(40), []);
  const columns = useMemo(
    () => [
      {
        Header: 'First Name',
        accessor: 'firstName'
      },
      {
        Header: 'Last Name',
        accessor: 'lastName'
      },
      {
        Header: 'Email',
        accessor: 'email'
      },
      {
        Header: 'Age',
        accessor: 'age',
        className: 'cell-right'
      },
      {
        Header: 'Role',
        accessor: 'role'
      },
      {
        Header: 'Visits',
        accessor: 'visits',
        className: 'cell-right'
      },
      {
        Header: 'Status',
        accessor: 'status',
        Cell: ({ value }: any) => {
          switch (value) {
            case 'Complicated':
              return <Chip color="error" label="Complicated" size="small" variant="light" />;
            case 'Relationship':
              return <Chip color="success" label="Relationship" size="small" variant="light" />;
            case 'Single':
            default:
              return <Chip color="info" label="Single" size="small" variant="light" />;
          }
        }
      },
      {
        Header: 'Profile Progress',
        accessor: 'progress',
        Cell: ({ value }: any) => <LinearWithLabel value={value} sx={{ minWidth: 75 }} />
      }
    ],
    []
  );

  return (
    <MainCard content={false}>
      <ScrollX>
        <ReactTable columns={columns} data={data} getHeaderProps={(column: any) => column.getSortByToggleProps()} />
      </ScrollX>
    </MainCard>
  );
};

export default SortingTable;
