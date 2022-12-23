// material-ui
import { Box } from '@mui/material';
import { TreeView, TreeItem } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// assets
import { DownOutlined, RightOutlined } from '@ant-design/icons';

// ==============================|| TREE VIEW - DISABLED ||============================== //

export default function DisabledTreeView() {
  const disabledTreeviewCodeString = `<TreeView aria-label="disabled items" defaultCollapseIcon={<DownOutlined />} defaultExpandIcon={<RightOutlined />} multiSelect>
  <TreeItem nodeId="1" label="One">
    <TreeItem nodeId="2" label="Two" />
    <TreeItem nodeId="3" label="Three" />
    <TreeItem nodeId="4" label="Four" />
  </TreeItem>
  <TreeItem nodeId="5" label="Five" disabled>
    <TreeItem nodeId="6" label="Six" />
  </TreeItem>
  <TreeItem nodeId="7" label="Seven">
    <TreeItem nodeId="8" label="Eight" />
    <TreeItem nodeId="9" label="Nine">
      <TreeItem nodeId="10" label="Ten">
        <TreeItem nodeId="11" label="Eleven" />
        <TreeItem nodeId="12" label="Twelve" />
      </TreeItem>
    </TreeItem>
  </TreeItem>
</TreeView>`;

  return (
    <MainCard title="Disabled" codeString={disabledTreeviewCodeString}>
      <Box sx={{ height: 240, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}>
        <TreeView aria-label="disabled items" defaultCollapseIcon={<DownOutlined />} defaultExpandIcon={<RightOutlined />} multiSelect>
          <TreeItem nodeId="1" label="One">
            <TreeItem nodeId="2" label="Two" />
            <TreeItem nodeId="3" label="Three" />
            <TreeItem nodeId="4" label="Four" />
          </TreeItem>
          <TreeItem nodeId="5" label="Five" disabled>
            <TreeItem nodeId="6" label="Six" />
          </TreeItem>
          <TreeItem nodeId="7" label="Seven">
            <TreeItem nodeId="8" label="Eight" />
            <TreeItem nodeId="9" label="Nine">
              <TreeItem nodeId="10" label="Ten">
                <TreeItem nodeId="11" label="Eleven" />
                <TreeItem nodeId="12" label="Twelve" />
              </TreeItem>
            </TreeItem>
          </TreeItem>
        </TreeView>
      </Box>
    </MainCard>
  );
}
