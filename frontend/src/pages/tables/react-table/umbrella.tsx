import { useEffect, useMemo } from 'react';

// material-ui
import { alpha, useTheme } from '@mui/material/styles';
import { Box, Chip, Stack, Table, TableBody, TableCell, TableFooter, TableHead, TableRow, Typography } from '@mui/material';

// third-party
import NumberFormat from 'react-number-format';
import update from 'immutability-helper';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import {
  useColumnOrder,
  useExpanded,
  useFilters,
  useGroupBy,
  useGlobalFilter,
  usePagination,
  useRowSelect,
  useSortBy,
  useTable,
  Column
} from 'react-table';

// project import
import MainCard from 'components/MainCard';
import Avatar from 'components/@extended/Avatar';
import ScrollX from 'components/ScrollX';
import LinearWithLabel from 'components/@extended/Progress/LinearWithLabel';
import makeData from 'data/react-table';
import SyntaxHighlight from 'utils/SyntaxHighlight';
import {
  DraggableHeader,
  DragPreview,
  HidingSelect,
  HeaderSort,
  IndeterminateCheckbox,
  TablePagination,
  TableRowSelection
} from 'components/third-party/ReactTable';
import {
  roundedMedian,
  renderFilterTypes,
  filterGreaterThan,
  GlobalFilter,
  DefaultColumnFilter,
  SelectColumnFilter,
  SliderColumnFilter,
  NumberRangeColumnFilter
} from 'utils/react-table';

// assets
import { DownOutlined, GroupOutlined, RightOutlined, UngroupOutlined } from '@ant-design/icons';

const avatarImage = require.context('assets/images/users', true);

// ==============================|| REACT TABLE ||============================== //

