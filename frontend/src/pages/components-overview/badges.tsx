import { useState } from 'react';

// material-ui
import { Badge, Button, ButtonGroup, FormControlLabel, Grid, Stack, Switch, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ComponentHeader from 'components/cards/ComponentHeader';
import AntAvatar from 'components/@extended/Avatar';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';

// assets
import { MailOutlined, MinusOutlined, PlusOutlined, UserOutlined } from '@ant-design/icons';

// ==============================|| COMPONENTS - BADGES ||============================== //

const ComponentBadge = () => {
  const [count, setCount] = useState(1);
  const [invisible, setInvisible] = useState(false);

  const handleBadgeVisibility = () => {
    setInvisible(!invisible);
  };

  const basicBadgesCodeString = `<Badge badgeContent={4} color="primary"><MailOutlined /></Badge>
<Badge badgeContent={4} color="secondary"><MailOutlined /></Badge>
<Badge badgeContent={4} color="success"><MailOutlined /></Badge>
<Badge badgeContent={4} color="warning"><MailOutlined /></Badge>
<Badge badgeContent={4} color="info"><MailOutlined /></Badge>
<Badge badgeContent={4} color="error"><MailOutlined /></Badge>`;

  const lightBadgesCodeString = `<Badge badgeContent={4} color="primary" variant="light"><MailOutlined /></Badge>
<Badge badgeContent={4} color="secondary" variant="light"><MailOutlined /></Badge>
<Badge badgeContent={4} color="success" variant="light"><MailOutlined /></Badge>
<Badge badgeContent={4} color="warning" variant="light"><MailOutlined /></Badge>
<Badge badgeContent={4} color="info" variant="light"><MailOutlined /></Badge>
<Badge badgeContent={4} color="error" variant="light"><MailOutlined /></Badge>`;

  const maxBadgesCodeString = `<Badge badgeContent={99} color="primary"><MailOutlined /></Badge>
<Badge badgeContent={100} color="secondary"><MailOutlined /></Badge>
<Badge badgeContent={1000} max={999} color="primary" variant="light"><MailOutlined /></Badge>
<Badge badgeContent={99} color="secondary" variant="light"><MailOutlined /></Badge>
<Badge badgeContent={99} color="error"><MailOutlined /></Badge>`;

  const dotBadgesCodeString = `<Badge color="primary" variant="dot"><MailOutlined /></Badge>
<Badge color="secondary" variant="dot"><MailOutlined /></Badge>
<Badge max={999} color="success" variant="dot"><MailOutlined /></Badge>
<Badge color="warning" variant="dot"><MailOutlined /></Badge>
<Badge color="info" variant="dot"><MailOutlined /></Badge>
<Badge color="error" variant="dot"><Typography variant="h6">Typography</Typography></Badge>`;

  const alignmentBadgesCodeString = `<Badge badgeContent={9} color="primary">
  <MailOutlined />
</Badge>
<Badge color="primary" variant="dot">
  <MailOutlined />
</Badge>
<Badge
  badgeContent={9}
  color="primary"
  anchorOrigin={{
    vertical: 'bottom',
    horizontal: 'right'
  }}
>
  <MailOutlined />
</Badge>
<Badge
  badgeContent={9}
  color="primary"
  anchorOrigin={{
    vertical: 'top',
    horizontal: 'left'
  }}
>
  <MailOutlined />
</Badge>
<Badge
  badgeContent={99}
  color="primary"
  anchorOrigin={{
    vertical: 'bottom',
    horizontal: 'left'
  }}
>
  <MailOutlined />
</Badge>`;

  const overlapBadgesCodeString = `<Badge color="error" overlap="circular" variant="dot">
  <AntAvatar alt="Basic">
    <UserOutlined />
  </AntAvatar>
</Badge>
<Badge color="error" variant="dot">
  <AntAvatar alt="Basic" variant="rounded" type="filled">
    <UserOutlined />
  </AntAvatar>
</Badge>
<Badge color="error" variant="dot">
  <AntAvatar alt="Basic" variant="square" type="outlined">
    <UserOutlined />
  </AntAvatar>
</Badge>
<Badge badgeContent=" " color="error" overlap="circular">
  <AntAvatar alt="Basic" type="outlined">
    U
  </AntAvatar>
</Badge>
<Badge badgeContent=" " color="error">
  <AntAvatar alt="Basic" variant="rounded" type="filled">
    U
  </AntAvatar>
</Badge>
<Badge badgeContent=" " color="error">
  <AntAvatar alt="Basic" variant="square" type="outlined">
    U
  </AntAvatar>
</Badge>`;

  const visibleBadgesCodeString = `<Badge color="primary" badgeContent={count}><MailOutlined /></Badge>
<ButtonGroup>
  <Button
    aria-label="reduce"
    onClick={() => {
      setCount(Math.max(count - 1, 0));
    }}
  >
    <MinusOutlined />
  </Button>
  <Button
    aria-label="increase"
    onClick={() => {
      setCount(count + 1);
    }}
  >
    <PlusOutlined />
  </Button>
</ButtonGroup>
<Badge color="primary" variant="dot" invisible={invisible}><MailOutlined /></Badge>
<FormControlLabel
  sx={{ color: 'text.primary' }}
  control={<Switch checked={!invisible} onChange={handleBadgeVisibility} />}
  label="Show Badge"
  labelPlacement="start"
/>`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Badge"
        caption="Badge generates a small badge to the top-right of its child(ren)."
        directory="src/pages/components-overview/badges"
        link="https://mui.com/material-ui/react-badge/"
      />
      <ComponentWrapper>
        <Grid container spacing={3}>
          <Grid item xs={12} lg={6}>
            <MainCard title="Basic" codeHighlight codeString={basicBadgesCodeString}>
              <Grid container spacing={3}>
                <Grid item>
                  <Badge badgeContent={4} color="primary">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="secondary">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="success">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="warning">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="info">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="error">
                    <MailOutlined />
                  </Badge>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Light" codeString={lightBadgesCodeString}>
              <Grid container spacing={3}>
                <Grid item>
                  <Badge badgeContent={4} color="primary" variant="light">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="secondary" variant="light">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="success" variant="light">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="warning" variant="light">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="info" variant="light">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={4} color="error" variant="light">
                    <MailOutlined />
                  </Badge>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Maximmum Value" codeString={maxBadgesCodeString}>
              <Grid container spacing={4}>
                <Grid item>
                  <Badge badgeContent={99} color="primary">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={100} color="secondary">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={1000} max={999} color="primary" variant="light">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={99} color="secondary" variant="light">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent={99} color="error">
                    <MailOutlined />
                  </Badge>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Dot Badges" codeString={dotBadgesCodeString}>
              <Grid container spacing={3}>
                <Grid item>
                  <Badge color="primary" variant="dot">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge color="secondary" variant="dot">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge max={999} color="success" variant="dot">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge color="warning" variant="dot">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge color="info" variant="dot">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge color="error" variant="dot">
                    <Typography variant="h6">Typography</Typography>
                  </Badge>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Alignment" codeString={alignmentBadgesCodeString}>
              <Grid container spacing={4}>
                <Grid item>
                  <Badge badgeContent={9} color="primary">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge color="primary" variant="dot">
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge
                    badgeContent={9}
                    color="primary"
                    anchorOrigin={{
                      vertical: 'bottom',
                      horizontal: 'right'
                    }}
                  >
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge
                    badgeContent={9}
                    color="primary"
                    anchorOrigin={{
                      vertical: 'top',
                      horizontal: 'left'
                    }}
                  >
                    <MailOutlined />
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge
                    badgeContent={99}
                    color="primary"
                    anchorOrigin={{
                      vertical: 'bottom',
                      horizontal: 'left'
                    }}
                  >
                    <MailOutlined />
                  </Badge>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Overlap" codeString={overlapBadgesCodeString}>
              <Grid container spacing={2}>
                <Grid item>
                  <Badge color="error" overlap="circular" variant="dot">
                    <AntAvatar alt="Basic">
                      <UserOutlined />
                    </AntAvatar>
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge color="error" variant="dot">
                    <AntAvatar alt="Basic" variant="rounded" type="filled">
                      <UserOutlined />
                    </AntAvatar>
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge color="error" variant="dot">
                    <AntAvatar alt="Basic" variant="square" type="outlined">
                      <UserOutlined />
                    </AntAvatar>
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent=" " color="error" overlap="circular">
                    <AntAvatar alt="Basic" type="outlined">
                      U
                    </AntAvatar>
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent=" " color="error">
                    <AntAvatar alt="Basic" variant="rounded" type="filled">
                      U
                    </AntAvatar>
                  </Badge>
                </Grid>
                <Grid item>
                  <Badge badgeContent=" " color="error">
                    <AntAvatar alt="Basic" variant="square" type="outlined">
                      U
                    </AntAvatar>
                  </Badge>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Visibility" codeString={visibleBadgesCodeString}>
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <Stack direction="row" spacing={3} alignItems="center">
                    <Badge color="primary" badgeContent={count}>
                      <MailOutlined />
                    </Badge>
                    <ButtonGroup>
                      <Button
                        aria-label="reduce"
                        onClick={() => {
                          setCount(Math.max(count - 1, 0));
                        }}
                      >
                        <MinusOutlined />
                      </Button>
                      <Button
                        aria-label="increase"
                        onClick={() => {
                          setCount(count + 1);
                        }}
                      >
                        <PlusOutlined />
                      </Button>
                    </ButtonGroup>
                  </Stack>
                </Grid>
                <Grid item xs={12}>
                  <Stack direction="row" spacing={3} alignItems="center">
                    <Badge color="primary" variant="dot" invisible={invisible}>
                      <MailOutlined />
                    </Badge>
                    <FormControlLabel
                      sx={{ color: 'text.primary' }}
                      control={<Switch checked={!invisible} onChange={handleBadgeVisibility} />}
                      label="Show Badge"
                      labelPlacement="start"
                    />
                  </Stack>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentBadge;
