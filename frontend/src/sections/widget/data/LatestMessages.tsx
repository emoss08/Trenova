import { Link as RouterLink } from 'react-router-dom';

// material-ui
import { CardContent, Grid, Link, Typography } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';
import Avatar from 'components/@extended/Avatar';

// assets
import { TwitterOutlined, ShoppingOutlined, CheckOutlined, UserOutlined } from '@ant-design/icons';

// ==========================|| DATA WIDGET - LATEST MESSAGES ||========================== //

const LatestMessages = () => (
  <MainCard
    title="Latest Messages"
    content={false}
    secondary={
      <Link component={RouterLink} to="#" color="primary">
        View all
      </Link>
    }
  >
    <CardContent>
      <Grid
        container
        spacing={3}
        alignItems="center"
        sx={{
          position: 'relative',
          '&>*': {
            position: 'relative',
            zIndex: '5'
          },
          '&:after': {
            content: '""',
            position: 'absolute',
            top: 8,
            left: 110,
            width: 2,
            height: '100%',
            background: '#ebebeb',
            zIndex: '1'
          }
        }}
      >
        <Grid item xs={12}>
          <Grid container spacing={2}>
            <Grid item>
              <Grid container spacing={2} alignItems="center">
                <Grid item xs zeroMinWidth>
                  <Typography align="left" variant="caption" color="secondary">
                    2 hrs ago
                  </Typography>
                </Grid>
                <Grid item>
                  <Avatar color="info">
                    <TwitterOutlined />
                  </Avatar>
                </Grid>
              </Grid>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={1}>
                <Grid item xs={12}>
                  <Typography component="div" align="left" variant="subtitle1">
                    + 1652 Followers
                  </Typography>
                  <Typography color="secondary" align="left" variant="caption">
                    You’re getting more and more followers, keep it up!
                  </Typography>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
        <Grid item xs={12}>
          <Grid container spacing={2}>
            <Grid item>
              <Grid container spacing={2} alignItems="center">
                <Grid item xs zeroMinWidth>
                  <Typography align="left" variant="caption" color="secondary">
                    4 hrs ago
                  </Typography>
                </Grid>
                <Grid item>
                  <Avatar color="error">
                    <ShoppingOutlined />
                  </Avatar>
                </Grid>
              </Grid>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={1}>
                <Grid item xs={12}>
                  <Typography component="div" align="left" variant="subtitle1">
                    + 5 New Products were added!
                  </Typography>
                  <Typography color="secondary" align="left" variant="caption">
                    Congratulations!
                  </Typography>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
        <Grid item xs={12}>
          <Grid container spacing={2}>
            <Grid item>
              <Grid container spacing={2} alignItems="center">
                <Grid item xs zeroMinWidth>
                  <Typography align="left" variant="caption" color="secondary">
                    1 day ago
                  </Typography>
                </Grid>
                <Grid item>
                  <Avatar color="success">
                    <CheckOutlined />
                  </Avatar>
                </Grid>
              </Grid>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={1}>
                <Grid item xs={12}>
                  <Typography component="div" align="left" variant="subtitle1">
                    Database backup completed!
                  </Typography>
                  <Typography color="secondary" align="left" variant="caption">
                    Download the{' '}
                    <Link component={RouterLink} to="#" underline="hover">
                      latest backup
                    </Link>
                    .
                  </Typography>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
        <Grid item xs={12}>
          <Grid container spacing={2}>
            <Grid item>
              <Grid container spacing={2} alignItems="center">
                <Grid item xs zeroMinWidth>
                  <Typography align="left" variant="caption" color="secondary">
                    2 day ago
                  </Typography>
                </Grid>
                <Grid item>
                  <Avatar color="primary">
                    <UserOutlined />
                  </Avatar>
                </Grid>
              </Grid>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={1}>
                <Grid item xs={12}>
                  <Typography component="div" align="left" variant="subtitle1">
                    +2 Friend Requests
                  </Typography>
                  <Typography color="secondary" align="left" variant="caption">
                    This is great, keep it up!
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

export default LatestMessages;
