import { DocumentTemplateSchema } from "@/lib/schemas/document-template-schema";
import { useCallback } from "react";
import { useFormContext, useWatch } from "react-hook-form";

export function DocumentTemplateLivePreview() {
  const { control } = useFormContext<DocumentTemplateSchema>();

  const cssContent = useWatch({ control, name: "cssContent" });
  const headerHtml = useWatch({ control, name: "headerHtml" });
  const footerHtml = useWatch({ control, name: "footerHtml" });
  const pageSize = useWatch({ control, name: "pageSize" });
  const orientation = useWatch({ control, name: "orientation" });
  const marginTop = useWatch({ control, name: "marginTop" });
  const marginBottom = useWatch({ control, name: "marginBottom" });
  const marginLeft = useWatch({ control, name: "marginLeft" });
  const marginRight = useWatch({ control, name: "marginRight" });
  const htmlContent = useWatch({ control, name: "htmlContent" });

  const generatePreviewHtml = useCallback(() => {
    const styles = cssContent ? `<style>${cssContent}</style>` : "";
    const header = headerHtml
      ? `<header class="page-header">${headerHtml}</header>`
      : "";
    const footer = footerHtml
      ? `<footer class="page-footer">${footerHtml}</footer>`
      : "";

    const paperStyles = {
      Letter: {
        Portrait: { width: "8.5in", height: "11in" },
        Landscape: { width: "11in", height: "8.5in" },
      },
      A4: {
        Portrait: { width: "210mm", height: "297mm" },
        Landscape: { width: "297mm", height: "210mm" },
      },
      Legal: {
        Portrait: { width: "8.5in", height: "14in" },
        Landscape: { width: "14in", height: "8.5in" },
      },
    };

    const paper =
      paperStyles[pageSize as keyof typeof paperStyles]?.[
        orientation as "Portrait" | "Landscape"
      ] || paperStyles.Letter.Portrait;

    return `
      <!DOCTYPE html>
      <html>
        <head>
          <meta charset="utf-8">
          <meta name="viewport" content="width=device-width, initial-scale=1">
          ${styles}
          <style>
            * { box-sizing: border-box; }
            html, body {
              margin: 0;
              padding: 0;
              background: #f3f4f6;
              min-height: 100%;
            }
            .page {
              width: ${paper.width};
              min-height: ${paper.height};
              margin: 20px auto;
              padding: ${marginTop || 20}mm ${marginRight || 20}mm ${marginBottom || 20}mm ${marginLeft || 20}mm;
              background: white;
              box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1);
              font-family: system-ui, -apple-system, sans-serif;
              color: #1f2937;
              position: relative;
            }
            .page-header {
              position: absolute;
              top: 10mm;
              left: ${marginLeft || 20}mm;
              right: ${marginRight || 20}mm;
              font-size: 10px;
              color: #6b7280;
            }
            .page-footer {
              position: absolute;
              bottom: 10mm;
              left: ${marginLeft || 20}mm;
              right: ${marginRight || 20}mm;
              font-size: 10px;
              color: #6b7280;
              text-align: center;
            }
            .placeholder-text {
              color: #9ca3af;
              font-style: italic;
            }
          </style>
        </head>
        <body>
          <div class="page">
            ${header}
            ${htmlContent || '<p class="placeholder-text">Start typing HTML content to see preview...</p>'}
            ${footer}
          </div>
        </body>
      </html>
    `;
  }, [
    htmlContent,
    cssContent,
    headerHtml,
    footerHtml,
    pageSize,
    orientation,
    marginTop,
    marginBottom,
    marginLeft,
    marginRight,
  ]);

  return (
    <div className="min-h-0 flex-1 overflow-auto bg-neutral-100 p-4 dark:bg-neutral-900">
      <iframe
        srcDoc={generatePreviewHtml()}
        className="size-full rounded-lg border-0 bg-transparent"
        title="Template Preview"
        sandbox="allow-same-origin"
      />
    </div>
  );
}
