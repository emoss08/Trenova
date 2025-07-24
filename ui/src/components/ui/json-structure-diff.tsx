/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";
import { detectSensitiveDataType } from "@/lib/json-sensitive-utils";
import { SensitiveBadge } from "./sensitive-badge";
import React, { useMemo } from "react";
import { ScrollArea } from "./scroll-area";

interface DiffLine {
  lineNumber: number;
  content: string;
  type: 'added' | 'removed' | 'unchanged' | 'context';
  indentLevel: number;
  isKey?: boolean;
  value?: any;
  key?: string;
}

interface JsonStructureDiffProps {
  oldData: any;
  newData: any;
  title?: {
    old: string;
    new: string;
  };
  className?: string;
}

function jsonToLines(obj: any, indent: number = 0, lines: DiffLine[] = [], lineNum = { current: 1 }): DiffLine[] {
  const spaces = '  '.repeat(indent);
  
  if (obj === null || obj === undefined) {
    lines.push({
      lineNumber: lineNum.current++,
      content: `${spaces}null`,
      type: 'unchanged',
      indentLevel: indent,
      value: null,
    });
    return lines;
  }

  if (typeof obj !== 'object') {
    // Primitive value
    const content = typeof obj === 'string' ? `"${obj}"` : String(obj);
    lines.push({
      lineNumber: lineNum.current++,
      content: `${spaces}${content}`,
      type: 'unchanged',
      indentLevel: indent,
      value: obj,
    });
    return lines;
  }

  const isArray = Array.isArray(obj);
  const entries = isArray ? obj.map((v, i) => [i, v]) : Object.entries(obj);
  
  // Opening bracket
  lines.push({
    lineNumber: lineNum.current++,
    content: `${spaces}${isArray ? '[' : '{'}`,
    type: 'unchanged',
    indentLevel: indent,
  });

  // Entries
  entries.forEach(([key, value], index) => {
    const isLast = index === entries.length - 1;
    const keyStr = isArray ? '' : `"${key}": `;
    
    if (value !== null && typeof value === 'object') {
      // Object or array
      if (isArray) {
        jsonToLines(value, indent + 1, lines, lineNum);
      } else {
        lines.push({
          lineNumber: lineNum.current++,
          content: `${spaces}  "${key}": ${Array.isArray(value) ? '[' : '{'}`,
          type: 'unchanged',
          indentLevel: indent + 1,
          isKey: true,
          key: key,
        });
        
        const subLines: DiffLine[] = [];
        jsonToLines(value, 0, subLines, { current: 1 });
        
        // Skip the opening bracket from subLines
        subLines.slice(1, -1).forEach(line => {
          lines.push({
            ...line,
            lineNumber: lineNum.current++,
            content: spaces + '  ' + line.content,
            indentLevel: line.indentLevel + indent + 1,
          });
        });
        
        lines.push({
          lineNumber: lineNum.current++,
          content: `${spaces}  }${isLast ? '' : ','}`,
          type: 'unchanged',
          indentLevel: indent + 1,
        });
      }
    } else {
      // Primitive value
      const valueStr = value === null ? 'null' : 
                      typeof value === 'string' ? `"${value}"` : 
                      String(value);
      lines.push({
        lineNumber: lineNum.current++,
        content: `${spaces}  ${keyStr}${valueStr}${isLast ? '' : ','}`,
        type: 'unchanged',
        indentLevel: indent + 1,
        key: isArray ? undefined : String(key),
        value: value,
      });
    }
  });

  // Closing bracket
  lines.push({
    lineNumber: lineNum.current++,
    content: `${spaces}${isArray ? ']' : '}'}`,
    type: 'unchanged',
    indentLevel: indent,
  });

  return lines;
}

