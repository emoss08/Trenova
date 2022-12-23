import { useMemo } from 'react';

// material-ui
import { Grid } from '@mui/material';

// project import
import RowDragDrop from 'sections/tables/react-table/RowDragDrop';
import ColumnDragDrop from 'sections/tables/react-table/ColumnDragDrop';
import makeData from 'data/react-table';

// ==============================|| REACT TABLE - DRAG & DROP ||============================== //

const DragDrop = () => {
  const data = useMemo(() => makeData(20), []);

  return (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <RowDragDrop data={data.slice(0, 10)} />
      </Grid>
      <Grid item xs={12}>
        <ColumnDragDrop data={data.slice(10, 19)} />
      </Grid>
    </Grid>
  );
};

export default DragDrop;
