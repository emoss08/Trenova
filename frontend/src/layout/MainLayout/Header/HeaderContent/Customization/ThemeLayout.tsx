import { useState, ChangeEvent } from 'react';
import { useDispatch, useSelector } from 'react-redux';

// material-ui
import { useTheme } from '@mui/material/styles';
import { CardMedia, FormControlLabel, Grid, Radio, RadioGroup, Stack, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import useConfig from 'hooks/useConfig';
import { openDrawer } from 'store/reducers/menu';

// assets
import defaultLayout from 'assets/images/customization/default.svg';
import rtlLayout from 'assets/images/customization/rtl.svg';
import miniMenu from 'assets/images/customization/mini-menu.svg';

// types
import { RootStateProps } from 'types/root';

// ==============================|| CUSTOMIZATION - LAYOUT ||============================== //

const ThemeLayout = () => {
  const theme = useTheme();
  const dispatch = useDispatch();

  const { miniDrawer, themeDirection, onChangeDirection, onChangeMiniDrawer } = useConfig();
  const { drawerOpen } = useSelector((state: RootStateProps) => state.menu);

  let initialTheme = 'default';
  if (miniDrawer === true) initialTheme = 'mini';
  if (themeDirection === 'rtl') initialTheme = 'rtl';

  const [value, setValue] = useState<string | null>(initialTheme);
  const handleRadioChange = (event: ChangeEvent<HTMLInputElement>) => {
    const newValue = event.target.value;
    setValue(newValue);
    if (newValue === 'default') {
      onChangeDirection('ltr');
      if (!drawerOpen) {
        dispatch(openDrawer({ drawerOpen: true }));
      }
    }
    if (newValue === 'mini') {
      onChangeMiniDrawer(true);
      if (drawerOpen) {
        dispatch(openDrawer({ drawerOpen: false }));
      }
    }
    if (newValue === 'rtl') {
      onChangeDirection('rtl');
    }
  };

  return (
    <RadioGroup row aria-label="payment-card" name="payment-card" value={value} onChange={handleRadioChange}>
      <Grid container spacing={1.75} sx={{ ml: 0 }}>
        <Grid item>
          <FormControlLabel
            value="default"
            control={<Radio value="default" sx={{ display: 'none' }} />}
            sx={{ display: 'flex', '& .MuiFormControlLabel-label': { flex: 1 } }}
            label={
              <MainCard
                content={false}
                sx={{ bgcolor: value === 'default' ? 'primary.lighter' : 'secondary.lighter', p: 1 }}
                border={false}
                {...(value === 'default' && { boxShadow: true, shadow: theme.customShadows.primary })}
              >
                <Stack spacing={1.25} alignItems="center">
                  <CardMedia component="img" src={defaultLayout} alt="Vertical" sx={{ borderRadius: 1, width: 64, height: 64 }} />
                  <Typography variant="caption">Default</Typography>
                </Stack>
              </MainCard>
            }
          />
        </Grid>
        <Grid item>
          <FormControlLabel
            value="mini"
            control={<Radio value="mini" sx={{ display: 'none' }} />}
            sx={{ display: 'flex', '& .MuiFormControlLabel-label': { flex: 1 } }}
            label={
              <MainCard
                content={false}
                sx={{ bgcolor: value === 'mini' ? 'primary.lighter' : 'secondary.lighter', p: 1 }}
                border={false}
                {...(value === 'mini' && { boxShadow: true, shadow: theme.customShadows.primary })}
              >
                <Stack spacing={1.25} alignItems="center">
                  <CardMedia component="img" src={miniMenu} alt="Vertical" sx={{ borderRadius: 1, width: 64, height: 64 }} />
                  <Typography variant="caption">Mini Drawer</Typography>
                </Stack>
              </MainCard>
            }
          />
        </Grid>
        <Grid item>
          <FormControlLabel
            value="rtl"
            control={<Radio value="rtl" sx={{ display: 'none' }} />}
            sx={{ display: 'flex', '& .MuiFormControlLabel-label': { flex: 1 } }}
            label={
              <MainCard
                content={false}
                sx={{ bgcolor: value === 'rtl' ? 'primary.lighter' : 'secondary.lighter', p: 1 }}
                border={false}
                {...(value === 'rtl' && { boxShadow: true, shadow: theme.customShadows.primary })}
              >
                <Stack spacing={1.25} alignItems="center">
                  <CardMedia component="img" src={rtlLayout} alt="Vertical" sx={{ borderRadius: 1, width: 64, height: 64 }} />
                  <Typography variant="caption">RTL</Typography>
                </Stack>
              </MainCard>
            }
          />
        </Grid>
      </Grid>
    </RadioGroup>
  );
};

export default ThemeLayout;
