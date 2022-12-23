import { useState } from 'react';
import { useDispatch } from 'store';

// material-ui
import { Button, CardContent, Grid, InputAdornment, InputLabel, Stack, TextField, Tooltip } from '@mui/material';

// third-party
import { CopyToClipboard } from 'react-copy-to-clipboard';

// project imports
import MainCard from 'components/MainCard';
import IconButton from 'components/@extended/IconButton';
import { openSnackbar } from 'store/reducers/snackbar';

// assets
import { CopyOutlined, ScissorOutlined } from '@ant-design/icons';

// ==============================|| PLUGIN - CLIPBOARD ||============================== //

const ClipboardPage = () => {
  const dispatch = useDispatch();

  const [text1, setText1] = useState('https://berrydashboard.io/');
  const [text2, setText2] = useState(
    'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.'
  );
  const [text3] = useState(
    'Lorem ipsum cacilds, vidis litro abertis. Consetis adipiscings elitis. Pra lá , depois divoltis porris, paradis. Paisis, filhis, espiritis santis. Mé faiz elementum girarzis, nisi eros vermeio, in elementis mé pra quem é amistosis quis leo. Manduma pindureta quium dia nois paga.'
  );

  return (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <MainCard title="Copy from TextField">
          <Stack spacing={0.5}>
            <InputLabel>Enter Website</InputLabel>
            <TextField
              fullWidth
              placeholder="Website"
              type="text"
              value={text1}
              onChange={(e) => {
                setText1(e.target.value);
              }}
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <CopyToClipboard
                      text={text1}
                      onCopy={() =>
                        dispatch(
                          openSnackbar({
                            open: true,
                            message: 'Text Copied',
                            variant: 'alert',
                            alert: {
                              color: 'success'
                            },
                            close: false,
                            anchorOrigin: { vertical: 'top', horizontal: 'right' },
                            transition: 'SlideLeft'
                          })
                        )
                      }
                    >
                      <Tooltip title="Copy">
                        <IconButton aria-label="Copy from another element" color="secondary" edge="end" size="large">
                          <CopyOutlined />
                        </IconButton>
                      </Tooltip>
                    </CopyToClipboard>
                  </InputAdornment>
                )
              }}
            />
          </Stack>
        </MainCard>
      </Grid>
      <Grid item xs={12} md={6}>
        <MainCard title="Copy from TextArea">
          <InputLabel sx={{ pb: 0.5 }}>Enter Text to Copy</InputLabel>
          <TextField
            multiline
            rows={4}
            fullWidth
            placeholder="Copy text"
            onChange={(e) => setText2(e.target.value)}
            value={text2}
            sx={{ mb: 3 }}
          />
          <CopyToClipboard
            text={text2}
            onCopy={() =>
              dispatch(
                openSnackbar({
                  open: true,
                  message: 'Text Copied',
                  variant: 'alert',
                  alert: {
                    color: 'success'
                  },
                  close: false,
                  anchorOrigin: { vertical: 'top', horizontal: 'right' },
                  transition: 'SlideLeft'
                })
              )
            }
          >
            <Button variant="contained" size="small" sx={{ mr: 1.5 }}>
              <CopyOutlined style={{ marginRight: 1 }} /> Copy
            </Button>
          </CopyToClipboard>
          <CopyToClipboard
            text={text2}
            onCopy={() => {
              setText2('');
              dispatch(
                openSnackbar({
                  open: true,
                  message: 'Text Cut',
                  variant: 'alert',
                  alert: {
                    color: 'success'
                  },
                  close: false,
                  anchorOrigin: { vertical: 'top', horizontal: 'right' },
                  transition: 'SlideLeft'
                })
              );
            }}
          >
            <Button variant="contained" size="small" color="error">
              <ScissorOutlined style={{ marginRight: 1 }} /> Cut
            </Button>
          </CopyToClipboard>
        </MainCard>
      </Grid>
      <Grid item xs={12} md={6}>
        <MainCard
          title="Copy from Container"
          secondary={
            <CopyToClipboard
              text={text3}
              onCopy={() =>
                dispatch(
                  openSnackbar({
                    open: true,
                    message: 'Text Copied',
                    variant: 'alert',
                    alert: {
                      color: 'success'
                    },
                    close: false,
                    anchorOrigin: { vertical: 'top', horizontal: 'right' },
                    transition: 'SlideLeft'
                  })
                )
              }
            >
              <Tooltip title="Copy">
                <IconButton size="large">
                  <CopyOutlined />
                </IconButton>
              </Tooltip>
            </CopyToClipboard>
          }
        >
          <CardContent sx={{ p: 0, pb: 2.5 }}>{text3}</CardContent>
        </MainCard>
      </Grid>
    </Grid>
  );
};

export default ClipboardPage;
