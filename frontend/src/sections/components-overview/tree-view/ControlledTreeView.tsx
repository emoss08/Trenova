import { useState, SyntheticEvent } from 'react';

// material-ui
import { Box, Button } from '@mui/material';
import { TreeItem, TreeView } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// assets
import { DownOutlined, RightOutlined } from '@ant-design/icons';

// ==============================|| TREE VIEW - CONTROLLED ||============================== //

export default function ControlledTreeView() {
  const [expanded, setExpanded] = useState<string[]>(['1']);
  const [selected, setSelected] = useState<string[]>([]);

  const handleToggle = (event: SyntheticEvent, nodeIds: string[]) => {
    setExpanded(nodeIds);
  };

  const handleSelect = (event: SyntheticEvent, nodeIds: string[]) => {
    setSelected(nodeIds);
  };

  const handleExpandClick = () => {
    setExpanded((oldExpanded) => (oldExpanded.length === 0 ? ['1', '5', '6', '7'] : []));
  };

  const handleSelectClick = () => {
    setSelected((oldSelected) => (oldSelected.length === 0 ? ['1', '2', '3', '4', '5', '6', '7', '8', '9'] : []));
  };

  const controlledTreeviewCodeString = `<Box sx={{ height: 270, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}>
  <Box sx={{ mb: 1 }}>
    <Button onClick={handleExpandClick}>{expanded.length === 0 ? 'Expand all' : 'Collapse all'}</Button>
    <Button onClick={handleSelectClick}>{selected.length === 0 ? 'Select all' : 'Unselect all'}</Button>
  </Box>
  <TreeView
    aria-label="controlled"
    defaultCollapseIcon={<DownOutlined />}
    defaultExpandIcon={<RightOutlined />}
    expanded={expanded}
    selected={selected}
    onNodeToggle={handleToggle}
    onNodeSelect={handleSelect}
    multiSelect
  >
    <TreeItem nodeId="1" label="Applications">
      <TreeItem nodeId="2" label="Calendar" />
      <TreeItem nodeId="3" label="Chrome" />
      <TreeItem nodeId="4" label="Webstorm" />
    </TreeItem>
    <TreeItem nodeId="5" label="Documents">
      <TreeItem nodeId="6" label="MUI">
        <TreeItem nodeId="7" label="src">
          <TreeItem nodeId="8" label="index.js" />
          <TreeItem nodeId="9" label="tree-view.js" />
        </TreeItem>
      </TreeItem>
    </TreeItem>
  </TreeView>
</Box>`;

  return (
    <MainCard title="Controlled" codeString={controlledTreeviewCodeString}>
      <Box sx={{ height: 270, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}>
        <Box sx={{ mb: 1 }}>
          <Button onClick={handleExpandClick}>{expanded.length === 0 ? 'Expand all' : 'Collapse all'}</Button>
          <Button onClick={handleSelectClick}>{selected.length === 0 ? 'Select all' : 'Unselect all'}</Button>
        </Box>
        <TreeView
          aria-label="controlled"
          defaultCollapseIcon={<DownOutlined />}
          defaultExpandIcon={<RightOutlined />}
          expanded={expanded}
          selected={selected}
          onNodeToggle={handleToggle}
          onNodeSelect={handleSelect}
          multiSelect
        >
          <TreeItem nodeId="1" label="Applications">
            <TreeItem nodeId="2" label="Calendar" />
            <TreeItem nodeId="3" label="Chrome" />
            <TreeItem nodeId="4" label="Webstorm" />
          </TreeItem>
          <TreeItem nodeId="5" label="Documents">
            <TreeItem nodeId="6" label="MUI">
              <TreeItem nodeId="7" label="src">
                <TreeItem nodeId="8" label="index.js" />
                <TreeItem nodeId="9" label="tree-view.js" />
              </TreeItem>
            </TreeItem>
          </TreeItem>
        </TreeView>
      </Box>
    </MainCard>
  );
}
