// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, Button, Grid, Stack, TextField, Typography, useMediaQuery } from '@mui/material';

// third party
import { useTimer } from 'react-timer-hook';

// project import
import MainCard from 'components/MainCard';

// assets
import coming from 'assets/images/maintenance/coming-soon.png';

// ==============================|| COMING SOON - MAIN ||============================== //

const TimerBox = ({ count, label }: { count: number; label: string }) => {
  const theme = useTheme();
  const matchDownSM = useMediaQuery(theme.breakpoints.down('sm'));

  return (
    <MainCard content={false} sx={{ width: { xs: 60, sm: 80 } }}>
      <Stack justifyContent="center" alignItems="center">
        <Box sx={{ py: 1.75 }}>
          <Typography variant={matchDownSM ? 'h4' : 'h2'}>{count}</Typography>
        </Box>
        <Box sx={{ p: 0.5, bgcolor: 'secondary.lighter', width: '100%' }}>
          <Typography align="center" variant="subtitle2">
            {label}
          </Typography>
        </Box>
      </Stack>
    </MainCard>
  );
};

function ComingSoon() {
  const time = new Date();
  time.setSeconds(time.getSeconds() + 3600 * 24 * 2 - 3600 * 15.5);

  const { seconds, minutes, hours, days } = useTimer({ expiryTimestamp: time });

  return (
    <>
      <Grid container spacing={4} direction="column" alignItems="center" justifyContent="center" sx={{ minHeight: '100vh', py: 2 }}>
        <Grid item xs={12}>
          <Box sx={{ height: { xs: 310, sm: 420 }, width: { xs: 360, sm: 'auto' } }}>
            <img src={coming} alt="mantis" style={{ height: '100%', width: '100%' }} />
          </Box>
        </Grid>
        <Grid item xs={12}>
          <Stack spacing={1} justifyContent="center" alignItems="center" sx={{ mt: -2 }}>
            <Typography align="center" variant="h1">
              Coming Soon
            </Typography>
            <Typography align="center" color="textSecondary">
              Something new is on its way
            </Typography>
          </Stack>
        </Grid>
        <Grid item xs={12} sx={{ width: { xs: '95%', md: '40%' } }}>
          <Stack direction="row" alignItems="center" justifyContent="center" spacing={{ xs: 1, sm: 2 }}>
            <TimerBox count={days} label="day" />
            <Typography variant="h1"> : </Typography>
            <TimerBox count={hours} label="hour" />
            <Typography variant="h1"> : </Typography>
            <TimerBox count={minutes} label="min" />
            <Typography variant="h1"> : </Typography>
            <TimerBox count={seconds} label="sec" />
          </Stack>
        </Grid>
        <Grid item xs={12} sx={{ width: { xs: 380, md: '40%', lg: '30%' } }}>
          <Stack spacing={2} sx={{ mt: 2 }}>
            <Typography align="center" color="textSecondary">
              Be the first to be notified when Mantis launches.
            </Typography>
            <Stack direction="row" spacing={1}>
              <TextField fullWidth placeholder="Email Address" />
              <Button variant="contained" sx={{ width: '50%' }}>
                Notify Me
              </Button>
            </Stack>
          </Stack>
        </Grid>
      </Grid>
    </>
  );
}

export default ComingSoon;
