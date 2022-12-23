// material-ui
import { useTheme } from '@mui/material/styles';
import { Alert, AlertTitle, Button, Grid, Stack, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';

// assets
import {
  BugFilled,
  BugTwoTone,
  CheckSquareFilled,
  CheckSquareTwoTone,
  DatabaseFilled,
  DatabaseTwoTone,
  InfoCircleFilled,
  InfoCircleTwoTone,
  QuestionCircleFilled,
  QuestionCircleOutlined,
  QuestionCircleTwoTone,
  WarningFilled,
  WarningOutlined,
  WarningTwoTone
} from '@ant-design/icons';

// ==============================|| COMPONENTS - ALERTS ||============================== //

const ComponentAlert = () => {
  const theme = useTheme();

  const basicAlertCodeString = `<Alert color="primary" icon={<QuestionCircleFilled />}>
  Primary Text
</Alert>
<Alert color="secondary" icon={<DatabaseFilled />}>
  Secondary Text
</Alert>
<Alert color="success" icon={<CheckSquareFilled />}>
  Success Text
</Alert>
<Alert color="warning" icon={<WarningFilled />}>
  Warning Text
</Alert>
<Alert color="info" icon={<InfoCircleFilled />}>
  Info Text
</Alert>
<Alert color="error" icon={<BugFilled />}>
  Error Text
</Alert>`;

  const actionsAlertCodeString = `<Alert variant="border" color="success" onClose={() => {}}>
  Success Text
</Alert>
<Alert
  variant="border"
  color="warning"
  icon={<WarningOutlined />}
  action={
    <Button color="warning" size="small">
      Undo
    </Button>
  }
>
  Warning Text
</Alert>
<Alert
  variant="border"
  color="primary"
  icon={<QuestionCircleOutlined />}
  action={
    <Button variant="contained" size="small">
      Continue
    </Button>
  }
>
  Primary Text
</Alert>`;

  const filledAlertCodeString = `<Alert color="primary" variant="filled" icon={<QuestionCircleFilled />}>
  Primary Text
</Alert>
<Alert color="secondary" variant="filled" icon={<DatabaseFilled />}>
  Secondary Text
</Alert>
<Alert color="success" variant="filled" icon={<CheckSquareFilled />}>
  Success Text
</Alert>
<Alert color="warning" variant="filled" icon={<WarningFilled />}>
  Warning Text
</Alert>
<Alert color="info" variant="filled" icon={<InfoCircleFilled />}>
  Info Text
</Alert>
<Alert color="error" variant="filled" icon={<BugFilled />}>
  Error Text
</Alert>`;

  const descriptionAlertCodeString = `<Alert color="primary" variant="border" icon={<QuestionCircleFilled />}>
  <AlertTitle>Primary Text</AlertTitle>
  <Typography variant="h6"> This is an primary alert.</Typography>
</Alert>
<Alert color="secondary" variant="border" icon={<DatabaseFilled />}>
  <AlertTitle>Secondary Text</AlertTitle>
  <Typography variant="h6"> This is an secondary alert.</Typography>
</Alert>
<Alert color="success" variant="border" icon={<CheckSquareFilled />}>
  <AlertTitle>Success Text</AlertTitle>
  <Typography variant="h6"> This is an success alert.</Typography>
</Alert>
<Alert color="warning" variant="border" icon={<WarningFilled />}>
  <AlertTitle>Warning Text</AlertTitle>
  <Typography variant="h6"> This is an warning alert.</Typography>
</Alert>
<Alert color="info" variant="border" icon={<InfoCircleFilled />}>
  <AlertTitle>Info Text</AlertTitle>
  <Typography variant="h6"> This is an info alert.</Typography>
</Alert>
<Alert color="error" variant="border" icon={<BugFilled />}>
  <AlertTitle>Error Text</AlertTitle>
  <Typography variant="h6"> This is an error alert.</Typography>
</Alert>`;

  const outlinedAlertCodeString = `<Alert color="primary" variant="outlined" icon={<QuestionCircleTwoTone />}>
  Primary Text
</Alert>
<Alert color="secondary" variant="outlined" icon={<DatabaseTwoTone twoToneColor={theme.palette.secondary.main} />}>
  Secondary Text
</Alert>
<Alert color="success" variant="outlined" icon={<CheckSquareTwoTone twoToneColor={theme.palette.success.main} />}>
  Success Text
</Alert>
<Alert color="warning" variant="outlined" icon={<WarningTwoTone twoToneColor={theme.palette.warning.main} />}>
  Warning Text
</Alert>
<Alert color="info" variant="outlined" icon={<InfoCircleTwoTone twoToneColor={theme.palette.info.main} />}>
  Info Text
</Alert>
<Alert color="error" variant="outlined" icon={<BugTwoTone twoToneColor={theme.palette.error.main} />}>
  Error Text
</Alert>`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Alert"
        caption="An alert displays a short, important message in a way that attracts the user's attention without interrupting the user's task."
        directory="src/pages/components-overview/alert"
        link="https://mui.com/material-ui/react-alert/"
      />
      <ComponentWrapper>
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Stack spacing={3}>
              <MainCard title="Basic" codeHighlight codeString={basicAlertCodeString}>
                <Stack sx={{ width: '100%' }} spacing={2}>
                  <Alert color="primary" icon={<QuestionCircleFilled />}>
                    Primary Text
                  </Alert>
                  <Alert color="secondary" icon={<DatabaseFilled />}>
                    Secondary Text
                  </Alert>
                  <Alert color="success" icon={<CheckSquareFilled />}>
                    Success Text
                  </Alert>
                  <Alert color="warning" icon={<WarningFilled />}>
                    Warning Text
                  </Alert>
                  <Alert color="info" icon={<InfoCircleFilled />}>
                    Info Text
                  </Alert>
                  <Alert color="error" icon={<BugFilled />}>
                    Error Text
                  </Alert>
                </Stack>
              </MainCard>
              <MainCard title="Actions" codeString={actionsAlertCodeString}>
                <Stack sx={{ width: '100%' }} spacing={2}>
                  <Alert
                    variant="border"
                    color="success"
                    onClose={() => {}}
                    sx={{
                      '& .MuiIconButton-root:focus-visible': {
                        outline: `2px solid ${theme.palette.success.dark}`,
                        outlineOffset: 2
                      }
                    }}
                  >
                    Success Text
                  </Alert>
                  <Alert
                    variant="border"
                    color="warning"
                    icon={<WarningOutlined />}
                    action={
                      <Button color="warning" size="small">
                        Undo
                      </Button>
                    }
                  >
                    Warning Text
                  </Alert>
                  <Alert
                    variant="border"
                    color="primary"
                    icon={<QuestionCircleOutlined />}
                    action={
                      <Button variant="contained" size="small">
                        Continue
                      </Button>
                    }
                  >
                    Primary Text
                  </Alert>
                </Stack>
              </MainCard>
              <MainCard title="Filled" codeString={filledAlertCodeString}>
                <Stack sx={{ width: '100%' }} spacing={2}>
                  <Alert color="primary" variant="filled" icon={<QuestionCircleFilled />}>
                    Primary Text
                  </Alert>
                  <Alert color="secondary" variant="filled" icon={<DatabaseFilled />}>
                    Secondary Text
                  </Alert>
                  <Alert color="success" variant="filled" icon={<CheckSquareFilled />}>
                    Success Text
                  </Alert>
                  <Alert color="warning" variant="filled" icon={<WarningFilled />}>
                    Warning Text
                  </Alert>
                  <Alert color="info" variant="filled" icon={<InfoCircleFilled />}>
                    Info Text
                  </Alert>
                  <Alert color="error" variant="filled" icon={<BugFilled />}>
                    Error Text
                  </Alert>
                </Stack>
              </MainCard>
            </Stack>
          </Grid>
          <Grid item xs={12} md={6}>
            <Stack spacing={3}>
              <MainCard title="Description" codeString={descriptionAlertCodeString}>
                <Stack sx={{ width: '100%' }} spacing={2}>
                  <Alert color="primary" variant="border" icon={<QuestionCircleFilled />}>
                    <AlertTitle>Primary Text</AlertTitle>
                    <Typography variant="h6"> This is an primary alert.</Typography>
                  </Alert>
                  <Alert color="secondary" variant="border" icon={<DatabaseFilled />}>
                    <AlertTitle>Secondary Text</AlertTitle>
                    <Typography variant="h6"> This is an secondary alert.</Typography>
                  </Alert>
                  <Alert color="success" variant="border" icon={<CheckSquareFilled />}>
                    <AlertTitle>Success Text</AlertTitle>
                    <Typography variant="h6"> This is an success alert.</Typography>
                  </Alert>
                  <Alert color="warning" variant="border" icon={<WarningFilled />}>
                    <AlertTitle>Warning Text</AlertTitle>
                    <Typography variant="h6"> This is an warning alert.</Typography>
                  </Alert>
                  <Alert color="info" variant="border" icon={<InfoCircleFilled />}>
                    <AlertTitle>Info Text</AlertTitle>
                    <Typography variant="h6"> This is an info alert.</Typography>
                  </Alert>
                  <Alert color="error" variant="border" icon={<BugFilled />}>
                    <AlertTitle>Error Text</AlertTitle>
                    <Typography variant="h6"> This is an error alert.</Typography>
                  </Alert>
                </Stack>
              </MainCard>
              <MainCard title="Outlined" codeString={outlinedAlertCodeString}>
                <Stack sx={{ width: '100%' }} spacing={2}>
                  <Alert color="primary" variant="outlined" icon={<QuestionCircleTwoTone />}>
                    Primary Text
                  </Alert>
                  <Alert color="secondary" variant="outlined" icon={<DatabaseTwoTone twoToneColor={theme.palette.secondary.main} />}>
                    Secondary Text
                  </Alert>
                  <Alert color="success" variant="outlined" icon={<CheckSquareTwoTone twoToneColor={theme.palette.success.main} />}>
                    Success Text
                  </Alert>
                  <Alert color="warning" variant="outlined" icon={<WarningTwoTone twoToneColor={theme.palette.warning.main} />}>
                    Warning Text
                  </Alert>
                  <Alert color="info" variant="outlined" icon={<InfoCircleTwoTone twoToneColor={theme.palette.info.main} />}>
                    Info Text
                  </Alert>
                  <Alert color="error" variant="outlined" icon={<BugTwoTone twoToneColor={theme.palette.error.main} />}>
                    Error Text
                  </Alert>
                </Stack>
              </MainCard>
            </Stack>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentAlert;
