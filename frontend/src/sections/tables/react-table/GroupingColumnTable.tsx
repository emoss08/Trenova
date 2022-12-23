import { useMemo } from 'react';

// material-ui
import { Box, Chip, Stack, Table, TableBody, TableCell, TableHead, TableRow, Typography } from '@mui/material';

// third-party
import { useTable, useGroupBy, useExpanded, Column } from 'react-table';

// project-import
import MainCard from 'components/MainCard';
import ScrollX from 'components/ScrollX';
import LinearWithLabel from 'components/@extended/Progress/LinearWithLabel';
import { roundedMedian, useControlledState } from 'utils/react-table';

// assets
import { DownOutlined, GroupOutlined, RightOutlined, UngroupOutlined } from '@ant-design/icons';

// ==============================|| REACT TABLE ||============================== //

function ReactTable({ columns, data }: { columns: Column[]; data: [] }) {
  const { getTableProps, getTableBodyProps, headerGroups, rows, prepareRow } = useTable(
    {
      columns,
      data,
      // @ts-ignore
      initialState: { groupBy: ['status'] }
    },
    useGroupBy,
    useExpanded,
    (hooks) => {
      hooks.useControlledState.push(useControlledState);
      hooks.visibleColumns.push((columns, { instance }) => {
        // @ts-ignore
        if (!instance.state.groupBy.length) {
          return columns;
        }
        return [
          {
            id: 'expander',
            // @ts-ignore
            Header: ({ allColumns, state: { groupBy } }) =>
              groupBy.map((columnId: any) => {
                const column: any = allColumns.find((d) => d.id === columnId);
                const groupIcon = column.isGrouped ? <UngroupOutlined /> : <GroupOutlined />;

                return (
                  <Stack
                    direction="row"
                    spacing={1.25}
                    alignItems="center"
                    {...column.getHeaderProps()}
                    sx={{ display: 'inline-flex', '&:not(:last-of-type)': { mr: 1.5 } }}
                  >
                    {column.canGroupBy ? (
                      <Box
                        sx={{ color: column.isGrouped ? 'error.main' : 'primary.main', fontSize: '1rem' }}
                        {...column.getGroupByToggleProps()}
                      >
                        {groupIcon}
                      </Box>
                    ) : null}
                    <Typography variant="subtitle1">{column.render('Header')}</Typography>
                  </Stack>
                );
              }),
            Cell: ({ row }: { row: any }) => {
              if (row.canExpand) {
                const groupedCell = row.allCells.find((d: any) => d.isGrouped);
                const collapseIcon = row.isExpanded ? <DownOutlined /> : <RightOutlined />;

                return (
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Box
                      sx={{ pl: row.depth * 2, pr: 1.25, fontSize: '0.75rem', color: 'text.secondary' }}
                      {...row.getToggleRowExpandedProps()}
                    >
                      {collapseIcon}
                    </Box>
                    {groupedCell.render('Cell')} ({row.subRows.length})
                  </Stack>
                );
              }
              return null;
            }
          },
          ...columns
        ];
      });
    }
  );

  const firstPageRows = rows.slice(0, 15);

  return (
    <Table {...getTableProps()}>
      <TableHead>
        {headerGroups.map((headerGroup, i: number) => (
          <TableRow {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map((column: any, index: number) => {
              const groupIcon = column.isGrouped ? <UngroupOutlined /> : <GroupOutlined />;
              return (
                <TableCell key={`group-header-cell-${index}`} {...column.getHeaderProps([{ className: column.className }])}>
                  <Stack direction="row" spacing={1.15} alignItems="center" sx={{ display: 'inline-flex' }}>
                    {column.canGroupBy ? (
                      <Box
                        sx={{ color: column.isGrouped ? 'error.main' : 'primary.main', fontSize: '1rem' }}
                        {...column.getGroupByToggleProps()}
                      >
                        {groupIcon}
                      </Box>
                    ) : null}
                    <Box>{column.render('Header')}</Box>
                  </Stack>
                </TableCell>
              );
            })}
          </TableRow>
        ))}
      </TableHead>
      <TableBody {...getTableBodyProps()}>
        {firstPageRows.map((row: any) => {
          prepareRow(row);
          return (
            <TableRow {...row.getRowProps()}>
              {row.cells.map((cell: any) => {
                let bgcolor = 'background.paper';
                if (cell.isAggregated) bgcolor = 'warning.lighter';
                if (cell.isGrouped) bgcolor = 'success.lighter';
                if (cell.isPlaceholder) bgcolor = 'error.lighter';

                return (
                  <TableCell {...cell.getCellProps([{ className: cell.column.className }])} sx={{ bgcolor }}>
                    {/* eslint-disable-next-line */}
                    {cell.isAggregated ? cell.render('Aggregated') : cell.isPlaceholder ? null : cell.render('Cell')}
                  </TableCell>
                );
              })}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

// ==============================|| LEGEND ||============================== //

function Legend() {
  return (
    <Stack direction="row" spacing={1} alignItems="center" justifyContent="flex-end">
      <Chip color="warning" variant="light" label="Aggregated" />
    </Stack>
  );
}

// ==============================|| REACT TABLE - EXPANDING DETAILS ||============================== //

function GroupingColumnTable({ data }: { data: [] }) {
  const columns = useMemo(
    () => [
      {
        Header: 'First Name',
        accessor: 'firstName',
        aggregate: 'count',
        Aggregated: ({ value }: { value: number }) => `${value} Person`,
        disableGroupBy: true
      },
      {
        Header: 'Last Name',
        accessor: 'lastName',
        disableGroupBy: true
      },
      {
        Header: 'Email',
        accessor: 'email',
        disableGroupBy: true
      },
      {
        Header: 'Age',
        accessor: 'age',
        className: 'cell-right',
        aggregate: 'average',
        Aggregated: ({ value }: { value: number }) => `${value} (avg)`
      },
      {
        Header: 'Visits',
        accessor: 'visits',
        className: 'cell-right',
        aggregate: 'sum',
        Aggregated: ({ value }: { value: number }) => `${value} (total)`,
        disableGroupBy: true
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
        aggregate: roundedMedian,
        Aggregated: ({ value }: { value: number }) => `${value} (med)`,
        disableGroupBy: true,
        Cell: ({ value }: any) => <LinearWithLabel value={value} sx={{ minWidth: 75 }} />
      }
    ],
    []
  );

  return (
    <MainCard content={false} title="Grouping With Single Column" secondary={<Legend />}>
      <ScrollX>
        <ReactTable columns={columns} data={data} />
      </ScrollX>
    </MainCard>
  );
}

export default GroupingColumnTable;
