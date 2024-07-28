import React from "react";

interface JsonViewerProps {
  json: any;
  dark?: boolean;
}

const JsonViewer: React.FC<JsonViewerProps> = ({ json, dark = true }) => {
  const syntaxHighlight = (obj: any) => {
    const jsonStr = JSON.stringify(obj, null, 2);
    return jsonStr.replace(
      /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+-]?\d+)?)/g,
      (match) => {
        let cls = "json-number";
        if (/^"/.test(match)) {
          if (/:$/.test(match)) {
            cls = "json-key";
          } else {
            cls = "json-string";
          }
        } else if (/true|false/.test(match)) {
          cls = "json-boolean";
        } else if (/null/.test(match)) {
          cls = "json-null";
        }
        return `<span class="${cls}">${match}</span>`;
      },
    );
  };

  const highlightedJson = syntaxHighlight(json);

  const baseStyle = {
    fontFamily: "monospace",
    fontSize: "14px",
    lineHeight: "1.5",
    whiteSpace: "pre-wrap" as const,
    wordWrap: "break-word" as const,
    padding: "10px",
  };

  const darkTheme = {
    ...baseStyle,
    backgroundColor: "#09090b",
    color: "#d4d4d4",
  };

  const lightTheme = {
    ...baseStyle,
    backgroundColor: "#f4f4f4",
    color: "#333",
  };

  const style = dark ? darkTheme : lightTheme;

  return (
    <div style={style}>
      <style>{`
        .json-key { color: ${dark ? "#9cdcfe" : "#0451a5"}; }
        .json-string { color: ${dark ? "#ce9178" : "#a31515"}; }
        .json-number { color: ${dark ? "#b5cea8" : "#098658"}; }
        .json-boolean { color: ${dark ? "#569cd6" : "#0000ff"}; }
        .json-null { color: ${dark ? "#569cd6" : "#0000ff"}; }
      `}</style>
      <div dangerouslySetInnerHTML={{ __html: highlightedJson }} />
    </div>
  );
};

export default JsonViewer;
