import { useEffect, useState, ChangeEvent } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, Button, FormLabel, Grid, InputLabel, MenuItem, Select, Stack, TextField, Typography } from '@mui/material';

// third-party
import NumericFormat from 'react-number-format';
import ContentLoader from 'react-content-loader';

// project import
import Avatar from 'components/@extended/Avatar';
import MainCard from 'components/MainCard';

// assets
import { CameraOutlined } from '@ant-design/icons';
import axios from 'axios';
import useAuth from '../../../../hooks/useAuth';
import LocalStorageService from '../../../../services/LocalStorageService';

const avatarImage = require.context('assets/images/users', true);

// ==============================|| ACCOUNT PROFILE - PERSONAL ||============================== //

const TabPersonal = () => {
  interface IValues {
    isLoading: boolean;
    id: string;
    username: string;
    organization: string;
    department: string;
    email: string;
    profile: {
      id: string;
      first_name: string;
      last_name: string;
      title: string;
      phone: string;
      bio: string;
      address_line_1: string;
      address_line_2: string;
      city: string;
      state: string;
      zip_code: string;
    };
  }

  const theme = useTheme();
  const [selectedImage, setSelectedImage] = useState<File | undefined>(undefined);
  const [avatar, setAvatar] = useState<string | undefined>(avatarImage(`./avatar-1.png`));
  const { user } = useAuth();
  const [values, setValues] = useState<IValues>({} as IValues);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  useEffect(() => {
    if (selectedImage) {
      setAvatar(URL.createObjectURL(selectedImage));
    }
    const getProfile = async () => {
      const response = await axios.get(`http://127.0.0.1:8000/api/users/${user?.uid}/`, {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `token ${LocalStorageService.getToken()}`
        }
      });
      setValues(response.data as IValues);
    };

    getProfile().then(() => setIsLoading(false));
  }, [selectedImage, user]);

  if (isLoading) {
    return (
      <ContentLoader speed="2" foregroundOpacity="0.3" backgroundOpacity="0.1" viewBox="0 0 380 100">
        <rect x="0" y="0" rx="2" ry="2" width="70" height="100" />
        <rect x="80" y="5" rx="2" ry="2" width="100" height="10" />
        <rect x="190" y="5" rx="2" ry="2" width="100" height="10" />
        <rect x="80" y="20" rx="2" ry="2" width="100" height="10" />
        <rect x="190" y="20" rx="2" ry="2" width="100" height="10" />
        <rect x="80" y="35" rx="2" ry="2" width="100" height="10" />
        <rect x="190" y="35" rx="2" ry="2" width="100" height="10" />
      </ContentLoader>
    );
  }

  return (
    <Grid container spacing={3}>
      <Grid item xs={12} sm={6}>
        <MainCard title="Personal Information">
          <Grid container spacing={3}>
            <Grid item xs={12}>
              <Stack spacing={2.5} alignItems="center" sx={{ m: 3 }}>
                <FormLabel
                  htmlFor="change-avtar"
                  sx={{
                    position: 'relative',
                    borderRadius: '50%',
                    overflow: 'hidden',
                    '&:hover .MuiBox-root': { opacity: 1 },
                    cursor: 'pointer'
                  }}
                >
                  <Avatar alt="Avatar 1" src={avatar} sx={{ width: 76, height: 76 }} />
                  <Box
                    sx={{
                      position: 'absolute',
                      top: 0,
                      left: 0,
                      backgroundColor: theme.palette.mode === 'dark' ? 'rgba(255, 255, 255, .75)' : 'rgba(0,0,0,.65)',
                      width: '100%',
                      height: '100%',
                      opacity: 0,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center'
                    }}
                  >
                    <Stack spacing={0.5} alignItems="center">
                      <CameraOutlined style={{ color: theme.palette.secondary.lighter, fontSize: '1.5rem' }} />
                      <Typography sx={{ color: 'secondary.lighter' }} variant="caption">
                        Upload
                      </Typography>
                    </Stack>
                  </Box>
                </FormLabel>
                <TextField
                  type="file"
                  id="change-avtar"
                  label="Outlined"
                  variant="outlined"
                  sx={{ display: 'none' }}
                  onChange={(e: ChangeEvent<HTMLInputElement>) => setSelectedImage(e.target.files?.[0])}
                />
              </Stack>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Stack spacing={1.25}>
                <InputLabel htmlFor="personal-first-name">First Name</InputLabel>
                <TextField
                  fullWidth
                  name="first_name"
                  value={values.profile.first_name}
                  id="personal-first-name"
                  placeholder="First Name"
                  autoFocus
                />
              </Stack>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Stack spacing={1.25}>
                <InputLabel htmlFor="personal-last-name">Last Name</InputLabel>
                <TextField fullWidth name="last_name" value={values.profile.last_name} id="personal-last-name" placeholder="Last Name" />
              </Stack>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Stack spacing={1.25}>
                <InputLabel htmlFor="personal-city">Address Line 1</InputLabel>
                <TextField
                  fullWidth
                  name="city"
                  value={values.profile.address_line_1}
                  id="personal-address_line_1"
                  placeholder="Address Line 1 (Primary Address)"
                />
              </Stack>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Stack spacing={1.25}>
                <InputLabel htmlFor="personal-city">Address Line 2</InputLabel>
                <TextField
                  fullWidth
                  name="city"
                  value={values.profile.address_line_2}
                  id="personal-address_line_2"
                  placeholder="Address Line 1 (Primary Address)"
                />
              </Stack>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Stack spacing={1.25}>
                <InputLabel htmlFor="personal-city">City</InputLabel>
                <TextField fullWidth name="city" value={values.profile.city} id="personal-city" placeholder="City" />
              </Stack>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Stack spacing={1.25}>
                <InputLabel htmlFor="personal-zipcode">Zipcode</InputLabel>
                <TextField fullWidth defaultValue="956754" id="personal-zipcode" placeholder="Zipcode" />
              </Stack>
            </Grid>
            <Grid item xs={12}>
              <Stack spacing={1.25}>
                <InputLabel htmlFor="personal-location">Bio</InputLabel>
                <TextField fullWidth multiline rows={3} value={values.profile.bio} id="personal-location" placeholder="Location" />
              </Stack>
            </Grid>
          </Grid>
        </MainCard>
      </Grid>
      <Grid item xs={12} sm={6}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <MainCard title="Contact Information">
              <Grid container spacing={3}>
                <Grid item xs={12} sm={6}>
                  <Stack spacing={1.25}>
                    <InputLabel htmlFor="personal-phone">Phone Number</InputLabel>
                    <Stack direction="row" justifyContent="space-between" alignItems="center" spacing={2}>
                      <Select defaultValue="1-876">
                        <MenuItem value="91">+91</MenuItem>
                        <MenuItem value="1-671">1-671</MenuItem>
                        <MenuItem value="36">+36</MenuItem>
                        <MenuItem value="225">(255)</MenuItem>
                        <MenuItem value="39">+39</MenuItem>
                        <MenuItem value="1-876">1-876</MenuItem>
                        <MenuItem value="7">+7</MenuItem>
                        <MenuItem value="254">(254)</MenuItem>
                        <MenuItem value="373">(373)</MenuItem>
                        <MenuItem value="1-664">1-664</MenuItem>
                        <MenuItem value="95">+95</MenuItem>
                        <MenuItem value="264">(264)</MenuItem>
                      </Select>
                      <NumericFormat
                        format="+1 (###) ###-####"
                        mask="_"
                        fullWidth
                        customInput={TextField}
                        label="Phone Number"
                        defaultValue="8654239581"
                        onBlur={() => {}}
                        onChange={() => {}}
                      />
                    </Stack>
                  </Stack>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Stack spacing={1.25}>
                    <InputLabel htmlFor="personal-email">Email Address</InputLabel>
                    <TextField type="email" fullWidth defaultValue="stebin.ben@gmail.com" id="personal-email" placeholder="Email Address" />
                  </Stack>
                </Grid>
                <Grid item xs={12}>
                  <Stack spacing={1.25}>
                    <InputLabel htmlFor="personal-email">Portfolio URL</InputLabel>
                    <TextField fullWidth defaultValue="https://anshan.dh.url" id="personal-url" placeholder="Portfolio URL" />
                  </Stack>
                </Grid>
                <Grid item xs={12}>
                  <Stack spacing={1.25}>
                    <InputLabel htmlFor="personal-address">Address</InputLabel>
                    <TextField
                      fullWidth
                      defaultValue="Street 110-B Kalians Bag, Dewan, M.P. New York"
                      id="personal-address"
                      placeholder="Address"
                    />
                  </Stack>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
        </Grid>
      </Grid>
      <Grid item xs={12}>
        <Stack direction="row" justifyContent="flex-end" alignItems="center" spacing={2}>
          <Button variant="outlined" color="secondary">
            Cancel
          </Button>
          <Button variant="contained">Update Profile</Button>
        </Stack>
      </Grid>
    </Grid>
  );
};

export default TabPersonal;
