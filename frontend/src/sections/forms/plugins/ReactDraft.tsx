import { useState } from 'react';

// third-party
import { Editor, EditorState as EditorType } from 'react-draft-wysiwyg';
import { ContentState, EditorState } from 'draft-js';
import 'react-draft-wysiwyg/dist/react-draft-wysiwyg.css';

// ==============================|| DRAFT EDITOR ||============================== //

const ReactDraft = () => {
  const [editorState, setEditorState] = useState(() => {
    const initialContent =
      "Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.";
    return EditorState.createWithContent(ContentState.createFromText(initialContent));
  });

  const onEditorStateChange = (editor: EditorType) => {
    setEditorState(editor);
  };
  return (
    <Editor
      editorState={editorState}
      toolbarClassName="toolbarClassName"
      wrapperClassName="wrapperClassName"
      editorClassName="editorClassName"
      onEditorStateChange={onEditorStateChange}
    />
  );
};

export default ReactDraft;