function ReactTable({ columns, data }: { columns: Column[]; data: [] }) {
  const theme = useTheme();
  const filterTypes = useMemo(() => renderFilterTypes, []);
  const defaultColumn = useMemo(() => ({ Filter: DefaultColumnFilter }), []);
  const initialState = useMemo(
    () => ({
      filters: [{ id: 'status', value: '' }],
      hiddenColumns: ['id', 'role', 'contact', 'country', 'fatherName'],
      columnOrder: ['selection', 'avatar', 'firstName', 'lastName', 'email', 'age', 'visits', 'status', 'progress'],
      pageIndex: 0,
      pageSize: 10
    }),
    []
  );

  const {
    getTableProps,
    getTableBodyProps,
    headerGroups,
    footerGroups,
    rows,
    // @ts-ignore
    page,
    prepareRow,
    // @ts-ignore
    setColumnOrder,
    // @ts-ignore
    gotoPage,
    // @ts-ignore
    setPageSize,
    setHiddenColumns,
    allColumns,
    visibleColumns,
    // @ts-ignore
    state: { globalFilter, hiddenColumns, pageIndex, pageSize, columnOrder, selectedRowIds },
    // @ts-ignore
    preGlobalFilteredRows,
    // @ts-ignore
    setGlobalFilter,
    // @ts-ignore
    selectedFlatRows
  } = useTable(
    {
      columns,
      data,
      // @ts-ignore
      defaultColumn,
      // @ts-ignore
      initialState,
      filterTypes
    },
    useGlobalFilter,
    useFilters,
    useColumnOrder,
    useGroupBy,
    useSortBy,
    useExpanded,
    usePagination,
    useRowSelect
  );

  const reorder = (item: any, newIndex: number) => {
    const { index: currentIndex } = item;

    const dragRecord = columnOrder[currentIndex];
    setColumnOrder(
      update(columnOrder, {
        $splice: [
          [currentIndex, 1],
          [newIndex, 0, dragRecord]
        ]
      })
    );
  };

  useEffect(() => {
    // @ts-ignore
    const newColumnOrder = visibleColumns.map((column) => column.id);
    setColumnOrder(newColumnOrder);

    // eslint-disable-next-line
  }, [hiddenColumns]);

  return (
    <>
      <TableRowSelection selected={Object.keys(selectedRowIds).length} />
      <Stack spacing={2}>
        <Stack direction="row" justifyContent="space-between" sx={{ p: 2, pb: 0 }}>
          <GlobalFilter
            preGlobalFilteredRows={preGlobalFilteredRows}
            globalFilter={globalFilter}
            setGlobalFilter={setGlobalFilter}
            size="small"
          />
          <HidingSelect hiddenColumns={hiddenColumns} setHiddenColumns={setHiddenColumns} allColumns={allColumns} />
        </Stack>

        <Box sx={{ width: '100%', overflowX: 'auto', display: 'block' }}>
          <Table {...getTableProps()}>
            <TableHead sx={{ borderTopWidth: 2 }}>
              {headerGroups.map((headerGroup) => (
                <TableRow {...headerGroup.getHeaderGroupProps()}>
                  {headerGroup.headers.map((column: any, index: number) => {
                    const groupIcon = column.isGrouped ? <UngroupOutlined /> : <GroupOutlined />;
                    return (
                      <TableCell key={`umbrella-header-cell-${index}`} {...column.getHeaderProps([{ className: column.className }])}>
                        <DraggableHeader reorder={reorder} key={column.id} column={column} index={index}>
                          <Stack direction="row" spacing={1.15} alignItems="center" sx={{ display: 'inline-flex' }}>
                            {column.canGroupBy ? (
                              <Box
                                sx={{ color: column.isGrouped ? 'error.main' : 'primary.main', fontSize: '1rem' }}
                                {...column.getGroupByToggleProps()}
                              >
                                {groupIcon}
                              </Box>
                            ) : null}
                            <HeaderSort column={column} sort />
                          </Stack>
                        </DraggableHeader>
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))}
            </TableHead>

            {/* striped table -> add class 'striped' */}
            <TableBody {...getTableBodyProps()} className="striped">
              {headerGroups.map((group: any) => (
                <TableRow {...group.getHeaderGroupProps()}>
                  {group.headers.map((column: any) => (
                    <TableCell {...column.getHeaderProps([{ className: column.className }])}>
                      {column.canFilter ? column.render('Filter') : null}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
              {page.map((row: any, i: number) => {
                prepareRow(row);
                return (
                  <TableRow
                    {...row.getRowProps()}
                    onClick={() => {
                      row.toggleRowSelected();
                    }}
                    sx={{ cursor: 'pointer', bgcolor: row.isSelected ? alpha(theme.palette.primary.lighter, 0.35) : 'inherit' }}
                  >
                    {row.cells.map((cell: any) => {
                      let bgcolor = 'inherit';
                      if (cell.isGrouped) bgcolor = 'success.lighter';
                      if (cell.isAggregated) bgcolor = 'warning.lighter';
                      if (cell.isPlaceholder) bgcolor = 'error.lighter';
                      if (cell.isPlaceholder) bgcolor = 'error.lighter';
                      if (row.isSelected) bgcolor = alpha(theme.palette.primary.lighter, 0.35);

                      const collapseIcon = row.isExpanded ? <DownOutlined /> : <RightOutlined />;

                      return (
                        <TableCell {...cell.getCellProps([{ className: cell.column.className }])} sx={{ bgcolor }}>
                          {/* eslint-disable-next-line */}
                          {cell.isGrouped ? (
                            <Stack direction="row" spacing={1} alignItems="center" sx={{ display: 'inline-flex' }}>
                              <Box
                                sx={{ pr: 1.25, fontSize: '0.75rem', color: 'text.secondary' }}
                                onClick={(e: any) => {
                                  row.toggleRowExpanded();
                                  e.stopPropagation();
                                }}
                              >
                                {collapseIcon}
                              </Box>
                              {cell.render('Cell')} ({row.subRows.length})
                            </Stack>
                          ) : // eslint-disable-next-line
                          cell.isAggregated ? (
                            cell.render('Aggregated')
                          ) : cell.isPlaceholder ? null : (
                            cell.render('Cell')
                          )}
                        </TableCell>
                      );
                    })}
                  </TableRow>
                );
              })}
            </TableBody>

            {/* footer table */}
            <TableFooter sx={{ borderBottomWidth: 2 }}>
              {footerGroups.map((group) => (
                <TableRow {...group.getFooterGroupProps()}>
                  {group.headers.map((column: any) => (
                    <TableCell {...column.getFooterProps([{ className: column.className }])}>{column.render('Footer')}</TableCell>
                  ))}
                </TableRow>
              ))}
            </TableFooter>
          </Table>
        </Box>
        <Box sx={{ p: 2, py: 0 }}>
          <TablePagination gotoPage={gotoPage} rows={rows} setPageSize={setPageSize} pageIndex={pageIndex} pageSize={pageSize} />
        </Box>

        <SyntaxHighlight>
          {JSON.stringify(
            {
              selectedRowIndices: selectedRowIds,
              'selectedFlatRows[].original': selectedFlatRows.map((d: any) => d.original)
            },
            null,
            2
          )}
        </SyntaxHighlight>
      </Stack>
    </>
  );
}

// ==============================|| REACT TABLE - UMBRELLA ||============================== //

const UmbrellaTable = () => {
  const data = useMemo(() => makeData(200), []);
  const columns = useMemo(
    () => [
      {
        title: 'Row Selection',
        Header: ({ getToggleAllPageRowsSelectedProps }: any) => (
          <IndeterminateCheckbox indeterminate {...getToggleAllPageRowsSelectedProps()} />
        ),
        Footer: '#',
        accessor: 'selection',
        Cell: ({ row }: any) => <IndeterminateCheckbox {...row.getToggleRowSelectedProps()} />,
        disableSortBy: true,
        disableFilters: true,
        disableGroupBy: true,
        Aggregated: () => null
      },
      {
        Header: '#',
        Footer: '#',
        accessor: 'id',
        className: 'cell-center',
        disableFilters: true,
        disableGroupBy: true
      },
      {
        Header: 'Avatar',
        Footer: 'Avatar',
        accessor: 'avatar',
        className: 'cell-center',
        disableFilters: true,
        disableGroupBy: true,
        Cell: ({ value }: any) => <Avatar alt="Avatar 1" size="sm" src={avatarImage(`./avatar-${!value ? 1 : value}.png`)} />
      },
      {
        Header: 'First Name',
        Footer: 'First Name',
        accessor: 'firstName',
        disableGroupBy: true,
        aggregate: 'count',
        Aggregated: ({ value }: { value: number }) => `${value} Person`
      },
      {
        Header: 'Last Name',
        Footer: 'Last Name',
        accessor: 'lastName',
        filter: 'fuzzyText',
        disableGroupBy: true
      },
      {
        Header: 'Father Name',
        Footer: 'Father Name',
        accessor: 'fatherName',
        disableGroupBy: true
      },
      {
        Header: 'Email',
        Footer: 'Email',
        accessor: 'email',
        disableGroupBy: true
      },
      {
        Header: 'Age',
        Footer: 'Age',
        accessor: 'age',
        className: 'cell-right',
        Filter: SliderColumnFilter,
        filter: 'equals',
        aggregate: 'average',
        Aggregated: ({ value }: { value: number }) => `${Math.round(value * 100) / 100} (avg)`
      },
      {
        Header: 'Role',
        Footer: 'Role',
        accessor: 'role',
        disableGroupBy: true
      },
      {
        Header: 'Contact',
        Footer: 'Contact',
        accessor: 'contact',
        disableGroupBy: true
      },
      {
        Header: 'Country',
        Footer: 'Country',
        accessor: 'country',
        disableGroupBy: true
      },
      {
        Header: 'Visits',
        accessor: 'visits',
        className: 'cell-right',
        Filter: NumberRangeColumnFilter,
        filter: 'between',
        disableGroupBy: true,
        aggregate: 'sum',
        Aggregated: ({ value }: { value: number }) => `${value} (total)`,
        Footer: (info: any) => {
          const { rows } = info;
          // only calculate total visits if rows change
          const total = useMemo(() => rows.reduce((sum: number, row: any) => row.values.visits + sum, 0), [rows]);

          return (
            <Typography variant="subtitle1">
              <NumberFormat value={total} displayType="text" thousandSeparator />
            </Typography>
          );
        }
      },
      {
        Header: 'Status',
        Footer: 'Status',
        accessor: 'status',
        Filter: SelectColumnFilter,
        filter: 'includes',
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
        Filter: SliderColumnFilter,
        filter: filterGreaterThan,
        disableGroupBy: true,
        aggregate: roundedMedian,
        Aggregated: ({ value }: { value: number }) => `${value} (med)`,
        Cell: ({ value }: any) => <LinearWithLabel value={value} sx={{ minWidth: 140 }} />
      }
    ],
    []
  );

  return (
    <MainCard
      title="Umbrella Table"
      subheader="This page consist combination of most possible features of react-table in to one table. Sorting, grouping, row selection, hidden row, filter, search, pagination, footer row available in below table."
      content={false}
    >
      <ScrollX>
        <DndProvider backend={HTML5Backend}>
          <ReactTable columns={columns} data={data} />
          <DragPreview />
        </DndProvider>
      </ScrollX>
    </MainCard>
  );
};

export default UmbrellaTable;
