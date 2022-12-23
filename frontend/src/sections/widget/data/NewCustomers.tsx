// material-ui
import { Avatar, CardContent, Grid, Stack, Typography } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';
import SimpleBar from 'components/third-party/SimpleBar';
import Dot from 'components/@extended/Dot';

// assets
import { ClockCircleOutlined } from '@ant-design/icons';

import Avatar1 from 'assets/images/users/avatar-1.png';
import Avatar2 from 'assets/images/users/avatar-2.png';
import Avatar3 from 'assets/images/users/avatar-3.png';
import Avatar4 from 'assets/images/users/avatar-4.png';
import Avatar5 from 'assets/images/users/avatar-5.png';

const iconSX = {
  fontSize: '0.675rem'
};

// ===========================|| DATA WIDGET - NEW CUSTOMERS CARD ||=========================== //

const NewCustomers = () => (
  <MainCard title="New Customers" content={false}>
    <SimpleBar sx={{ height: 334 }}>
      <CardContent>
        <Grid container spacing={3} alignItems="center">
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar1} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  Alex Thompson
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      Cheers!
                    </Typography>
                  </Grid>
                  <Grid item sx={{ display: 'flex' }}>
                    <Dot color="success" size={10} />
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar2} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  John Doue
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      stay hungry stay foolish!
                    </Typography>
                  </Grid>
                  <Grid item sx={{ display: 'flex' }}>
                    <Dot color="success" size={10} />
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar3} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  Alex Thompson
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      Cheers!
                    </Typography>
                  </Grid>
                  <Grid item>
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <Typography variant="caption" color="secondary">
                        30 min ago
                      </Typography>
                      <ClockCircleOutlined style={iconSX} />
                    </Stack>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar4} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  John Doue
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      stay hungry stay foolish!
                    </Typography>
                  </Grid>
                  <Grid item>
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <Typography variant="caption" color="secondary">
                        10 min ago
                      </Typography>
                      <ClockCircleOutlined style={iconSX} />
                    </Stack>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar5} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  Shirley Hoe
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      Cheers!
                    </Typography>
                  </Grid>
                  <Grid item>
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <Typography variant="caption" color="secondary">
                        30 min ago
                      </Typography>
                      <ClockCircleOutlined style={iconSX} />
                    </Stack>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar1} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  Alex Thompson
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      Cheers!
                    </Typography>
                  </Grid>
                  <Grid item sx={{ display: 'flex' }}>
                    <Dot color="success" size={10} />
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar2} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  John Doue
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      stay hungry stay foolish!
                    </Typography>
                  </Grid>
                  <Grid item sx={{ display: 'flex' }}>
                    <Dot color="success" size={10} />
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar3} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  Alex Thompson
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      Cheers!
                    </Typography>
                  </Grid>
                  <Grid item>
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <Typography variant="caption" color="secondary">
                        10 min ago
                      </Typography>
                      <ClockCircleOutlined style={iconSX} />
                    </Stack>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar4} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  John Doue
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      stay hungry stay foolish!
                    </Typography>
                  </Grid>
                  <Grid item>
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <Typography variant="caption" color="secondary">
                        10 min ago
                      </Typography>
                      <ClockCircleOutlined style={iconSX} />
                    </Stack>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={2}>
              <Grid item>
                <Avatar alt="coverimage" src={Avatar5} />
              </Grid>
              <Grid item xs zeroMinWidth>
                <Typography align="left" variant="subtitle1">
                  Shirley Hoe
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs zeroMinWidth>
                    <Typography align="left" variant="caption" color="secondary">
                      Cheers!
                    </Typography>
                  </Grid>
                  <Grid item>
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <Typography variant="caption" color="secondary">
                        30 min ago
                      </Typography>
                      <ClockCircleOutlined style={iconSX} />
                    </Stack>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
      </CardContent>
    </SimpleBar>
  </MainCard>
);

export default NewCustomers;
