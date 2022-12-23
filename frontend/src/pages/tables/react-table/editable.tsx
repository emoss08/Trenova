import { useEffect, useMemo, useState, ChangeEvent } from 'react';

// material-ui
import { Chip, Table, TableBody, TableCell, TableHead, TableRow, TextField } from '@mui/material';

// third-party
import { useTable, useFilters, Column } from 'react-table';

// project import
import MainCard from 'components/MainCard';
import ScrollX from 'components/ScrollX';
import LinearWithLabel from 'components/@extended/Progress/LinearWithLabel';
import makeData from 'data/react-table';

// ==============================|| REACT TABLE - EDITABLE CELL ||============================== //

const EditableCell = ({ value: initialValue, row: { index }, column: { id }, updateMyData }: any) => {
  const [value, setValue] = useState(initialValue);

  const onChange = (e: ChangeEvent<HTMLInputElement>) => {
    setValue(e.target?.value);
  };

  const onBlur = () => {
    updateMyData(index, id, value);
  };

  useEffect(() => {
    setValue(initialValue);
  }, [initialValue]);

  return (
    <TextField
      value={value}
      onChange={onChange}
      onBlur={onBlur}
      sx={{ '& .MuiOutlinedInput-input': { py: 0.75, px: 1 }, '& .MuiOutlinedInput-notchedOutline': { border: 'none' } }}
    />
  );
};

const defaultColumn = {
  Cell: EditableCell
};

// ==============================|| REACT TABLE ||============================== //

interface Props {
  columns: Column[];
  data: [];
  updateMyData: (rowIndex: number, columnId: any, value: any) => void;
  skipPageReset: boolean;
}

function ReactTable({ columns, data, updateMyData, skipPageReset }: Props) {
  const { getTableProps, getTableBodyProps, headerGroups, prepareRow, rows } = useTable(
    {
      columns,
      data,
      defaultColumn,
      // @ts-ignore
      autoResetPage: !skipPageReset,
      updateMyData
    },
    useFilters
  );

  return (
    <Table {...getTableProps()}>
      <TableHead>
        {headerGroups.map((headerGroup) => (
          <TableRow {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map((column: any) => (
              <TableCell {...column.getHeaderProps()}>{column.render('Header')}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableHead>
      <TableBody {...getTableBodyProps()}>
        {rows.map((row: any, i: number) => {
          prepareRow(row);
          return (
            <TableRow {...row.getRowProps()}>
              {row.cells.map((cell: any) => (
                <TableCell {...cell.getCellProps()}>{cell.render('Cell')}</TableCell>
              ))}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

// ==============================|| REACT TABLE - EDITABLE ||============================== //

const Editable = () => {
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
        accessor: 'age'
      },
      {
        Header: 'Visits',
        accessor: 'visits'
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

  const [data, setData] = useState(() => makeData(20));
  const [skipPageReset, setSkipPageReset] = useState(false);

  const updateMyData = (rowIndex: number, columnId: any, value: any) => {
    // We also turn on the flag to not reset the page
    setSkipPageReset(true);
    setData((old: []) =>
      old.map((row, index) => {
        if (index === rowIndex) {
          return {
            // @ts-ignore
            ...old[rowIndex],
            [columnId]: value
          };
        }
        return row;
      })
    );
  };

  useEffect(() => {
    setSkipPageReset(false);
  }, [data]);

  return (
    <MainCard content={false}>
      <ScrollX>
        <ReactTable columns={columns} data={data} updateMyData={updateMyData} skipPageReset={skipPageReset} />
      </ScrollX>
    </MainCard>
  );
};

export default Editable;
