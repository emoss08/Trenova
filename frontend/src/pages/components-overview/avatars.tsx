import { useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { AvatarGroup, Badge, Box, Divider, Grid, Stack, Tooltip, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ComponentHeader from 'components/cards/ComponentHeader';
import Avatar from 'components/@extended/Avatar';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';

// assets
import {
  CheckOutlined,
  DatabaseFilled,
  DeleteFilled,
  InfoCircleFilled,
  PlusOutlined,
  UserOutlined,
  WarningFilled,
  ZoomInOutlined,
  ZoomOutOutlined
} from '@ant-design/icons';

const avatarImage = require.context('assets/images/users', true);

// ==============================|| COMPONENTS - AVATAR ||============================== //

const ComponentAvatar = () => {
  const theme = useTheme();

  const [open, setOpen] = useState<boolean>(false);
  const [show, setShow] = useState<boolean>(false);
  // const handleOpen = () => {
  //   setOpen(!open);
  // };

  const basicAvatarCodeString = `<Avatar alt="Basic"><UserOutlined /></Avatar>`;

  const imageAvatarCodeString = `<Avatar alt="Avatar 1" src={avatarImage('./avatar-1.png')} />
<Avatar alt="Avatar 2" src={avatarImage('./avatar-2.png')} />
<Avatar alt="Avatar 3" src={avatarImage('./avatar-3.png')} />
<Avatar alt="Avatar 4" src={avatarImage('./avatar-4.png')} />`;

  const vectorAvatarCodeString = `<Avatar><img alt="Natacha" src={avatarImage('./vector-1.png')} height={40} /></Avatar>
<Avatar><img alt="Natacha" src={avatarImage('./vector-2.png')} height={40} /></Avatar>
<Avatar><img alt="Natacha" src={avatarImage('./vector-3.png')} height={40} /></Avatar>
<Avatar><img alt="Natacha" src={avatarImage('./vector-4.png')} height={40} /></Avatar>`;

  const letterAvatarCodeString = `<Avatar alt="Natacha" size="sm">U</Avatar>
<Avatar color="error" alt="Natacha" size="sm">UI</Avatar>
<Avatar color="error" type="filled" alt="Natacha" size="sm">A</Avatar>`;

  const variantsAvatarCodeString = `<Avatar alt="Natacha"><UserOutlined /></Avatar>
<Avatar alt="Natacha" variant="rounded" type="combined"><UserOutlined /></Avatar>
<Avatar alt="Natacha" variant="square" type="filled"><UserOutlined /></Avatar>
<Avatar alt="Natacha">U</Avatar>
<Avatar alt="Natacha" variant="rounded" type="combined">U</Avatar>
<Avatar alt="Natacha" variant="square" type="filled">U</Avatar>`;

  const outlinedAvatarCodeString = `<Avatar alt="Natacha" type="outlined"><UserOutlined /></Avatar>
<Avatar alt="Natacha" variant="rounded" type="outlined"><UserOutlined /></Avatar>
<Avatar alt="Natacha" variant="square" type="outlined"><UserOutlined /></Avatar>
<Avatar alt="Natacha" type="outlined">U</Avatar>
<Avatar alt="Natacha" variant="rounded" type="outlined">U</Avatar>
<Avatar alt="Natacha" variant="square" type="outlined">U</Avatar>`;

  const iconAvatarCodeString = `<Avatar alt="Natacha" size="sm" type="filled"><UserOutlined /></Avatar>
<Avatar alt="Natacha" size="sm" type="filled" color="success"><ZoomInOutlined /></Avatar>
<Avatar alt="Natacha" size="sm" type="filled" color="error"><ZoomOutOutlined /></Avatar>
<Avatar alt="Natacha" size="sm"><PlusOutlined /></Avatar>`;

  const groupAvatarCodeString = `<AvatarGroup max={4}>
  <Avatar alt="Trevor Henderson" src={avatarImage('./avatar-5.png')} />
  <Avatar alt="Jone Doe" src={avatarImage('./avatar-6.png')} />
  <Avatar alt="Lein Ket" src={avatarImage('./avatar-7.png')} />
  <Avatar alt="Stebin Ben" src={avatarImage('./avatar-8.png')} />
  <Avatar alt="Wungh Tend" src={avatarImage('./avatar-9.png')} />
  <Avatar alt="Trevor Das" src={avatarImage('./avatar-10.png')} />
</AvatarGroup>
<Box sx={{ width: 186 }}>
  <Tooltip
    open={show}
    placement="top-end"
    title={
      <AvatarGroup max={10}>
        <Avatar alt="Trevor Henderson" src={avatarImage('./avatar-5.png')} />
        <Avatar alt="Jone Doe" src={avatarImage('./avatar-6.png')} />
        <Avatar alt="Lein Ket" src={avatarImage('./avatar-7.png')} />
        <Avatar alt="Stebin Ben" src={avatarImage('./avatar-8.png')} />
        <Avatar alt="Wungh Tend" src={avatarImage('./avatar-9.png')} />
        <Avatar alt="Trevor Das" src={avatarImage('./avatar-10.png')} />
      </AvatarGroup>
    }
  >
    <AvatarGroup
      sx={{ '& .MuiAvatarGroup-avatar': { bgcolor: theme.palette.primary.main, cursor: 'pointer' } }}
      componentsProps={{
        additionalAvatar: {
          onMouseEnter: () => {
            setShow(true);
          },
          onMouseLeave: () => {
            setShow(false);
          }
        }
      }}
    >
      <Avatar alt="Remy Sharp" src={avatarImage('./avatar-1.png')} />
      <Avatar alt="Travis Howard" src={avatarImage('./avatar-2.png')} />
      <Avatar alt="Cindy Baker" src={avatarImage('./avatar-3.png')} />
      <Avatar alt="Agnes Walker" src={avatarImage('./avatar-4.png')} />
      <Avatar alt="Trevor Henderson" src={avatarImage('./avatar-5.png')} />
      <Avatar alt="Jone Doe" src={avatarImage('./avatar-6.png')} />
      <Avatar alt="Lein Ket" src={avatarImage('./avatar-7.png')} />
      <Avatar alt="Stebin Ben" src={avatarImage('./avatar-8.png')} />
      <Avatar alt="Wungh Tend" src={avatarImage('./avatar-9.png')} />
      <Avatar alt="Trevor Das" src={avatarImage('./avatar-10.png')} />
    </AvatarGroup>
  </Tooltip>
</Box>
<Box sx={{ width: 222 }}>
  <Tooltip
    open={open}
    placement="top-end"
    title={
      <AvatarGroup max={10}>
        <Avatar alt="Jone Doe" src={avatarImage('./avatar-6.png')} />
        <Avatar alt="Lein Ket" src={avatarImage('./avatar-7.png')} />
        <Avatar alt="Stebin Ben" src={avatarImage('./avatar-8.png')} />
        <Avatar alt="Wungh Tend" src={avatarImage('./avatar-9.png')} />
        <Avatar alt="Trevor Das" src={avatarImage('./avatar-10.png')} />
      </AvatarGroup>
    }
  >
    <AvatarGroup
      max={6}
      sx={{ '& .MuiAvatarGroup-avatar': { bgcolor: theme.palette.primary.main, cursor: 'pointer' } }}
      componentsProps={{
        additionalAvatar: {
          onClick: () => {
            setOpen(!open);
          }
        }
      }}
    >
      <Avatar alt="Remy Sharp" src={avatarImage('./avatar-1.png')} />
      <Avatar alt="Travis Howard" src={avatarImage('./avatar-2.png')} />
      <Avatar alt="Cindy Baker" src={avatarImage('./avatar-3.png')} />
      <Avatar alt="Agnes Walker" src={avatarImage('./avatar-4.png')} />
      <Avatar alt="Trevor Henderson" src={avatarImage('./avatar-5.png')} />
      <Avatar alt="Jone Doe" src={avatarImage('./avatar-6.png')} />
      <Avatar alt="Lein Ket" src={avatarImage('./avatar-7.png')} />
      <Avatar alt="Stebin Ben" src={avatarImage('./avatar-8.png')} />
      <Avatar alt="Wungh Tend" src={avatarImage('./avatar-9.png')} />
      <Avatar alt="Trevor Das" src={avatarImage('./avatar-10.png')} />
    </AvatarGroup>
  </Tooltip>
</Box>`;

  const badgeAvatarCodeString = `<Badge badgeContent={4} color="error" overlap="circular">
  <Avatar alt="Natacha" type="filled" src={avatarImage('./avatar-6.png')} />
</Badge>
<Badge color="error" overlap="circular" variant="dot">
  <Avatar alt="Natacha" color="secondary" type="filled">
    <UserOutlined />
  </Avatar>
</Badge>
<Badge color="error" overlap="circular" variant="dot">
  <Avatar alt="Natacha" type="filled" src={avatarImage('./avatar-2.png')} />
</Badge>
<Badge color="error" overlap="circular" variant="dot">
  <Avatar alt="Natacha" type="outlined">
    U
  </Avatar>
</Badge>
<Badge color="error" overlap="circular" variant="dot">
  <Avatar>
    <img alt="Natacha" src={avatarImage('./vector-2.png')} width={40} />
  </Avatar>
</Badge>
<Badge color="success" variant="dot">
  <Avatar alt="Natacha" variant="rounded" type="filled" src={avatarImage('./avatar-1.png')} />
</Badge>
<Badge
  overlap="circular"
  anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
  badgeContent={<Avatar size="badge" alt="Remy Sharp" src={avatarImage('./avatar-6.png')} />}
>
  <Avatar alt="Travis Howard" src={avatarImage('./avatar-1.png')} />
</Badge>`;

  const sizesAvatarCodeString = `<Avatar size="xs" alt="Avatar 1" src={avatarImage('./avatar-1.png')} />
<Avatar size="xl" alt="Avatar 5" src={avatarImage('./avatar-5.png')} />
<Avatar size="lg" alt="Avatar 4" src={avatarImage('./avatar-4.png')} />
<Avatar size="md" alt="Avatar 3" src={avatarImage('./avatar-3.png')} />
<Avatar size="sm" alt="Avatar 2" src={avatarImage('./avatar-2.png')} />`;

  const colorsAvatarCodeString = `<Avatar alt="Basic" type="filled"><UserOutlined /></Avatar>
<Avatar alt="Basic" type="filled" color="error"><DeleteFilled /></Avatar>
<Avatar alt="Basic" type="filled" color="info"><InfoCircleFilled /></Avatar>
<Avatar alt="Basic" type="filled" color="warning"><WarningFilled /></Avatar>
<Avatar alt="Basic" type="filled" color="success"><CheckOutlined /></Avatar>
<Avatar alt="Basic" type="filled" color="secondary"><DatabaseFilled /></Avatar>`;

  const fallbacksAvatarCodeString = `<Avatar alt="Remy Sharp" src="/broken-image.jpg" color="error" type="filled">B</Avatar>
<Avatar src="/broken-image.jpg" color="error" />
<Avatar alt="Remy Sharp" src="/broken-image.jpg" color="error" type="outlined" />`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Avatar"
        caption="Avatars are found throughout material design with uses in everything from tables to dialog menus."
        directory="src/pages/components-overview/avatars"
        link="https://mui.com/material-ui/react-avatar/"
      />
      <ComponentWrapper>
        <Grid container spacing={3}>
          <Grid item xs={12} lg={6}>
            <Stack spacing={3}>
              <MainCard title="Basic" codeHighlight codeString={basicAvatarCodeString}>
                <Avatar alt="Basic">
                  <UserOutlined />
                </Avatar>
              </MainCard>
              <MainCard title="Vector" codeString={vectorAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Avatar>
                      <img alt="Natacha" src={avatarImage(`./vector-1.png`)} height={40} />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar>
                      <img alt="Natacha" src={avatarImage(`./vector-2.png`)} height={40} />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar>
                      <img alt="Natacha" src={avatarImage(`./vector-3.png`)} height={40} />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar>
                      <img alt="Natacha" src={avatarImage(`./vector-4.png`)} height={40} />
                    </Avatar>
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Variants" codeString={variantsAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Avatar alt="Natacha">
                      <UserOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" variant="rounded" type="combined">
                      <UserOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" variant="square" type="filled">
                      <UserOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha">U</Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" variant="rounded" type="combined">
                      U
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" variant="square" type="filled">
                      U
                    </Avatar>
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Icon" codeString={iconAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Avatar alt="Natacha" size="sm" type="filled">
                      <UserOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" size="sm" type="filled" color="success">
                      <ZoomInOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" size="sm" type="filled" color="error">
                      <ZoomOutOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" size="sm">
                      <PlusOutlined />
                    </Avatar>
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="With Badge" codeString={badgeAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Badge badgeContent={4} color="error" overlap="circular">
                      <Avatar alt="Natacha" type="filled" src={avatarImage(`./avatar-6.png`)} />
                    </Badge>
                  </Grid>
                  <Grid item>
                    <Badge color="error" overlap="circular" variant="dot">
                      <Avatar alt="Natacha" color="secondary" type="filled">
                        <UserOutlined />
                      </Avatar>
                    </Badge>
                  </Grid>
                  <Grid item>
                    <Badge color="error" overlap="circular" variant="dot">
                      <Avatar alt="Natacha" type="filled" src={avatarImage(`./avatar-2.png`)} />
                    </Badge>
                  </Grid>
                  <Grid item>
                    <Badge color="error" overlap="circular" variant="dot">
                      <Avatar alt="Natacha" type="outlined">
                        U
                      </Avatar>
                    </Badge>
                  </Grid>
                  <Grid item>
                    <Badge color="error" overlap="circular" variant="dot">
                      <Avatar>
                        <img alt="Natacha" src={avatarImage(`./vector-2.png`)} width={40} />
                      </Avatar>
                    </Badge>
                  </Grid>
                  <Grid item>
                    <Badge color="success" variant="dot">
                      <Avatar alt="Natacha" variant="rounded" type="filled" src={avatarImage(`./avatar-1.png`)} />
                    </Badge>
                  </Grid>
                  <Grid item>
                    <Badge
                      overlap="circular"
                      anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                      badgeContent={<Avatar size="badge" alt="Remy Sharp" src={avatarImage(`./avatar-6.png`)} />}
                    >
                      <Avatar alt="Travis Howard" src={avatarImage(`./avatar-1.png`)} />
                    </Badge>
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Image" codeString={imageAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Avatar alt="Avatar 1" src={avatarImage(`./avatar-1.png`)} />
                  </Grid>
                  <Grid item>
                    <Avatar alt="Avatar 2" src={avatarImage(`./avatar-2.png`)} />
                  </Grid>
                  <Grid item>
                    <Avatar alt="Avatar 3" src={avatarImage(`./avatar-3.png`)} />
                  </Grid>
                  <Grid item>
                    <Avatar alt="Avatar 4" src={avatarImage(`./avatar-4.png`)} />
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Colors" codeString={colorsAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Avatar alt="Basic" type="filled">
                      <UserOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Basic" type="filled" color="secondary">
                      <DatabaseFilled />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Basic" type="filled" color="success">
                      <CheckOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Basic" type="filled" color="warning">
                      <WarningFilled />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Basic" type="filled" color="info">
                      <InfoCircleFilled />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Basic" type="filled" color="error">
                      <DeleteFilled />
                    </Avatar>
                  </Grid>
                </Grid>
              </MainCard>
            </Stack>
          </Grid>
          <Grid item xs={12} lg={6}>
            <Stack spacing={3}>
              <MainCard title="Letter" codeString={letterAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Avatar alt="Natacha" size="sm">
                      U
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar color="error" alt="Natacha" size="sm">
                      UI
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar color="error" type="filled" alt="Natacha" size="sm">
                      A
                    </Avatar>
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Outlined" codeString={outlinedAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Avatar alt="Natacha" type="outlined">
                      <UserOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" variant="rounded" type="outlined">
                      <UserOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" variant="square" type="outlined">
                      <UserOutlined />
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" type="outlined">
                      U
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" variant="rounded" type="outlined">
                      U
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Natacha" variant="square" type="outlined">
                      U
                    </Avatar>
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Avatar Group" codeString={groupAvatarCodeString}>
                <Stack spacing={2}>
                  <Typography variant="subtitle1">Default</Typography>
                  <Box sx={{ width: 148 }}>
                    <AvatarGroup max={4}>
                      <Avatar alt="Trevor Henderson" src={avatarImage(`./avatar-5.png`)} />
                      <Avatar alt="Jone Doe" src={avatarImage(`./avatar-6.png`)} />
                      <Avatar alt="Lein Ket" src={avatarImage(`./avatar-7.png`)} />
                      <Avatar alt="Stebin Ben" src={avatarImage(`./avatar-8.png`)} />
                      <Avatar alt="Wungh Tend" src={avatarImage(`./avatar-9.png`)} />
                      <Avatar alt="Trevor Das" src={avatarImage(`./avatar-10.png`)} />
                    </AvatarGroup>
                  </Box>
                  <Divider sx={{ my: 2 }} />
                  <Typography variant="subtitle1">On Hover</Typography>
                  <Box sx={{ width: 186 }}>
                    <Tooltip
                      open={show}
                      placement="top-end"
                      title={
                        <AvatarGroup max={10}>
                          <Avatar alt="Trevor Henderson" src={avatarImage(`./avatar-5.png`)} />
                          <Avatar alt="Jone Doe" src={avatarImage(`./avatar-6.png`)} />
                          <Avatar alt="Lein Ket" src={avatarImage(`./avatar-7.png`)} />
                          <Avatar alt="Stebin Ben" src={avatarImage(`./avatar-8.png`)} />
                          <Avatar alt="Wungh Tend" src={avatarImage(`./avatar-9.png`)} />
                          <Avatar alt="Trevor Das" src={avatarImage(`./avatar-10.png`)} />
                        </AvatarGroup>
                      }
                    >
                      <AvatarGroup
                        sx={{ '& .MuiAvatarGroup-avatar': { bgcolor: theme.palette.primary.main, cursor: 'pointer' } }}
                        componentsProps={{
                          additionalAvatar: {
                            onMouseEnter: () => {
                              setShow(true);
                            },
                            onMouseLeave: () => {
                              setShow(false);
                            }
                          }
                        }}
                      >
                        <Avatar alt="Remy Sharp" src={avatarImage(`./avatar-1.png`)} />
                        <Avatar alt="Travis Howard" src={avatarImage(`./avatar-2.png`)} />
                        <Avatar alt="Cindy Baker" src={avatarImage(`./avatar-3.png`)} />
                        <Avatar alt="Agnes Walker" src={avatarImage(`./avatar-4.png`)} />
                        <Avatar alt="Trevor Henderson" src={avatarImage(`./avatar-5.png`)} />
                        <Avatar alt="Jone Doe" src={avatarImage(`./avatar-6.png`)} />
                        <Avatar alt="Lein Ket" src={avatarImage(`./avatar-7.png`)} />
                        <Avatar alt="Stebin Ben" src={avatarImage(`./avatar-8.png`)} />
                        <Avatar alt="Wungh Tend" src={avatarImage(`./avatar-9.png`)} />
                        <Avatar alt="Trevor Das" src={avatarImage(`./avatar-10.png`)} />
                      </AvatarGroup>
                    </Tooltip>
                  </Box>
                </Stack>
                <Divider sx={{ my: 2 }} />
                <Stack spacing={2}>
                  <Typography variant="subtitle1">On Click</Typography>
                  <Box sx={{ width: 222 }}>
                    <Tooltip
                      open={open}
                      placement="top-end"
                      title={
                        <AvatarGroup max={10}>
                          <Avatar alt="Jone Doe" src={avatarImage(`./avatar-6.png`)} />
                          <Avatar alt="Lein Ket" src={avatarImage(`./avatar-7.png`)} />
                          <Avatar alt="Stebin Ben" src={avatarImage(`./avatar-8.png`)} />
                          <Avatar alt="Wungh Tend" src={avatarImage(`./avatar-9.png`)} />
                          <Avatar alt="Trevor Das" src={avatarImage(`./avatar-10.png`)} />
                        </AvatarGroup>
                      }
                    >
                      <AvatarGroup
                        max={6}
                        sx={{ '& .MuiAvatarGroup-avatar': { bgcolor: theme.palette.primary.main, cursor: 'pointer' } }}
                        componentsProps={{
                          additionalAvatar: {
                            onClick: () => {
                              setOpen(!open);
                            }
                          }
                        }}
                      >
                        <Avatar alt="Remy Sharp" src={avatarImage(`./avatar-1.png`)} />
                        <Avatar alt="Travis Howard" src={avatarImage(`./avatar-2.png`)} />
                        <Avatar alt="Cindy Baker" src={avatarImage(`./avatar-3.png`)} />
                        <Avatar alt="Agnes Walker" src={avatarImage(`./avatar-4.png`)} />
                        <Avatar alt="Trevor Henderson" src={avatarImage(`./avatar-5.png`)} />
                        <Avatar alt="Jone Doe" src={avatarImage(`./avatar-6.png`)} />
                        <Avatar alt="Lein Ket" src={avatarImage(`./avatar-7.png`)} />
                        <Avatar alt="Stebin Ben" src={avatarImage(`./avatar-8.png`)} />
                        <Avatar alt="Wungh Tend" src={avatarImage(`./avatar-9.png`)} />
                        <Avatar alt="Trevor Das" src={avatarImage(`./avatar-10.png`)} />
                      </AvatarGroup>
                    </Tooltip>
                  </Box>
                </Stack>
              </MainCard>
              <MainCard title="Sizes" codeString={sizesAvatarCodeString}>
                <Grid container spacing={1} alignItems="center">
                  <Grid item>
                    <Avatar size="xs" alt="Avatar 1" src={avatarImage(`./avatar-1.png`)} />
                  </Grid>
                  <Grid item>
                    <Avatar size="sm" alt="Avatar 2" src={avatarImage(`./avatar-2.png`)} />
                  </Grid>
                  <Grid item>
                    <Avatar size="md" alt="Avatar 3" src={avatarImage(`./avatar-3.png`)} />
                  </Grid>
                  <Grid item>
                    <Avatar size="lg" alt="Avatar 4" src={avatarImage(`./avatar-4.png`)} />
                  </Grid>
                  <Grid item>
                    <Avatar size="xl" alt="Avatar 5" src={avatarImage(`./avatar-5.png`)} />
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Fallbacks" codeString={fallbacksAvatarCodeString}>
                <Grid container spacing={1}>
                  <Grid item>
                    <Avatar alt="Remy Sharp" src="/broken-image.jpg" color="error" type="filled">
                      B
                    </Avatar>
                  </Grid>
                  <Grid item>
                    <Avatar alt="Remy Sharp" src="/broken-image.jpg" color="error" type="outlined" />
                  </Grid>
                  <Grid item>
                    <Avatar src="/broken-image.jpg" color="error" />
                  </Grid>
                </Grid>
              </MainCard>
            </Stack>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentAvatar;