function createDiff(oldData: any, newData: any): { oldLines: DiffLine[], newLines: DiffLine[] } {
  // For now, let's create a simple diff that shows the full structure
  // In a real implementation, you'd want to use a proper diff algorithm
  const oldLines = jsonToLines(oldData);
  const newLines = jsonToLines(newData);
  
  // Mark changes based on simple comparison
  // This is a simplified version - a real diff would be more sophisticated
  const oldJson = JSON.stringify(oldData, null, 2);
  const newJson = JSON.stringify(newData, null, 2);
  
  if (oldJson !== newJson) {
    // Mark all lines as changed for now
    oldLines.forEach(line => {
      if (line.value !== undefined || line.key !== undefined) {
        line.type = 'removed';
      }
    });
    newLines.forEach(line => {
      if (line.value !== undefined || line.key !== undefined) {
        line.type = 'added';
      }
    });
  }
  
  return { oldLines, newLines };
}

function DiffLineComponent({ line }: { line: DiffLine }) {
  const bgColor = useMemo(() => {
    switch (line.type) {
      case 'added': return 'bg-green-50 dark:bg-green-950/20';
      case 'removed': return 'bg-red-50 dark:bg-red-950/20';
      default: return '';
    }
  }, [line.type]);

  const renderContent = () => {
    const parts: React.ReactNode[] = [];
    
    // Parse the content to apply syntax highlighting
    const content = line.content;
    
    // Match JSON structure
    const keyMatch = content.match(/^(\s*)"([^"]+)":\s*/);
    if (keyMatch) {
      // It's a key-value pair
      parts.push(
        <span key="indent" className="whitespace-pre">{keyMatch[1]}</span>,
        <span key="key" className="text-blue-600 dark:text-blue-400">"{keyMatch[2]}"</span>,
        <span key="colon" className="text-muted-foreground">: </span>
      );
      
      const remainingContent = content.slice(keyMatch[0].length);
      
      // Check if value is sensitive
      if (line.value !== undefined) {
        const sensitiveInfo = detectSensitiveDataType(line.value);
        if (sensitiveInfo.isSensitive) {
          parts.push(
            <span key="value" className="inline-flex items-center gap-1">
              <span className={cn(
                "text-green-600 dark:text-green-400",
                sensitiveInfo.type === "redacted" && "text-red-600 dark:text-red-400"
              )}>
                {remainingContent.replace(/,$/, '')}
              </span>
              <SensitiveBadge 
                size="xs" 
                variant={sensitiveInfo.type === "redacted" ? "destructive" : "warning"} 
              />
              {remainingContent.endsWith(',') && <span className="text-muted-foreground">,</span>}
            </span>
          );
          return parts;
        }
      }
      
      // Regular value
      if (remainingContent.startsWith('"')) {
        // String value
        const stringMatch = remainingContent.match(/^"[^"]*"/);
        if (stringMatch) {
          parts.push(
            <span key="value" className="text-green-600 dark:text-green-400">
              {stringMatch[0]}
            </span>
          );
          if (remainingContent.endsWith(',')) {
            parts.push(<span key="comma" className="text-muted-foreground">,</span>);
          }
        }
      } else if (remainingContent.match(/^(true|false)/)) {
        // Boolean
        parts.push(
          <span key="value" className="text-purple-600 dark:text-purple-400">
            {remainingContent.replace(/,$/, '')}
          </span>
        );
        if (remainingContent.endsWith(',')) {
          parts.push(<span key="comma" className="text-muted-foreground">,</span>);
        }
      } else if (remainingContent.match(/^-?\d+(\.\d+)?/)) {
        // Number
        parts.push(
          <span key="value" className="text-blue-600 dark:text-blue-400">
            {remainingContent.replace(/,$/, '')}
          </span>
        );
        if (remainingContent.endsWith(',')) {
          parts.push(<span key="comma" className="text-muted-foreground">,</span>);
        }
      } else if (remainingContent.match(/^null/)) {
        // Null
        parts.push(
          <span key="value" className="text-gray-500 dark:text-gray-400">
            {remainingContent.replace(/,$/, '')}
          </span>
        );
        if (remainingContent.endsWith(',')) {
          parts.push(<span key="comma" className="text-muted-foreground">,</span>);
        }
      } else {
        parts.push(<span key="value">{remainingContent}</span>);
      }
    } else {
      // It's a bracket or other content
      const bracketMatch = content.match(/^(\s*)([{}[\],]+)$/);
      if (bracketMatch) {
        parts.push(
          <span key="indent" className="whitespace-pre">{bracketMatch[1]}</span>,
          <span key="bracket" className="text-muted-foreground">{bracketMatch[2]}</span>
        );
      } else {
        // Check if it's just a value (for arrays)
        const valueMatch = content.match(/^(\s*)(.+?)([,]?)$/);
        if (valueMatch && line.value !== undefined) {
          parts.push(<span key="indent" className="whitespace-pre">{valueMatch[1]}</span>);
          
          const sensitiveInfo = detectSensitiveDataType(line.value);
          if (sensitiveInfo.isSensitive) {
            parts.push(
              <span key="value" className="inline-flex items-center gap-1">
                <span className={cn(
                  "text-green-600 dark:text-green-400",
                  sensitiveInfo.type === "redacted" && "text-red-600 dark:text-red-400"
                )}>
                  {valueMatch[2]}
                </span>
                <SensitiveBadge 
                  size="xs" 
                  variant={sensitiveInfo.type === "redacted" ? "destructive" : "warning"} 
                />
              </span>
            );
          } else if (valueMatch[2].startsWith('"')) {
            parts.push(
              <span key="value" className="text-green-600 dark:text-green-400">
                {valueMatch[2]}
              </span>
            );
          } else if (valueMatch[2].match(/^(true|false)$/)) {
            parts.push(
              <span key="value" className="text-purple-600 dark:text-purple-400">
                {valueMatch[2]}
              </span>
            );
          } else if (valueMatch[2].match(/^-?\d+(\.\d+)?$/)) {
            parts.push(
              <span key="value" className="text-blue-600 dark:text-blue-400">
                {valueMatch[2]}
              </span>
            );
          } else {
            parts.push(<span key="value">{valueMatch[2]}</span>);
          }
          
          if (valueMatch[3]) {
            parts.push(<span key="comma" className="text-muted-foreground">{valueMatch[3]}</span>);
          }
        } else {
          parts.push(<span key="content">{content}</span>);
        }
      }
    }
    
    return parts;
  };

  return (
    <div className={cn("flex items-center px-2 py-0.5 font-mono text-sm", bgColor)}>
      <span className="w-12 text-right text-xs text-muted-foreground select-none pr-4">
        {line.lineNumber}
      </span>
      <span className="flex-1">
        {renderContent()}
      </span>
    </div>
  );
}

