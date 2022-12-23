import { useMemo } from 'react';

// material-ui
import { Grid } from '@mui/material';

// project import
import BasicTable from 'sections/tables/react-table/BasicTable';
import FooterTable from 'sections/tables/react-table/FooterTable';
import makeData from 'data/react-table';

// ==============================|| REACT TABLE - BASIC ||============================== //

const Basic = () => {
  const data = useMemo(() => makeData(20), []);

  return (
    <Grid container spacing={3}>
      <Grid item xs={12} lg={6}>
        <BasicTable title="Basic Table" data={data.slice(0, 10)} />
      </Grid>
      <Grid item xs={12} lg={6}>
        <BasicTable title="Striped Table" data={data.slice(0, 10)} striped />
      </Grid>
      <Grid item xs={12}>
        <FooterTable data={data.slice(10, 19)} />
      </Grid>
    </Grid>
  );
};

export default Basic;
