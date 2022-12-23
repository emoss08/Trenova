import { Link as RouterLink } from 'react-router-dom';

// material-ui
import { CardContent, Grid, Link, Typography } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';
import Avatar from 'components/@extended/Avatar';

// assets
import { TwitterCircleFilled, ClockCircleFilled, BugFilled, MobileFilled, WarningFilled } from '@ant-design/icons';

// ==============================|| DATA WIDGET - TASKS CARD ||============================== //

const TasksCard = () => (
  <MainCard
    title="Tasks"
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
        spacing={2.75}
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
            top: 10,
            left: 38,
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
              <Avatar type="filled" color="success" size="sm" sx={{ top: 10 }}>
                <TwitterCircleFilled />
              </Avatar>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={0}>
                <Grid item xs={12}>
                  <Typography align="left" variant="caption" color="secondary">
                    8:50
                  </Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography align="left" variant="body2">
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
              <Avatar type="filled" color="primary" size="sm" sx={{ top: 10 }}>
                <ClockCircleFilled />
              </Avatar>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={0}>
                <Grid item xs={12}>
                  <Typography align="left" variant="caption" color="secondary">
                    Sat, 5 Mar
                  </Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography align="left" variant="body2">
                    Design mobile Application
                  </Typography>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
        <Grid item xs={12}>
          <Grid container spacing={2}>
            <Grid item>
              <Avatar type="filled" color="error" size="sm" sx={{ top: 10 }}>
                <BugFilled />
              </Avatar>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={0}>
                <Grid item xs={12}>
                  <Typography align="left" variant="caption" color="secondary">
                    Sun, 17 Feb
                  </Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography align="left" variant="body2">
                    <Link component={RouterLink} to="#" underline="hover">
                      Jenny
                    </Link>{' '}
                    assign you a task{' '}
                    <Link component={RouterLink} to="#" underline="hover">
                      Mockup Design
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
              <Avatar type="filled" color="warning" size="sm" sx={{ top: 10 }}>
                <WarningFilled />
              </Avatar>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={0}>
                <Grid item xs={12}>
                  <Typography align="left" variant="caption" color="secondary">
                    Sat, 18 Mar
                  </Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography align="left" variant="body2">
                    Design logo
                  </Typography>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
        <Grid item xs={12}>
          <Grid container spacing={2}>
            <Grid item>
              <Avatar type="filled" color="success" size="sm" sx={{ top: 10 }}>
                <MobileFilled />
              </Avatar>
            </Grid>
            <Grid item xs zeroMinWidth>
              <Grid container spacing={0}>
                <Grid item xs={12}>
                  <Typography align="left" variant="caption" color="secondary">
                    Sat, 22 Mar
                  </Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography align="left" variant="body2">
                    Design mobile Application
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

export default TasksCard;