export function JsonStructureDiff({ 
  oldData, 
  newData, 
  title = { old: "Previous Version", new: "Current Version" },
  className 
}: JsonStructureDiffProps) {
  const { oldLines, newLines } = useMemo(
    () => createDiff(oldData, newData),
    [oldData, newData]
  );

  return (
    <div className={cn("grid grid-cols-2 gap-4 h-full", className)}>
      {/* Old Version */}
      <div className="flex flex-col border border-border rounded-lg overflow-hidden">
        <div className="px-4 py-2 bg-muted border-b border-border">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-sm">{title.old}</h3>
            <span className="text-xs text-muted-foreground">
              {oldLines.filter(l => l.type === 'removed').length} removals
            </span>
          </div>
        </div>
        <ScrollArea className="flex-1">
          <div className="min-w-0">
            {oldLines.map((line, idx) => (
              <DiffLineComponent key={idx} line={line} />
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* New Version */}
      <div className="flex flex-col border border-border rounded-lg overflow-hidden">
        <div className="px-4 py-2 bg-muted border-b border-border">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-sm">{title.new}</h3>
            <span className="text-xs text-muted-foreground">
              {newLines.filter(l => l.type === 'added').length} additions
            </span>
          </div>
        </div>
        <ScrollArea className="flex-1">
          <div className="min-w-0">
            {newLines.map((line, idx) => (
              <DiffLineComponent key={idx} line={line} />
            ))}
          </div>
        </ScrollArea>
      </div>
    </div>
  );
}