// material-ui
import { Divider, Grid, Typography } from '@mui/material';

// ==============================|| PRODUCT DETAILS - SPECIFICATIONS ||============================== //

function ProductSpecifications() {
  return (
    <Grid container spacing={2.5}>
      <Grid item xs={12} md={6}>
        <Grid container spacing={2.5}>
          <Grid item xs={12}>
            <Typography variant="h5">Product Category</Typography>
          </Grid>
          <Grid item xs={12}>
            <Divider />
          </Grid>
          <Grid item xs={6}>
            <Typography color="textSecondary">Wearable Device Type:</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography>Smart Band</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography color="textSecondary">Compatible Devices:</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography>Smartphones</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography color="textSecondary">Ideal For:</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography>Unisex</Typography>
          </Grid>
        </Grid>
      </Grid>
      <Grid item xs={12} md={6}>
        <Grid container spacing={2.5}>
          <Grid item xs={12}>
            <Typography variant="h5">Manufacturer Details</Typography>
          </Grid>
          <Grid item xs={12}>
            <Divider />
          </Grid>
          <Grid item xs={6}>
            <Typography color="textSecondary">Brand:</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography>Apple</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography color="textSecondary">Model Series:</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography>Watch SE</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography color="textSecondary">Model Number:</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography>MYDT2HN/A</Typography>
          </Grid>
        </Grid>
      </Grid>
    </Grid>
  );
}

export default ProductSpecifications;
