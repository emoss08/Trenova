import { Link as RouterLink } from 'react-router-dom';

// material-ui
import { CardContent, CardMedia, Grid, Link, Typography } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';

// assets
import Dashboard1 from 'assets/images/widget/dashborad-1.jpg';
import Dashboard2 from 'assets/images/widget/dashborad-3.jpg';

const mediaSX = {
  width: 90,
  height: 80,
  borderRadius: 1
};

// ===========================|| DATA WIDGET - LATEST POSTS ||=========================== //

const LatestPosts = () => (
  <MainCard
    title="Latest Posts"
    content={false}
    secondary={
      <Link component={RouterLink} to="#" color="primary">
        View all
      </Link>
    }
  >
    <CardContent>
      <Grid container spacing={3}>
        <Grid item xs={12}>
          <Grid container alignItems="center" spacing={2}>
            <Grid item>
              <CardMedia component="img" image={Dashboard1} title="image" sx={mediaSX} />
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={1}>
                <Grid item xs={12}>
                  <Typography align="left" variant="subtitle1">
                    Up unpacked friendly
                  </Typography>
                  <Typography align="left" variant="caption" color="secondary">
                    Video | 14 minutes ago
                  </Typography>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
        <Grid item xs={12}>
          <Grid container alignItems="center" spacing={2}>
            <Grid item>
              <CardMedia component="iframe" src="https://www.youtube.com/embed/668nUCeBHyY" title="image" sx={mediaSX} />
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={1}>
                <Grid item xs={12}>
                  <Typography align="left" variant="subtitle1">
                    Up unpacked friendly
                  </Typography>
                  <Typography align="left" variant="caption" color="secondary">
                    Video | 14 minutes ago
                  </Typography>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
        <Grid item xs={12}>
          <Grid container alignItems="center" spacing={2}>
            <Grid item>
              <CardMedia component="img" image={Dashboard2} title="image" sx={mediaSX} />
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={1}>
                <Grid item xs={12}>
                  <Typography align="left" variant="subtitle1">
                    Up unpacked friendly
                  </Typography>
                  <Typography align="left" variant="caption" color="secondary">
                    Video | 14 minutes ago
                  </Typography>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
      </Grid>
    </CardContent>
  </MainCard>
);

export default LatestPosts;
