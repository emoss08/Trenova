// material-ui
import { Grid, Typography } from '@mui/material';

// ==============================|| PRODUCT DETAILS - FEATURES ||============================== //

function ProductFeatures() {
  return (
    <Grid container spacing={2}>
      <Grid item xs={3}>
        <Typography color="textSecondary">Band :</Typography>
      </Grid>
      <Grid item xs={9}>
        <Typography>Smart Band</Typography>
      </Grid>
      <Grid item xs={3}>
        <Typography color="textSecondary" noWrap>
          Compatible Devices :
        </Typography>
      </Grid>
      <Grid item xs={9}>
        <Typography>Smartphones</Typography>
      </Grid>
      <Grid item xs={3}>
        <Typography color="textSecondary" noWrap>
          Ideal For :
        </Typography>
      </Grid>
      <Grid item xs={9}>
        <Typography>Unisex</Typography>
      </Grid>
      <Grid item xs={3}>
        <Typography color="textSecondary" noWrap>
          Lifestyle :
        </Typography>
      </Grid>
      <Grid item xs={9}>
        <Typography noWrap>Fitness | Indoor | Sports | Swimming | Outdoor</Typography>
      </Grid>
      <Grid item xs={3}>
        <Typography color="textSecondary" noWrap>
          Basic Features :
        </Typography>
      </Grid>
      <Grid item xs={9}>
        <Typography noWrap>Calendar | Date & Time | Timer/Stop Watch</Typography>
      </Grid>
      <Grid item xs={3}>
        <Typography color="textSecondary">Health Tracker :</Typography>
      </Grid>
      <Grid item xs={9}>
        <Typography> Heart Rate | Exercise Tracker</Typography>
      </Grid>
    </Grid>
  );
}

export default ProductFeatures;
