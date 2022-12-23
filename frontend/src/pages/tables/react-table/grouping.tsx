import { useMemo } from 'react';

// material-ui
import { Grid } from '@mui/material';

// project import
import makeData from 'data/react-table';
import GroupingTable from 'sections/tables/react-table/GroupingTable';
import GroupingColumnTable from 'sections/tables/react-table/GroupingColumnTable';

// ==============================|| REACT TABLE - GROUPING ||============================== //

const Grouping = () => {
  const data = useMemo(() => makeData(1000), []);

  return (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <GroupingTable data={data} />
      </Grid>
      <Grid item xs={12}>
        <GroupingColumnTable data={data} />
      </Grid>
    </Grid>
  );
};

export default Grouping;
