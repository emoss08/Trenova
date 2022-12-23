import { useMemo } from 'react';

// material-ui
import { Chip, Table, TableBody, TableCell, TableFooter, TableHead, TableRow } from '@mui/material';

// third-party
import { useTable } from 'react-table';

// project import
import LinearWithLabel from 'components/@extended/Progress/LinearWithLabel';
import MainCard from 'components/MainCard';
import ScrollX from 'components/ScrollX';

// ==============================|| REACT TABLE ||============================== //

function ReactTable({ columns, data }: any) {
  const { getTableProps, getTableBodyProps, headerGroups, footerGroups, rows, prepareRow } = useTable({
    columns,
    data
  });

  return (
    <Table {...getTableProps()}>
      <TableHead>
        {headerGroups.map((headerGroup) => (
          <TableRow {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map((column: any) => (
              <TableCell {...column.getHeaderProps([{ className: column.className }])}>{column.render('Header')}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableHead>
      <TableBody {...getTableBodyProps()}>
        {rows.map((row, i) => {
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
      <TableFooter>
        {footerGroups.map((group) => (
          <TableRow {...group.getFooterGroupProps()}>
            {group.headers.map((column: any) => (
              <TableCell {...column.getFooterProps([{ className: column.className }])}>{column.render('Footer')}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableFooter>
    </Table>
  );
}

// ==============================|| REACT TABLE - BASIC FOOTER ||============================== //

const FooterTable = ({ data }: { data: any }) => {
  const columns = useMemo(
    () => [
      {
        Header: 'First Name',
        Footer: 'First Name',
        accessor: 'firstName'
      },
      {
        Header: 'Last Name',
        Footer: 'Last Name',
        accessor: 'lastName'
      },
      {
        Header: 'Email',
        Footer: 'Email',
        accessor: 'email'
      },
      {
        Header: 'Age',
        Footer: 'Age',
        accessor: 'age',
        className: 'cell-right'
      },
      {
        Header: 'Role',
        Footer: 'Role',
        accessor: 'role'
      },
      {
        Header: 'Visits',
        Footer: 'Visits',
        accessor: 'visits',
        className: 'cell-right'
      },
      {
        Header: 'Status',
        Footer: 'Status',
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
        Footer: 'Profile Progress',
        accessor: 'progress',
        Cell: ({ value }: any) => <LinearWithLabel value={value} sx={{ minWidth: 75 }} />
      }
    ],
    []
  );

  return (
    <MainCard title="Footer" content={false}>
      <ScrollX>
        <ReactTable columns={columns} data={data} />
      </ScrollX>
    </MainCard>
  );
};

export default FooterTable;
