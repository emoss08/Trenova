import { useState } from 'react';

// material-ui
import {
  Box,
  Button,
  Chip,
  Divider,
  Fade,
  Grid,
  Link,
  List,
  ListItem,
  ListItemAvatar,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Stack,
  Typography
} from '@mui/material';

// third-party
import NumberFormat from 'react-number-format';

// project import
import MainCard from 'components/MainCard';
import Avatar from 'components/@extended/Avatar';
import IconButton from 'components/@extended/IconButton';

// assets
import { EnvironmentOutlined, LinkOutlined, MailOutlined, MoreOutlined, PhoneOutlined } from '@ant-design/icons';

// types
import { UserCardProps } from 'types/user-profile';
import UserDialog from './UserDialog';

const avatarImage = require.context('assets/images/users', true);

// ==============================|| PROFILE - USER CARD ||============================== //

const UserCard = ({ user }: { user: UserCardProps }) => {
  const [open, setOpen] = useState(false);

  const handleClickOpen = () => {
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
  };

  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const openMenu = Boolean(anchorEl);
  const handleMenuClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };
  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  return (
    <>
      <MainCard sx={{ height: 1, '& .MuiCardContent-root': { height: 1, display: 'flex', flexDirection: 'column' } }}>
        <Grid container spacing={2.25}>
          <Grid item xs={12}>
            <List sx={{ width: 1, p: 0 }}>
              <ListItem
                disablePadding
                secondaryAction={
                  <IconButton edge="end" aria-label="comments" color="secondary" onClick={handleMenuClick}>
                    <MoreOutlined style={{ fontSize: '1.15rem' }} />
                  </IconButton>
                }
              >
                <ListItemAvatar>
                  <Avatar alt={user.fatherName} src={avatarImage(`./avatar-${!user.avatar ? 1 : user.avatar}.png`)} />
                </ListItemAvatar>
                <ListItemText
                  primary={<Typography variant="subtitle1">{user.fatherName}</Typography>}
                  secondary={
                    <Typography variant="caption" color="secondary">
                      {user.role}
                    </Typography>
                  }
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
              onClose={handleMenuClose}
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
              <MenuItem onClick={handleClickOpen}>Preview</MenuItem>
              <MenuItem onClick={handleMenuClose}>Share</MenuItem>
              <MenuItem onClick={handleMenuClose}>Edit</MenuItem>
              <MenuItem onClick={handleMenuClose}>Delete</MenuItem>
            </Menu>
          </Grid>
          <Grid item xs={12}>
            <Divider />
          </Grid>
          <Grid item xs={12}>
            <Typography>Hello, {user.about}</Typography>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={1}>
              <Grid item xs={6}>
                <List sx={{ p: 0, overflow: 'hidden', '& .MuiListItem-root': { px: 0, py: 0.5 } }}>
                  <ListItem>
                    <ListItemIcon>
                      <MailOutlined />
                    </ListItemIcon>
                    <ListItemText primary={<Typography color="secondary">{user.email}</Typography>} />
                  </ListItem>
                  <ListItem>
                    <ListItemIcon>
                      <PhoneOutlined />
                    </ListItemIcon>
                    <ListItemText
                      primary={
                        <Typography color="secondary">
                          <NumberFormat displayType="text" format="+1 (###) ###-####" mask="_" defaultValue={user.contact} />
                        </Typography>
                      }
                    />
                  </ListItem>
                </List>
              </Grid>
              <Grid item xs={6}>
                <List sx={{ p: 0, overflow: 'hidden', '& .MuiListItem-root': { px: 0, py: 0.5 } }}>
                  <ListItem>
                    <ListItemIcon>
                      <EnvironmentOutlined />
                    </ListItemIcon>
                    <ListItemText primary={<Typography color="secondary">{user.country}</Typography>} />
                  </ListItem>
                  <ListItem>
                    <ListItemIcon>
                      <LinkOutlined />
                    </ListItemIcon>
                    <ListItemText
                      primary={
                        <Link href="https://google.com" target="_blank" sx={{ textTransform: 'lowercase' }}>
                          https://{user.firstName}.en
                        </Link>
                      }
                    />
                  </ListItem>
                </List>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Box>
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
            </Box>
          </Grid>
        </Grid>
        <Stack direction="row" alignItems="center" spacing={1} justifyContent="space-between" sx={{ mt: 'auto', mb: 0, pt: 2.25 }}>
          <Typography variant="caption" color="secondary">
            Updated in {user.time}
          </Typography>
          <Button variant="outlined" size="small" onClick={handleClickOpen}>
            Preview
          </Button>
        </Stack>
      </MainCard>
      {open && <UserDialog user={user} open={open} onClose={handleClose} />}
    </>
  );
};

export default UserCard;
