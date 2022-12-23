import { forwardRef, useState, Ref } from 'react';

// material-ui
import { Theme } from '@mui/material/styles';
import {
  useMediaQuery,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Fade,
  Grid,
  Link,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  Menu,
  MenuItem,
  Stack,
  Typography
} from '@mui/material';
import { TransitionProps } from '@mui/material/transitions';

// third-party
import NumberFormat from 'react-number-format';

// project import
import MainCard from 'components/MainCard';
import Avatar from 'components/@extended/Avatar';
import IconButton from 'components/@extended/IconButton';
import SimpleBar from 'components/third-party/SimpleBar';

// types
import { UserCardProps } from 'types/user-profile';

// assets
import { MoreOutlined } from '@ant-design/icons';

const avatarImage = require.context('assets/images/users', true);

const Transition = forwardRef((props: TransitionProps & { children: React.ReactElement<any, any> }, ref: Ref<unknown>) => (
  <Fade ref={ref} {...props} />
));

// ==============================|| DIALOG - USER CARD ||============================== //

export default function UserDialog({ user, open, onClose }: { user: UserCardProps; open: boolean; onClose: () => void }) {
  const matchDownMD = useMediaQuery((theme: Theme) => theme.breakpoints.down('md'));

  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const openMenu = Boolean(anchorEl);
  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };
  const handleClose = () => {
    setAnchorEl(null);
  };

  return (
    <Dialog
      open={open}
      TransitionComponent={Transition}
      keepMounted
      onClose={onClose}
      aria-describedby="alert-dialog-slide-description"
      sx={{ '& .MuiDialog-paper': { width: 1024, maxWidth: 1, m: { xs: 1.75, sm: 2.5, md: 4 } } }}
    >
      <Box sx={{ px: { xs: 2, sm: 3, md: 5 }, py: 1 }}>
        <DialogTitle sx={{ px: 0 }}>
          <List sx={{ width: 1, p: 0 }}>
            <ListItem
              disablePadding
              secondaryAction={
                <IconButton edge="end" aria-label="comments" color="secondary" onClick={handleClick}>
                  <MoreOutlined style={{ fontSize: '1.15rem' }} />
                </IconButton>
              }
            >
              <ListItemAvatar sx={{ mr: 0.75 }}>
                <Avatar alt={user.fatherName} size="lg" src={avatarImage(`./avatar-${!user.avatar ? 1 : user.avatar}.png`)} />
              </ListItemAvatar>
              <ListItemText
                primary={<Typography variant="h5">{user.fatherName}</Typography>}
                secondary={<Typography color="secondary">{user.role}</Typography>}
              />
            </ListItem>
          </List>
          <Menu
            id="fade-menu"
            MenuListProps={{
              'aria-labelledby': 'fade-button'
            }}
            anchorEl={anchorEl}
            open={openMenu}
            onClose={handleClose}
            TransitionComponent={Fade}
            anchorOrigin={{
              vertical: 'bottom',
              horizontal: 'right'
            }}
            transformOrigin={{
              vertical: 'top',
              horizontal: 'right'
            }}
          >
            <MenuItem onClick={handleClose}>Share</MenuItem>
            <MenuItem onClick={handleClose}>Edit</MenuItem>
            <MenuItem onClick={handleClose}>Delete</MenuItem>
          </Menu>
        </DialogTitle>

        <DialogContent dividers sx={{ px: 0 }}>
          <SimpleBar sx={{ height: 'calc(100vh - 290px)' }}>
            <Grid container spacing={3}>
              <Grid item xs={12} sm={8} xl={9}>
                <Grid container spacing={2.25}>
                  <Grid item xs={12}>
                    <MainCard title="About me">
                      <Typography>
                        Hello, Myself {user.fatherName}, I’m {user.role} in international company, {user.about}
                      </Typography>
                    </MainCard>
                  </Grid>
                  <Grid item xs={12}>
                    <MainCard title="Education">
                      <List sx={{ py: 0 }}>
                        <ListItem divider>
                          <Grid container spacing={matchDownMD ? 0.5 : 3}>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Master Degree (Year)</Typography>
                                <Typography>2014-2017</Typography>
                              </Stack>
                            </Grid>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Institute</Typography>
                                <Typography>-</Typography>
                              </Stack>
                            </Grid>
                          </Grid>
                        </ListItem>
                        <ListItem divider>
                          <Grid container spacing={matchDownMD ? 0.5 : 3}>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Bachelor (Year)</Typography>
                                <Typography>2011-2013</Typography>
                              </Stack>
                            </Grid>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Institute</Typography>
                                <Typography>Imperial College London</Typography>
                              </Stack>
                            </Grid>
                          </Grid>
                        </ListItem>
                        <ListItem>
                          <Grid container spacing={matchDownMD ? 0.5 : 3}>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">School (Year)</Typography>
                                <Typography>2009-2011</Typography>
                              </Stack>
                            </Grid>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Institute</Typography>
                                <Typography>School of London, England</Typography>
                              </Stack>
                            </Grid>
                          </Grid>
                        </ListItem>
                      </List>
                    </MainCard>
                  </Grid>
                  <Grid item xs={12}>
                    <MainCard title="Emplyment">
                      <List sx={{ py: 0 }}>
                        <ListItem divider>
                          <Grid container spacing={matchDownMD ? 0.5 : 3}>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Senior UI/UX designer (Year)</Typography>
                                <Typography>2019-Current</Typography>
                              </Stack>
                            </Grid>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Job Responsibility</Typography>
                                <Typography>
                                  Perform task related to project manager with the 100+ team under my observation. Team management is key
                                  role in this company.
                                </Typography>
                              </Stack>
                            </Grid>
                          </Grid>
                        </ListItem>
                        <ListItem>
                          <Grid container spacing={matchDownMD ? 0.5 : 3}>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Trainee cum Project Manager (Year)</Typography>
                                <Typography>2017-2019</Typography>
                              </Stack>
                            </Grid>
                            <Grid item xs={12} md={6}>
                              <Stack spacing={0.5}>
                                <Typography color="secondary">Job Responsibility</Typography>
                                <Typography>Team management is key role in this company.</Typography>
                              </Stack>
                            </Grid>
                          </Grid>
                        </ListItem>
                      </List>
                    </MainCard>
                  </Grid>
                  <Grid item xs={12}>
                    <MainCard title="Skills">
                      <Box
                        sx={{
                          display: 'flex',
                          flexWrap: 'wrap',
                          listStyle: 'none',
                          p: 0.5,
                          m: 0
                        }}
                        component="ul"
                      >
                        {user.skills.map((skill: string, index: number) => (
                          <ListItem disablePadding key={index} sx={{ width: 'auto', pr: 0.75, pb: 0.75 }}>
                            <Chip color="secondary" variant="outlined" size="small" label={skill} />
                          </ListItem>
                        ))}
                      </Box>
                    </MainCard>
                  </Grid>
                </Grid>
              </Grid>
              <Grid item xs={12} sm={4} xl={3}>
                <MainCard>
                  <Stack spacing={2}>
                    <Stack spacing={0.5}>
                      <Typography color="secondary">Father Name</Typography>
                      <Typography>
                        Mr. {user.firstName} {user.lastName}
                      </Typography>
                    </Stack>
                    <Stack spacing={0.5}>
                      <Typography color="secondary">Email</Typography>
                      <Typography>{user.email}</Typography>
                    </Stack>
                    <Stack spacing={0.5}>
                      <Typography color="secondary">Contact</Typography>
                      <Typography>
                        <NumberFormat displayType="text" format="+1 (###) ###-####" mask="_" defaultValue={user.contact} />
                      </Typography>
                    </Stack>
                    <Stack spacing={0.5}>
                      <Typography color="secondary">Location</Typography>
                      <Typography> {user.country} </Typography>
                    </Stack>
                    <Stack spacing={0.5}>
                      <Typography color="secondary">Website</Typography>
                      <Link href="https://google.com" target="_blank" sx={{ textTransform: 'lowercase' }}>
                        https://{user.firstName}.en
                      </Link>
                    </Stack>
                  </Stack>
                </MainCard>
              </Grid>
            </Grid>
          </SimpleBar>
        </DialogContent>

        <DialogActions>
          <Button color="error" onClick={onClose}>
            Close
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  );
}
