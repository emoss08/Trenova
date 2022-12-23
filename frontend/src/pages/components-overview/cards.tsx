import { Link as RouterLink } from 'react-router-dom';

// material-ui
import { useTheme } from '@mui/material/styles';
import { CardContent, CardMedia, Divider, Grid, Link, ToggleButton, ToggleButtonGroup, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import IconButton from 'components/@extended/IconButton';
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import CardTabs from 'sections/components-overview/cards/CardTabs';

// assets
import { EditOutlined, EllipsisOutlined, MoreOutlined, SettingOutlined } from '@ant-design/icons';
import media from 'assets/images/component/card-media.png';

// ==============================|| COMPONENTS - CARD ||============================== //

const ComponentCard = () => {
  const theme = useTheme();

  const cardAction = (
    <ToggleButtonGroup
      fullWidth
      color="primary"
      exclusive
      aria-label="text alignment"
      size="small"
      sx={{
        p: 1,
        '& .MuiToggleButton-root': {
          borderRadius: 0,
          p: 0.75,
          '&:not(.Mui-selected)': {
            borderTopColor: 'transparent',
            borderBottomColor: 'transparent'
          },
          '&:first-of-type': {
            borderLeftColor: 'transparent'
          },
          '&:last-of-type': {
            borderRightColor: 'transparent'
          },
          '&:hover': {
            bgcolor: 'transparent',
            color: theme.palette.primary.main
          }
        }
      }}
    >
      <ToggleButton value="web" aria-label="web">
        <SettingOutlined />
      </ToggleButton>
      <ToggleButton value="android" aria-label="android">
        <EditOutlined />
      </ToggleButton>
      <ToggleButton value="ios" aria-label="ios">
        <EllipsisOutlined />
      </ToggleButton>
    </ToggleButtonGroup>
  );

  const basicCardCodeString = `<MainCard border={false} boxShadow shadow={theme.customShadows.z1} sx={{ height: '100%' }}>
  <Typography variant="h6">Card Subtitle</Typography>
  <Typography variant="caption" color="textSecondary">
    This is card description
  </Typography>
</MainCard>

<MainCard title="Card Title" border={false} boxShadow shadow={theme.customShadows.z1} sx={{ height: '100%' }}>
  <Typography variant="h6">Card Subtitle</Typography>
  <Typography variant="caption" color="textSecondary">
    This is card description
  </Typography>
</MainCard>`;

  const outlinedCardCodeString = `<MainCard sx={{ height: '100%' }}>
  <Typography variant="h6">Card Subtitle</Typography>
  <Typography variant="caption" color="textSecondary">
    This is card description
  </Typography>
</MainCard>

<MainCard title="Card Title" sx={{ height: '100%' }}>
  <Typography variant="h6">Card Subtitle</Typography>
  <Typography variant="caption" color="textSecondary">
    This is card description
  </Typography>
</MainCard>`;

  const actionCardCodeString = `<MainCard
  title="Card Title"
  secondary={
    <Link color="primary" href="/">
      More
    </Link>
  }
>
  <Typography variant="h5" color="textSecondary" gutterBottom>
    Card Subtitle
  </Typography>
  <Typography variant="body1">
    Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim.
  </Typography>
</MainCard>
<MainCard
  title="Card Title"
  secondary={
    <Link color="primary" href="/">
      <MoreOutlined />
    </Link>
  }
>
  <Typography variant="h5" color="textSecondary" gutterBottom>
    Card Subtitle
  </Typography>
  <Typography variant="body1">
    Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim, viverra augue eu, semper
    ligula. Mauris purus sem.
  </Typography>
</MainCard>

<MainCard
  title="Card Title"
  secondary={
    <Link color="primary" href="/">
      <MoreOutlined />
    </Link>
  }
  content={false}
>
  <CardContent>
    <Typography variant="h5" color="textSecondary" gutterBottom>
      Card Subtitle
    </Typography>
    <Typography variant="body1">Lorem ipsum dolor sit amet, consectetur adipiscing elit.</Typography>
  </CardContent>
  <Divider />
  <ToggleButtonGroup
    fullWidth
    color="primary"
    exclusive
    aria-label="text alignment"
    size="small"
    sx={{
      p: 1,
      '& .MuiToggleButton-root': {
        borderRadius: 0,
        p: 0.75,
        '&:not(.Mui-selected)': {
          borderTopColor: 'transparent',
          borderBottomColor: 'transparent'
        },
        '&:first-of-type': {
          borderLeftColor: 'transparent'
        },
        '&:last-of-type': {
          borderRightColor: 'transparent'
        },
        '&:hover': {
          bgcolor: 'transparent',
          color: theme.palette.primary.main
        }
      }
    }}
  >
    <ToggleButton value="web" aria-label="web" disableRipple>
      <SettingOutlined />
    </ToggleButton>
    <ToggleButton value="android" aria-label="android" disableRipple>
      <EditOutlined />
    </ToggleButton>
    <ToggleButton value="ios" aria-label="ios" disableRipple>
      <EllipsisOutlined />
    </ToggleButton>
  </ToggleButtonGroup>
</MainCard>`;

  const mediaCardCodeString = `<MainCard content={false}>
  <CardMedia component="img" image={media} alt="green iguana" />
  <CardContent>
    <Typography variant="h5" color="textSecondary" gutterBottom>
      Card Subtitle
    </Typography>
    <Typography variant="body1">
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim, viverra augue eu.
    </Typography>
  </CardContent>
  <Divider />
  <ToggleButtonGroup
    fullWidth
    color="primary"
    exclusive
    aria-label="text alignment"
    size="small"
    sx={{
      p: 1,
      '& .MuiToggleButton-root': {
        borderRadius: 0,
        p: 0.75,
        '&:not(.Mui-selected)': {
          borderTopColor: 'transparent',
          borderBottomColor: 'transparent'
        },
        '&:first-of-type': {
          borderLeftColor: 'transparent'
        },
        '&:last-of-type': {
          borderRightColor: 'transparent'
        },
        '&:hover': {
          bgcolor: 'transparent',
          color: theme.palette.primary.main
        }
      }
    }}
  >
    <ToggleButton value="web" aria-label="web" disableRipple>
      <SettingOutlined />
    </ToggleButton>
    <ToggleButton value="android" aria-label="android" disableRipple>
      <EditOutlined />
    </ToggleButton>
    <ToggleButton value="ios" aria-label="ios" disableRipple>
      <EllipsisOutlined />
    </ToggleButton>
  </ToggleButtonGroup>
</MainCard>

<MainCard
  title="Card Title"
  secondary={
    <Link color="primary" href="/">
      <MoreOutlined />
    </Link>
  }
  content={false}
>
  <CardMedia component="img" image={media} alt="green iguana" />
  <CardContent>
    <Typography variant="h5" color="textSecondary" gutterBottom>
      Card Subtitle
    </Typography>
    <Typography variant="body1">
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim, viverra augue eu,
    </Typography>
  </CardContent>
</MainCard>

<MainCard
  title="Card Title"
  secondary={
    <Link color="primary" href="/">
      <MoreOutlined />
    </Link>
  }
  content={false}
>
  <CardMedia component="img" image={media} alt="green iguana" />
  <CardContent>
    <Typography variant="h5" color="textSecondary" gutterBottom>
      Card Subtitle
    </Typography>
    <Typography variant="body1">Lorem ipsum dolor sit amet, consectetur adipiscing elit.</Typography>
  </CardContent>
  <Divider />
  <ToggleButtonGroup
    fullWidth
    color="primary"
    exclusive
    aria-label="text alignment"
    size="small"
    sx={{
      p: 1,
      '& .MuiToggleButton-root': {
        borderRadius: 0,
        p: 0.75,
        '&:not(.Mui-selected)': {
          borderTopColor: 'transparent',
          borderBottomColor: 'transparent'
        },
        '&:first-of-type': {
          borderLeftColor: 'transparent'
        },
        '&:last-of-type': {
          borderRightColor: 'transparent'
        },
        '&:hover': {
          bgcolor: 'transparent',
          color: theme.palette.primary.main
        }
      }
    }}
  >
    <ToggleButton value="web" aria-label="web" disableRipple>
      <SettingOutlined />
    </ToggleButton>
    <ToggleButton value="android" aria-label="android" disableRipple>
      <EditOutlined />
    </ToggleButton>
    <ToggleButton value="ios" aria-label="ios" disableRipple>
      <EllipsisOutlined />
    </ToggleButton>
  </ToggleButtonGroup>
</MainCard>`;

  const complexCardCodeString = `<MainCard content={false}>
  <Box sx={{ width: '100%' }}>
    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
      <Tabs value={value} onChange={handleChange} aria-label="basic tabs example">
        <Tab label="Article" {...a11yProps(0)} />
        <Tab label="App" {...a11yProps(1)} />
        <Tab label="Project" {...a11yProps(2)} />
      </Tabs>
    </Box>
    <TabPanel value={value} index={0}>
      <Typography variant="h5" gutterBottom color="textSecondary">
        Article Content
      </Typography>
      <Typography variant="h6" gutterBottom={!activeTab}>
        Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim, viverra augue eu, semper ligula. Mauris
        purus sem, sagittis eu mauris et, viverra lobortis urna.
      </Typography>
      {!activeTab && (
        <Typography variant="h6">
          Suspendisse sed lectus ac nunc rhoncus scelerisque. Integer vitae fringilla leo. Aliquam tincidunt et turpis non mattis.
          Suspendisse blandit velit sit amet velit porta aliquet.
        </Typography>
      )}
    </TabPanel>
    <TabPanel value={value} index={1}>
      <Typography variant="h5" gutterBottom color="textSecondary">
        App Content
      </Typography>
      <Typography variant="h6">
        Suspendisse sed lectus ac nunc rhoncus scelerisque. Integer vitae fringilla leo. Aliquam tincidunt et turpis non mattis. Ut sed
        semper orci, sed facilisis mauris. Suspendisse blandit velit sit amet velit porta aliquet.
      </Typography>
    </TabPanel>
    <TabPanel value={value} index={2}>
      <Typography variant="h5" gutterBottom color="textSecondary">
        Project Content
      </Typography>
      <Typography variant="h6">
        Nam egestas sollicitudin nisl, sit amet aliquam risus pharetra ac. Donec ac lacinia orci. Phasellus ut enim eu ligula placerat
        cursus in nec est.
      </Typography>
    </TabPanel>
  </Box>
</MainCard>

<MainCard
  title="Card Title"
  divider={false}
  content={false}
  secondary={
    <Link color="primary" href="/">
      More
    </Link>
  }
>
  <Box sx={{ width: '100%' }}>
    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
      <Tabs value={value} onChange={handleChange} aria-label="basic tabs example">
        <Tab label="Article" {...a11yProps(0)} />
        <Tab label="App" {...a11yProps(1)} />
        <Tab label="Project" {...a11yProps(2)} />
      </Tabs>
    </Box>
    <TabPanel value={value} index={0}>
      <Typography variant="h5" gutterBottom color="textSecondary">
        Article Content
      </Typography>
      <Typography variant="h6" gutterBottom={!activeTab}>
        Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim, viverra augue eu, semper ligula. Mauris
        purus sem, sagittis eu mauris et, viverra lobortis urna.
      </Typography>
      {!activeTab && (
        <Typography variant="h6">
          Suspendisse sed lectus ac nunc rhoncus scelerisque. Integer vitae fringilla leo. Aliquam tincidunt et turpis non mattis.
          Suspendisse blandit velit sit amet velit porta aliquet.
        </Typography>
      )}
    </TabPanel>
    <TabPanel value={value} index={1}>
      <Typography variant="h5" gutterBottom color="textSecondary">
        App Content
      </Typography>
      <Typography variant="h6">
        Suspendisse sed lectus ac nunc rhoncus scelerisque. Integer vitae fringilla leo. Aliquam tincidunt et turpis non mattis. Ut sed
        semper orci, sed facilisis mauris. Suspendisse blandit velit sit amet velit porta aliquet.
      </Typography>
    </TabPanel>
    <TabPanel value={value} index={2}>
      <Typography variant="h5" gutterBottom color="textSecondary">
        Project Content
      </Typography>
      <Typography variant="h6">
        Nam egestas sollicitudin nisl, sit amet aliquam risus pharetra ac. Donec ac lacinia orci. Phasellus ut enim eu ligula placerat
        cursus in nec est.
      </Typography>
    </TabPanel>
  </Box>
</MainCard>`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Card"
        caption="Cards contain content and actions about a single subject."
        directory="src/pages/components-overview/cards"
        link="https://mui.com/material-ui/react-card/"
      />
      <ComponentWrapper>
        <Grid container spacing={3}>
          <Grid item xs={12} lg={6}>
            <MainCard title="Basic" codeString={basicCardCodeString}>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <MainCard border={false} boxShadow shadow={theme.customShadows.z1} sx={{ height: '100%' }}>
                    <Typography variant="h6">Card Subtitle</Typography>
                    <Typography variant="caption" color="textSecondary">
                      This is card description
                    </Typography>
                  </MainCard>
                </Grid>
                <Grid item xs={12} md={6}>
                  <MainCard title="Card Title" border={false} boxShadow shadow={theme.customShadows.z1} sx={{ height: '100%' }}>
                    <Typography variant="h6">Card Subtitle</Typography>
                    <Typography variant="caption" color="textSecondary">
                      This is card description
                    </Typography>
                  </MainCard>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Outlined" codeString={outlinedCardCodeString}>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <MainCard sx={{ height: '100%' }}>
                    <Typography variant="h6">Card Subtitle</Typography>
                    <Typography variant="caption" color="textSecondary">
                      This is card description
                    </Typography>
                  </MainCard>
                </Grid>
                <Grid item xs={12} md={6}>
                  <MainCard title="Card Title" sx={{ height: '100%' }}>
                    <Typography variant="h6">Card Subtitle</Typography>
                    <Typography variant="caption" color="textSecondary">
                      This is card description
                    </Typography>
                  </MainCard>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} xl={6}>
            <MainCard title="Action" codeString={actionCardCodeString}>
              <Grid container spacing={3}>
                <Grid item xs={12}>
                  <MainCard
                    title="Card Title"
                    secondary={
                      <Link component={RouterLink} to="#" color="primary">
                        More
                      </Link>
                    }
                  >
                    <Typography variant="h5" color="textSecondary" gutterBottom>
                      Card Subtitle
                    </Typography>
                    <Typography variant="body1">
                      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim.
                    </Typography>
                  </MainCard>
                </Grid>
                <Grid item xs={12} md={6}>
                  <MainCard
                    title="Card Title"
                    secondary={
                      <IconButton size="small" color="secondary">
                        <MoreOutlined style={{ fontSize: '1.15rem' }} />
                      </IconButton>
                    }
                  >
                    <Typography variant="h5" color="textSecondary" gutterBottom>
                      Card Subtitle
                    </Typography>
                    <Typography variant="body1">
                      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim, viverra augue eu, semper
                      ligula. Mauris purus sem.
                    </Typography>
                  </MainCard>
                </Grid>
                <Grid item xs={12} md={6}>
                  <MainCard
                    title="Card Title"
                    secondary={
                      <IconButton size="small" color="secondary">
                        <MoreOutlined style={{ fontSize: '1.15rem' }} />
                      </IconButton>
                    }
                    content={false}
                  >
                    <CardContent>
                      <Typography variant="h5" color="textSecondary" gutterBottom>
                        Card Subtitle
                      </Typography>
                      <Typography variant="body1">Lorem ipsum dolor sit amet, consectetur adipiscing elit.</Typography>
                    </CardContent>
                    <Divider />
                    {cardAction}
                  </MainCard>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} xl={6}>
            <MainCard title="Media" codeString={mediaCardCodeString}>
              <Grid container spacing={3}>
                <Grid item xs={12} sm={6} lg={4}>
                  <MainCard content={false}>
                    <CardMedia component="img" image={media} alt="green iguana" />
                    <CardContent>
                      <Typography variant="h5" color="textSecondary" gutterBottom>
                        Card Subtitle
                      </Typography>
                      <Typography variant="body1">
                        Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim, viverra augue eu.
                      </Typography>
                    </CardContent>
                    <Divider />
                    {cardAction}
                  </MainCard>
                </Grid>
                <Grid item xs={12} sm={6} lg={4}>
                  <MainCard
                    title="Card Title"
                    secondary={
                      <IconButton size="small" color="secondary">
                        <MoreOutlined style={{ fontSize: '1.15rem' }} />
                      </IconButton>
                    }
                    content={false}
                  >
                    <CardMedia component="img" image={media} alt="green iguana" />
                    <CardContent>
                      <Typography variant="h5" color="textSecondary" gutterBottom>
                        Card Subtitle
                      </Typography>
                      <Typography variant="body1">
                        Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque non libero dignissim, viverra augue eu,
                      </Typography>
                    </CardContent>
                  </MainCard>
                </Grid>
                <Grid item xs={12} sm={6} lg={4}>
                  <MainCard
                    title="Card Title"
                    secondary={
                      <IconButton size="small" color="secondary">
                        <MoreOutlined style={{ fontSize: '1.15rem' }} />
                      </IconButton>
                    }
                    content={false}
                  >
                    <CardMedia component="img" image={media} alt="green iguana" />
                    <CardContent>
                      <Typography variant="h5" color="textSecondary" gutterBottom>
                        Card Subtitle
                      </Typography>
                      <Typography variant="body1">Lorem ipsum dolor sit amet, consectetur adipiscing elit.</Typography>
                    </CardContent>
                    <Divider />
                    {cardAction}
                  </MainCard>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
          <Grid item xs={12} lg={6}>
            <MainCard title="Complex Interaction" codeString={complexCardCodeString}>
              <Grid container spacing={3}>
                <Grid item xs={12}>
                  <MainCard content={false}>
                    <CardTabs />
                  </MainCard>
                </Grid>
                <Grid item xs={12}>
                  <MainCard
                    title="Card Title"
                    divider={false}
                    content={false}
                    secondary={
                      <Link component={RouterLink} to="#" color="primary">
                        More
                      </Link>
                    }
                  >
                    <CardTabs activeTab={2} />
                  </MainCard>
                </Grid>
              </Grid>
            </MainCard>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentCard;
