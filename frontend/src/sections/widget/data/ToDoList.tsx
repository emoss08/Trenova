import { useState, ChangeEvent } from 'react';

// material-ui
import { CardContent, Checkbox, FormControlLabel, Grid, Tooltip } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';

// assets
import { PlusCircleOutlined } from '@ant-design/icons';
import IconButton from 'components/@extended/IconButton';

// ===========================|| DATA WIDGET - TODO LIST ||=========================== //

const ToDoList = () => {
  const [state, setState] = useState({
    checkedA: true,
    checkedB: true,
    checkedC: true,
    checkedD: false,
    checkedE: false,
    checkedF: false,
    checkedG: false
  });

  const handleChangeState = (event: ChangeEvent<HTMLInputElement>) => {
    setState({ ...state, [event.target.name]: event.target.checked });
  };

  return (
    <MainCard
      title="To Do List"
      content={false}
      secondary={
        <Tooltip title="Add Task">
          <IconButton>
            <PlusCircleOutlined />
          </IconButton>
        </Tooltip>
      }
      sx={{ '& .MuiCardHeader-root': { p: 1.75 } }}
    >
      <CardContent>
        <Grid container spacing={0} sx={{ '& .Mui-checked + span': { textDecoration: 'line-through' } }}>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox checked={state.checkedA} onChange={handleChangeState} name="checkedA" color="primary" />}
              label="Check your Email"
            />
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox checked={state.checkedB} onChange={handleChangeState} name="checkedB" color="primary" />}
              label="Make YouTube Video"
            />
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox checked={state.checkedC} onChange={handleChangeState} name="checkedC" color="primary" />}
              label="Create Banner"
            />
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox checked={state.checkedD} onChange={handleChangeState} name="checkedD" color="primary" />}
              label="Upload Project"
            />
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox checked={state.checkedE} onChange={handleChangeState} name="checkedE" color="primary" />}
              label="Update Task"
            />
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox checked={state.checkedF} onChange={handleChangeState} name="checkedF" color="primary" />}
              label="Task Issue"
            />
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={<Checkbox checked={state.checkedG} onChange={handleChangeState} name="checkedG" color="primary" />}
              label="Deploy Project"
            />
          </Grid>
        </Grid>
      </CardContent>
    </MainCard>
  );
};

export default ToDoList;
