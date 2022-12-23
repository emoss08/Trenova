import { useState } from 'react';

// material-ui
import { FormControl, Grid, InputAdornment, InputLabel, OutlinedInput, Stack, TextField } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import IconButton from 'components/@extended/IconButton';
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';

// assets
import { EyeInvisibleOutlined, EyeOutlined, MailOutlined } from '@ant-design/icons';

// ==============================|| COMPONENTS - TEXT FEILD ||============================== //

interface State {
  password: string;
  showPassword: boolean;
}

const ComponentTextField = () => {
  const [values, setValues] = useState({
    password: '',
    showPassword: false
  });

  const handleChange = (prop: keyof State) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setValues({ ...values, [prop]: event.target.value });
  };

  const handleClickShowPassword = () => {
    setValues({
      ...values,
      showPassword: !values.showPassword
    });
  };

  const handleMouseDownPassword = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
  };

  const basicTextfeildCodeString = `<TextField id="outlined-basic" label="Outlined" />
<TextField id="filled-basic" label="Filled" variant="filled" />
<TextField id="standard-basic" label="Standard" variant="standard" />`;

  const propsTextfeildCodeString = `<TextField required id="outlined-required" placeholder="Required *" defaultValue="Hello World" />
<TextField id="helper-text-basic" placeholder="Helper text" helperText="Helper text" />
<TextField id="outlined-number" label="Number" type="number" />
<TextField
  id="outlined-number"
  defaultValue="Read Only"
  InputProps={{
    readOnly: true
  }}
/>
<OutlinedInput
  id="outlined-adornment-password"
  type={values.showPassword ? 'text' : 'password'}
  value={values.password}
  onChange={handleChange('password')}
  endAdornment={
    <InputAdornment position="end">
      <IconButton
        aria-label="toggle password visibility"
        onClick={handleClickShowPassword}
        onMouseDown={handleMouseDownPassword}
        edge="end"
        color="secondary"
      >
        {values.showPassword ? <EyeOutlined /> : <EyeInvisibleOutlined />}
      </IconButton>
    </InputAdornment>
  }
/>
<FormControl variant="standard">
  <Stack spacing={3}>
    <InputLabel shrink htmlFor="with-label-input">
      With Label
    </InputLabel>
    <TextField id="with-label-input" placeholder="With Label" />
  </Stack>
</FormControl>
<TextField id="disabled-basic" label="Disabled" disabled />
<TextField id="filled-search" placeholder="Search" type="search" />`;

  const iconTextfeildCodeString = `<OutlinedInput id="start-adornment-email" placeholder="Email / UserId" startAdornment={<MailOutlined />} />
<OutlinedInput
  id="end-adornment-password"
  type="password"
  placeholder="Password"
  endAdornment={
    <InputAdornment position="end">
      <IconButton aria-label="toggle password visibility" edge="end" color="secondary">
        <EyeInvisibleOutlined />
      </IconButton>
    </InputAdornment>
  }
/>`;

  const sizeTextfeildCodeString = `<TextField id="outlined-basic-small" label="Small" size="small" />
<TextField id="outlined-basic-default" label="Medium" />
<TextField
  id="outlined-basic-custom"
  label="Custom"
  sx={{
    '& .MuiInputLabel-root': { fontSize: '1rem' },
    '& .MuiOutlinedInput-root': { fontSize: '1rem' }
  }}
/>`;

  const eventTextfeildCodeString = `<TextField id="outlined-basic-auto" label="Auto Focus" autoFocus />`;

  const validationTextfeildCodeString = `<TextField error id="outlined-error" label="Error" defaultValue="Hello World" />
<TextField
  error
  id="outlined-error-helper-text"
  label="Error"
  defaultValue="Hello World"
  helperText="Incorrect entry."
/>`;

  const multilineTextfeildCodeString = `<TextField
  id="outlined-multiline-static"
  fullWidth
  label="Multiline"
  multiline
  rows={5}
  defaultValue="Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text"
/>`;

  const adornmentTextfeildCodeString = `<TextField
  placeholder="Website URL"
  id="url-start-adornment"
  InputProps={{
    startAdornment: 'https://'
  }}
/>
<TextField
  placeholder="Website URL"
  id="outlined-end-adornment"
  InputProps={{
    endAdornment: '.com'
  }}
/>
<OutlinedInput
  id="text-adornment-password"
  type="password"
  placeholder="Password"
  endAdornment={
    <InputAdornment position="end">
      <IconButton aria-label="toggle password visibility" edge="end" color="secondary">
        <EyeInvisibleOutlined />
      </IconButton>
    </InputAdornment>
  }
/>
<TextField
  placeholder="0.00"
  id="outlined-start-adornment"
  InputProps={{
    startAdornment: '$'
  }}
/>`;

  const widthTextfeildCodeString = `<TextField fullWidth id="outlined-basic-fullwidth" label="Fullwidth" />`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Text Field"
        caption="Text fields let users enter and edit text."
        directory="src/pages/components-overview/textfield"
        link="https://mui.com/material-ui/react-text-field/"
      />
      <ComponentWrapper>
        <Grid container spacing={3}>
          <Grid item xs={12} lg={6}>
            <Stack spacing={3}>
              <MainCard title="Basic" codeHighlight codeString={basicTextfeildCodeString}>
                <Stack spacing={2}>
                  <TextField id="outlined-basic" label="Outlined" />
                  <TextField id="filled-basic" label="Filled" variant="filled" />
                  <TextField id="standard-basic" label="Standard" variant="standard" />
                </Stack>
              </MainCard>
              <MainCard title="Form Props" codeString={propsTextfeildCodeString}>
                <Grid container spacing={2}>
                  <Grid item xs={12} md={6}>
                    <Stack spacing={2}>
                      <TextField required id="outlined-required" placeholder="Required *" defaultValue="Hello World" />
                      <TextField id="helper-text-basic" placeholder="Helper text" helperText="Helper text" />
                      <TextField id="outlined-number" label="Number" type="number" />
                      <TextField
                        id="outlined-number"
                        defaultValue="Read Only"
                        InputProps={{
                          readOnly: true
                        }}
                      />
                    </Stack>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Stack spacing={2}>
                      <OutlinedInput
                        id="outlined-adornment-password"
                        type={values.showPassword ? 'text' : 'password'}
                        value={values.password}
                        onChange={handleChange('password')}
                        endAdornment={
                          <InputAdornment position="end">
                            <IconButton
                              aria-label="toggle password visibility"
                              onClick={handleClickShowPassword}
                              onMouseDown={handleMouseDownPassword}
                              edge="end"
                              color="secondary"
                            >
                              {values.showPassword ? <EyeOutlined /> : <EyeInvisibleOutlined />}
                            </IconButton>
                          </InputAdornment>
                        }
                      />
                      <FormControl variant="standard">
                        <Stack spacing={3}>
                          <InputLabel shrink htmlFor="with-label-input">
                            With Label
                          </InputLabel>
                          <TextField id="with-label-input" placeholder="With Label" />
                        </Stack>
                      </FormControl>
                      <TextField id="disabled-basic" label="Disabled" disabled />
                      <TextField id="filled-search" placeholder="Search" type="search" />
                    </Stack>
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="With Icon" codeString={iconTextfeildCodeString}>
                <Stack spacing={2}>
                  <OutlinedInput id="start-adornment-email" placeholder="Email / UserId" startAdornment={<MailOutlined />} />
                  <OutlinedInput
                    id="end-adornment-password"
                    type="password"
                    placeholder="Password"
                    endAdornment={
                      <InputAdornment position="end">
                        <IconButton aria-label="toggle password visibility" edge="end" color="secondary">
                          <EyeInvisibleOutlined />
                        </IconButton>
                      </InputAdornment>
                    }
                  />
                </Stack>
              </MainCard>
              <MainCard title="Sizes" codeString={sizeTextfeildCodeString}>
                <Stack spacing={2}>
                  <TextField id="outlined-basic-small" label="Small" size="small" />
                  <TextField id="outlined-basic-default" label="Medium" />
                  <TextField
                    id="outlined-basic-custom"
                    label="Custom"
                    sx={{
                      '& .MuiInputLabel-root': { fontSize: '1rem' },
                      '& .MuiOutlinedInput-root': { fontSize: '1rem' }
                    }}
                  />
                </Stack>
              </MainCard>
            </Stack>
          </Grid>
          <Grid item xs={12} lg={6}>
            <Stack spacing={3}>
              <MainCard title="Event" codeString={eventTextfeildCodeString}>
                <TextField id="outlined-basic-auto" label="Auto Focus" autoFocus />
              </MainCard>
              <MainCard title="Validation" codeString={validationTextfeildCodeString}>
                <Grid container spacing={2}>
                  <Grid item xs={12} md={6}>
                    <TextField error id="outlined-error" label="Error" defaultValue="Hello World" />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      error
                      id="outlined-error-helper-text"
                      label="Error"
                      defaultValue="Hello World"
                      helperText="Incorrect entry."
                    />
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Multiline" codeString={multilineTextfeildCodeString}>
                <TextField
                  id="outlined-multiline-static"
                  fullWidth
                  label="Multiline"
                  multiline
                  rows={5}
                  defaultValue="Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text"
                />
              </MainCard>
              <MainCard title="Input Adornments" codeString={adornmentTextfeildCodeString}>
                <Grid container spacing={2}>
                  <Grid item xs={12} md={6}>
                    <Stack spacing={2}>
                      <TextField
                        placeholder="Website URL"
                        id="url-start-adornment"
                        InputProps={{
                          startAdornment: 'https://'
                        }}
                      />
                      <TextField
                        placeholder="Website URL"
                        id="outlined-end-adornment"
                        InputProps={{
                          endAdornment: '.com'
                        }}
                      />
                    </Stack>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Stack spacing={2}>
                      <OutlinedInput
                        id="text-adornment-password"
                        type="password"
                        placeholder="Password"
                        endAdornment={
                          <InputAdornment position="end">
                            <IconButton aria-label="toggle password visibility" edge="end" color="secondary">
                              <EyeInvisibleOutlined />
                            </IconButton>
                          </InputAdornment>
                        }
                      />
                      <TextField
                        placeholder="0.00"
                        id="outlined-start-adornment"
                        InputProps={{
                          startAdornment: '$'
                        }}
                      />
                    </Stack>
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Full Width" codeString={widthTextfeildCodeString}>
                <TextField fullWidth id="outlined-basic-fullwidth" label="Fullwidth" />
              </MainCard>
            </Stack>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentTextField;
