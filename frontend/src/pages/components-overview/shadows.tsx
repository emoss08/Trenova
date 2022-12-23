// material-ui
import { useTheme } from '@mui/material/styles';
import { Grid, Stack, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';

// ===============================|| SHADOW BOX ||=============================== //

function ShadowBox({ shadow }: { shadow: string }) {
  return (
    <MainCard border={false} shadow={shadow} boxShadow>
      <Stack spacing={1} justifyContent="center" alignItems="center">
        <Typography variant="h6">boxShadow</Typography>
        <Typography variant="subtitle1">{shadow}</Typography>
      </Stack>
    </MainCard>
  );
}

// ===============================|| CUSTOM - SHADOW BOX ||=============================== //

function CustomShadowBox({ shadow, label, color, bgcolor }: { shadow: string; label: string; color: string; bgcolor?: string }) {
  return (
    <MainCard border={false} shadow={shadow} boxShadow sx={{ bgcolor: bgcolor || 'inherit' }}>
      <Stack spacing={1} justifyContent="center" alignItems="center">
        <Typography variant="subtitle1" color={color}>
          {label}
        </Typography>
      </Stack>
    </MainCard>
  );
}

// ============================|| COMPONENT - SHADOW ||============================ //

const ComponentShadow = () => {
  const theme = useTheme();

  const basicShadowCodeString = `<ShadowBox shadow="0" />
<ShadowBox shadow="1" />
<ShadowBox shadow="2" />
<ShadowBox shadow="3" />
<ShadowBox shadow="4" />
<ShadowBox shadow="5" />
<ShadowBox shadow="6" />
<ShadowBox shadow="7" />
<ShadowBox shadow="8" />
<ShadowBox shadow="9" />
<ShadowBox shadow="10" />
<ShadowBox shadow="11" />
<ShadowBox shadow="12" />
<ShadowBox shadow="13" />
<ShadowBox shadow="14" />
<ShadowBox shadow="15" />
<ShadowBox shadow="16" />
<ShadowBox shadow="17" />
<ShadowBox shadow="18" />
<ShadowBox shadow="19" />
<ShadowBox shadow="20" />
<ShadowBox shadow="21" />
<ShadowBox shadow="22" />
<ShadowBox shadow="23" />
<ShadowBox shadow="24" />`;

  const customShadowCodeString = `<CustomShadowBox shadow={theme.customShadows.z1} label="z1" color="inherit" />`;

  const colorShadowCodeString = `<CustomShadowBox
  color={theme.palette.primary.contrastText}
  bgcolor={theme.palette.primary.main}
  shadow={theme.customShadows.primaryButton}
  label="primary"
/>
<CustomShadowBox
  color={theme.palette.secondary.contrastText}
  bgcolor={theme.palette.secondary.main}
  shadow={theme.customShadows.secondaryButton}
  label="secondary"
/>
<CustomShadowBox
  color={theme.palette.success.contrastText}
  bgcolor={theme.palette.success.main}
  shadow={theme.customShadows.successButton}
  label="success"
/>
<CustomShadowBox
  color={theme.palette.warning.contrastText}
  bgcolor={theme.palette.warning.main}
  shadow={theme.customShadows.warningButton}
  label="warning"
/>
<CustomShadowBox
  color={theme.palette.info.contrastText}
  bgcolor={theme.palette.info.main}
  shadow={theme.customShadows.infoButton}
  label="info"
/>
<CustomShadowBox
  color={theme.palette.error.contrastText}
  bgcolor={theme.palette.error.main}
  shadow={theme.customShadows.errorButton}
  label="error"
/>
<CustomShadowBox color={theme.palette.primary.main} shadow={theme.customShadows.primary} label="primary" />
<CustomShadowBox color={theme.palette.secondary.main} shadow={theme.customShadows.secondary} label="secondary" />
<CustomShadowBox color={theme.palette.success.main} shadow={theme.customShadows.success} label="success" />
<CustomShadowBox color={theme.palette.warning.main} shadow={theme.customShadows.warning} label="warning" />
<CustomShadowBox color={theme.palette.info.main} shadow={theme.customShadows.info} label="info" />
<CustomShadowBox color={theme.palette.error.main} shadow={theme.customShadows.error} label="error" />`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Shadows"
        caption="Add or remove shadows to elements with box-shadow utilities."
        directory="src/pages/components-overview/shadows"
        link="https://mui.com/system/shadows/"
      />
      <ComponentWrapper>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <MainCard title="Basic Shadow" codeHighlight codeString={basicShadowCodeString}>
              <Grid container spacing={3}>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="0" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="1" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="2" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="3" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="4" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="5" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="6" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="7" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="8" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="9" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="10" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="11" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="12" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="13" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="14" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="15" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="16" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="17" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="18" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="19" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="20" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="21" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="22" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="23" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <ShadowBox shadow="24" />
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12}>
            <MainCard title="Custom Shadow" codeString={customShadowCodeString}>
              <Grid container spacing={3}>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox shadow={theme.customShadows.z1} label="z1" color="inherit" />
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12}>
            <MainCard title="Color Shadow" codeString={colorShadowCodeString}>
              <Grid container spacing={3}>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox
                    color={theme.palette.primary.contrastText}
                    bgcolor={theme.palette.primary.main}
                    shadow={theme.customShadows.primaryButton}
                    label="primary"
                  />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox
                    color={theme.palette.secondary.contrastText}
                    bgcolor={theme.palette.secondary.main}
                    shadow={theme.customShadows.secondaryButton}
                    label="secondary"
                  />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox
                    color={theme.palette.success.contrastText}
                    bgcolor={theme.palette.success.main}
                    shadow={theme.customShadows.successButton}
                    label="success"
                  />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox
                    color={theme.palette.warning.contrastText}
                    bgcolor={theme.palette.warning.main}
                    shadow={theme.customShadows.warningButton}
                    label="warning"
                  />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox
                    color={theme.palette.info.contrastText}
                    bgcolor={theme.palette.info.main}
                    shadow={theme.customShadows.infoButton}
                    label="info"
                  />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox
                    color={theme.palette.error.contrastText}
                    bgcolor={theme.palette.error.main}
                    shadow={theme.customShadows.errorButton}
                    label="error"
                  />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox color={theme.palette.primary.main} shadow={theme.customShadows.primary} label="primary" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox color={theme.palette.secondary.main} shadow={theme.customShadows.secondary} label="secondary" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox color={theme.palette.success.main} shadow={theme.customShadows.success} label="success" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox color={theme.palette.warning.main} shadow={theme.customShadows.warning} label="warning" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox color={theme.palette.info.main} shadow={theme.customShadows.info} label="info" />
                </Grid>
                <Grid item xs={6} sm={4} md={3} lg={2}>
                  <CustomShadowBox color={theme.palette.error.main} shadow={theme.customShadows.error} label="error" />
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentShadow;
