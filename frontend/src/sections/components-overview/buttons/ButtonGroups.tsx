// material-ui
import { Button, ButtonGroup, Grid } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// ==============================|| BUTTON GROUPS ||============================== //

const ButtonGroups = () => {
  const buttons = [<Button key="one">One</Button>, <Button key="two">Two</Button>, <Button key="three">Three</Button>];

  const groupButtonCodeString = `<ButtonGroup disableElevation variant="contained" aria-label="outlined primary button group">
  <Button key="one">One</Button>
  <Button key="two">Two</Button>
  <Button key="three">Three</Button>
</ButtonGroup>
<ButtonGroup variant="outlined" aria-label="outlined button group">
  <Button key="one">One</Button>
  <Button key="two">Two</Button>
  <Button key="three">Three</Button>
</ButtonGroup>
<ButtonGroup variant="text" aria-label="text button group">
  <Button key="one">One</Button>
  <Button key="two">Two</Button>
  <Button key="three">Three</Button>
</ButtonGroup>
<ButtonGroup color="warning" aria-label="medium secondary button group">
  <Button key="one">One</Button>
  <Button key="two">Two</Button>
  <Button key="three">Three</Button>
</ButtonGroup>
<ButtonGroup orientation="vertical" aria-label="vertical outlined button group">
  <Button key="one">One</Button>
  <Button key="two">Two</Button>
  <Button key="three">Three</Button>
</ButtonGroup>`;

  return (
    <MainCard title="Button Group" codeString={groupButtonCodeString}>
      <Grid container direction="column" spacing={2}>
        <Grid item>
          <ButtonGroup disableElevation variant="contained" aria-label="outlined primary button group">
            {buttons}
          </ButtonGroup>
        </Grid>
        <Grid item>
          <ButtonGroup variant="outlined" aria-label="outlined button group">
            {buttons}
          </ButtonGroup>
        </Grid>
        <Grid item>
          <ButtonGroup variant="text" aria-label="text button group">
            {buttons}
          </ButtonGroup>
        </Grid>
        <Grid item>
          <ButtonGroup color="warning" aria-label="medium secondary button group">
            {buttons}
          </ButtonGroup>
        </Grid>
        <Grid item>
          <ButtonGroup orientation="vertical" aria-label="vertical outlined button group">
            {buttons}
          </ButtonGroup>
        </Grid>
      </Grid>
    </MainCard>
  );
};

export default ButtonGroups;
