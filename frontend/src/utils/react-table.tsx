import { useMemo, useState } from 'react';

// material-ui
import { MenuItem, OutlinedInput, Select, Slider, Stack, TextField, Tooltip } from '@mui/material';

// third-party
import { useAsyncDebounce, Row } from 'react-table';
import { matchSorter } from 'match-sorter';

// project import
import IconButton from 'components/@extended/IconButton';

// assets
import { CloseOutlined, LineOutlined, SearchOutlined } from '@ant-design/icons';

export function GlobalFilter({ preGlobalFilteredRows, globalFilter, setGlobalFilter, ...other }: any) {
  const count = preGlobalFilteredRows.length;
  const [value, setValue] = useState(globalFilter);
  const onChange = useAsyncDebounce((value) => {
    setGlobalFilter(value || undefined);
  }, 200);

  return (
    <OutlinedInput
      value={value || ''}
      onChange={(e) => {
        setValue(e.target.value);
        onChange(e.target.value);
      }}
      placeholder={`Search ${count} records...`}
      id="start-adornment-email"
      startAdornment={<SearchOutlined />}
      {...other}
    />
  );
}

export function DefaultColumnFilter({ column: { filterValue, Header, setFilter } }: any) {
  return (
    <TextField
      fullWidth
      value={filterValue || ''}
      onChange={(e) => {
        setFilter(e.target.value || undefined); // Set undefined to remove the filter entirely
      }}
      placeholder={Header}
      size="small"
    />
  );
}

export function SelectColumnFilter({ column: { filterValue, setFilter, preFilteredRows, id } }: any) {
  const options = useMemo(() => {
    const options = new Set();
    preFilteredRows.forEach((row: any) => {
      options.add(row.values[id]);
    });
    return [...options.values()];
  }, [id, preFilteredRows]);

  return (
    <Select
      value={filterValue}
      onChange={(e) => {
        setFilter(e.target.value || undefined);
      }}
      displayEmpty
      size="small"
    >
      <MenuItem value="">All</MenuItem>
      {options.map((option: any, i: number) => (
        <MenuItem key={i} value={option}>
          {option}
        </MenuItem>
      ))}
    </Select>
  );
}

export function SliderColumnFilter({ column: { filterValue, setFilter, preFilteredRows, id } }: any) {
  const [min, max] = useMemo(() => {
    let min = preFilteredRows.length ? preFilteredRows[0].values[id] : 0;
    let max = preFilteredRows.length ? preFilteredRows[0].values[id] : 0;
    preFilteredRows.forEach((row: any) => {
      min = Math.min(row.values[id], min);
      max = Math.max(row.values[id], max);
    });
    return [min, max];
  }, [id, preFilteredRows]);

  return (
    <Stack direction="row" alignItems="center" spacing={1} sx={{ pl: 1, minWidth: 120 }}>
      <Slider
        value={filterValue || min}
        min={min}
        max={max}
        step={1}
        onChange={(event: Event, newValue: number | number[]) => {
          setFilter(newValue);
        }}
        valueLabelDisplay="auto"
        aria-labelledby="non-linear-slider"
      />
      <Tooltip title="Reset">
        <IconButton size="small" color="error" onClick={() => setFilter(undefined)}>
          <CloseOutlined />
        </IconButton>
      </Tooltip>
    </Stack>
  );
}

export function NumberRangeColumnFilter({ column: { filterValue = [], preFilteredRows, setFilter, id } }: any) {
  const [min, max] = useMemo(() => {
    let min = preFilteredRows.length ? preFilteredRows[0].values[id] : 0;
    let max = preFilteredRows.length ? preFilteredRows[0].values[id] : 0;
    preFilteredRows.forEach((row: any) => {
      min = Math.min(row.values[id], min);
      max = Math.max(row.values[id], max);
    });
    return [min, max];
  }, [id, preFilteredRows]);

  return (
    <Stack direction="row" alignItems="center" spacing={1} sx={{ minWidth: 168, maxWidth: 250 }}>
      <TextField
        fullWidth
        value={filterValue[0] || ''}
        type="number"
        onChange={(e) => {
          const val = e.target.value;
          setFilter((old = []) => [val ? parseInt(val, 10) : undefined, old[1]]);
        }}
        placeholder={`Min (${min})`}
        size="small"
      />
      <LineOutlined />
      <TextField
        fullWidth
        value={filterValue[1] || ''}
        type="number"
        onChange={(e) => {
          const val = e.target.value;
          setFilter((old = []) => [old[0], val ? parseInt(val, 10) : undefined]);
        }}
        placeholder={`Max (${max})`}
        size="small"
      />
    </Stack>
  );
}

// @ts-ignore
function fuzzyTextFilterFn(rows, id, filterValue) {
  // @ts-ignore
  return matchSorter(rows, filterValue, { keys: [(row: any) => row.values[id]] });
}

fuzzyTextFilterFn.autoRemove = (val: any) => !val;

export const renderFilterTypes = () => ({
  fuzzyText: fuzzyTextFilterFn,
  text: (rows: Row[], id: string, filterValue: string) => {
    rows.filter((row: Row) => {
      const rowValue = row.values[id];
      return rowValue !== undefined ? String(rowValue).toLowerCase().startsWith(String(filterValue).toLowerCase()) : true;
    });
  }
});

// @ts-ignore
export function filterGreaterThan(rows, id, filterValue) {
  return rows.filter((row: any) => {
    const rowValue = row.values[id];
    return rowValue >= filterValue;
  });
}

filterGreaterThan.autoRemove = (val: any) => typeof val !== 'number';

export function useControlledState(state: any, { instance }: any) {
  return useMemo(() => {
    if (state.groupBy.length) {
      return {
        ...state,
        hiddenColumns: [...state.hiddenColumns, ...state.groupBy].filter((d, i, all) => all.indexOf(d) === i)
      };
    }
    return state;
  }, [state]);
}

export function roundedMedian(leafValues: any) {
  let min = leafValues[0] || 0;
  let max = leafValues[0] || 0;

  leafValues.forEach((value: number) => {
    min = Math.min(min, value);
    max = Math.max(max, value);
  });

  return Math.round((min + max) / 2);
}
