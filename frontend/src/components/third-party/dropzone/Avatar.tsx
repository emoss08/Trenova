// material-ui
import { alpha, styled, useTheme } from '@mui/material/styles';
import { Stack, Typography } from '@mui/material';

// third-party
import { useDropzone } from 'react-dropzone';

// project import
import RejectionFiles from './RejectionFiles';

// assets
import { CameraOutlined } from '@ant-design/icons';

// types
import { CustomFile, UploadProps } from 'types/dropzone';

const RootWrapper = styled('div')(({ theme }) => ({
  width: 124,
  height: 124,
  borderRadius: '50%',
  border: `1px dashed ${theme.palette.primary.main}`,
  background: theme.palette.primary.lighter
}));

const DropzoneWrapper = styled('div')({
  zIndex: 0,
  width: '100%',
  height: '100%',
  outline: 'none',
  display: 'flex',
  overflow: 'hidden',
  borderRadius: '50%',
  position: 'relative',
  alignItems: 'center',
  justifyContent: 'center',
  '& > *': { width: '100%', height: '100%' },
  '&:hover': {
    cursor: 'pointer',
    '& .placeholder': {
      zIndex: 9
    }
  }
});

const PlaceholderWrapper = styled('div')(({ theme }) => ({
  display: 'flex',
  position: 'absolute',
  alignItems: 'center',
  justifyContent: 'center',
  color: theme.palette.text.secondary,
  backgroundColor: alpha(theme.palette.primary.lighter, 0.75),
  transition: theme.transitions.create('opacity', {
    easing: theme.transitions.easing.easeInOut,
    duration: theme.transitions.duration.shorter
  }),
  '&:hover': { opacity: 0.85 }
}));

// ==============================|| UPLOAD - AVATAR ||============================== //

const AvatarUpload = ({ error, file, setFieldValue, sx, ...other }: UploadProps) => {
  const theme = useTheme();

  const { getRootProps, getInputProps, isDragActive, isDragReject, fileRejections } = useDropzone({
    accept: {
      'image/*': []
    },
    multiple: false,
    onDrop: (acceptedFiles: CustomFile[]) => {
      setFieldValue(
        'files',
        acceptedFiles.map((file: CustomFile) =>
          Object.assign(file, {
            preview: URL.createObjectURL(file)
          })
        )
      );
    }
  });

  const thumbs =
    file &&
    file.map((item: CustomFile) => (
      <img
        key={item.name}
        alt={item.name}
        src={item.preview}
        onLoad={() => {
          URL.revokeObjectURL(item.preview!);
        }}
      />
    ));

  return (
    <>
      <RootWrapper
        sx={{
          ...((isDragReject || error) && {
            borderColor: 'error.light'
          }),
          ...sx
        }}
      >
        <DropzoneWrapper {...getRootProps()} sx={{ ...(isDragActive && { opacity: 0.6 }) }}>
          <input {...getInputProps()} />
          {thumbs}
          <PlaceholderWrapper
            className="placeholder"
            sx={{
              ...(thumbs && {
                opacity: 0,
                color: 'common.white',
                bgcolor: 'grey.900'
              }),
              ...((isDragReject || error) && {
                bgcolor: 'error.lighter'
              })
            }}
          >
            <Stack spacing={0.5} alignItems="center">
              <CameraOutlined style={{ color: theme.palette.secondary.main, fontSize: '2rem' }} />
              <Typography color="secondary">{file ? 'Update' : 'Upload'}</Typography>
            </Stack>
          </PlaceholderWrapper>
        </DropzoneWrapper>
      </RootWrapper>
      {fileRejections.length > 0 && <RejectionFiles fileRejections={fileRejections} />}
    </>
  );
};

export default AvatarUpload;
