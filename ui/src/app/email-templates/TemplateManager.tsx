import { useEffect, useState } from "react";
import { toast } from "sonner";
import SampleDataEditor from "./SampleDataEditor";
import TemplateEditor from "./TemplateEditor";
import TemplateList from "./TemplateList";
import TemplatePreview from "./TemplatePreview";

export default function TemplateManager() {
  const [templates, setTemplates] = useState<string[]>([]);
  const [currentTemplate, setCurrentTemplate] = useState<string | null>(null);
  const [templateContent, setTemplateContent] = useState("");
  const [sampleData, setSampleData] = useState<Record<string, any>>({});
  const [activeTab, setActiveTab] = useState<"editor" | "samples">("editor");
  const [socket, setSocket] = useState<WebSocket | null>(null);

  // Load templates
  useEffect(() => {
    fetchTemplates();
    setupWebSocket();

    return () => {
      if (socket) {
        socket.close();
      }
    };
  }, []);

  // Setup WebSocket
  const setupWebSocket = () => {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    const ws = new WebSocket(wsUrl);
    
    ws.addEventListener("open", () => {
      console.log("Connected to WebSocket server");
    });
    
    ws.addEventListener("message", (event) => {
      try {
        const data = JSON.parse(event.data);
        
        if (data.type === "template_updated" && data.templateName === currentTemplate) {
          toast.info(`Template "${data.templateName}" was updated by another user`);
          fetchTemplateContent(data.templateName);
        } else if (data.type === "sample_updated" && data.sampleName === currentTemplate) {
          toast.info(`Sample data for "${data.sampleName}" was updated by another user`);
          if (activeTab === "samples") {
            fetchSampleData(data.sampleName);
          }
        }
      } catch (error) {
        console.error("Error processing WebSocket message:", error);
      }
    });
    
    ws.addEventListener("close", () => {
      console.log("Disconnected from WebSocket server");
      // Try to reconnect after 3 seconds
      setTimeout(setupWebSocket, 3000);
    });
    
    setSocket(ws);
  };

  const fetchTemplates = async () => {
    try {
      const response = await fetch("/api/templates");
      const data = await response.json();
      setTemplates(data);
      
      // Select first template by default
      if (data.length > 0 && !currentTemplate) {
        selectTemplate(data[0]);
      }
    } catch (error) {
      console.error("Failed to fetch templates:", error);
      toast.error("Failed to load templates");
    }
  };

  const selectTemplate = async (name: string) => {
    setCurrentTemplate(name);
    await fetchTemplateContent(name);
    
    if (activeTab === "samples") {
      await fetchSampleData(name);
    }
  };

  const fetchTemplateContent = async (name: string) => {
    try {
      const response = await fetch(`/api/templates/${name}`);
      const content = await response.text();
      setTemplateContent(content);
    } catch (error) {
      console.error(`Failed to fetch template ${name}:`, error);
      toast.error(`Failed to load template ${name}`);
    }
  };

  const fetchSampleData = async (name: string) => {
    try {
      const response = await fetch(`/api/samples/${name}`);
      const data = await response.json();
      setSampleData(data);
    } catch (error) {
      console.error(`Failed to fetch sample data for ${name}:`, error);
      toast.error(`Failed to load sample data for ${name}`);
    }
  };

  const saveTemplate = async () => {
    if (!currentTemplate) return;
    
    try {
      const response = await fetch(`/api/templates/${currentTemplate}`, {
        method: "PUT",
        body: templateContent,
      });
      
      if (response.ok) {
        toast.success(`Template ${currentTemplate} saved successfully`);
      } else {
        throw new Error(`Server returned ${response.status}`);
      }
    } catch (error) {
      console.error(`Failed to save template ${currentTemplate}:`, error);
      toast.error(`Failed to save template ${currentTemplate}`);
    }
  };

  const saveSampleData = async () => {
    if (!currentTemplate) return;
    
    try {
      const response = await fetch(`/api/samples/${currentTemplate}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(sampleData),
      });
      
      if (response.ok) {
        toast.success(`Sample data for ${currentTemplate} saved successfully`);
      } else {
        throw new Error(`Server returned ${response.status}`);
      }
    } catch (error) {
      console.error(`Failed to save sample data for ${currentTemplate}:`, error);
      toast.error(`Failed to save sample data for ${currentTemplate}`);
    }
  };

  const previewTemplate = async () => {
    if (!currentTemplate) return "";
    
    try {
      const response = await fetch(`/api/templates/preview/${currentTemplate}`, {
        method: "POST",
        body: templateContent,
      });
      
      return await response.text();
    } catch (error) {
      console.error(`Failed to preview template ${currentTemplate}:`, error);
      toast.error(`Failed to preview template ${currentTemplate}`);
      return "";
    }
  };

  return (
    <div className="h-screen flex flex-col">
      <header className="bg-zinc-950 border-b border-zinc-800 shadow-sm py-3">
        <div className="px-4 flex justify-between items-center">
          <div className="flex items-center space-x-2">
            <h1 className="text-xl font-semibold text-white text-left">Email Template Manager</h1>
          </div>
          <div className="text-sm px-3 py-1 bg-primary-700/20 text-primary-200 border border-primary-700/20 rounded-full">Development Mode</div>
        </div>
      </header>
      
      <main className="flex-1 overflow-hidden flex">
        {/* Sidebar */}
        <TemplateList 
          templates={templates} 
          currentTemplate={currentTemplate}
          onSelectTemplate={selectTemplate}
        />
        
        {/* Main Content */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Tabs */}
          <div className="border-b border-zinc-800 bg-zinc-950">
            <div className="flex">
              <button 
                className={`whitespace-nowrap py-4 px-6 border-b-2 font-medium text-sm ${
                  activeTab === "editor" 
                    ? "text-zinc-200 border-zinc-500" 
                    : "text-zinc-500 border-transparent hover:text-zinc-400 hover:border-zinc-400"
                }`}
                onClick={() => setActiveTab("editor")}
              >
                Template Editor
              </button>
              <button 
                className={`whitespace-nowrap py-4 px-6 border-b-2 font-medium text-sm ${
                  activeTab === "samples" 
                    ? "text-zinc-200 border-zinc-500" 
                    : "text-zinc-500 border-transparent hover:text-zinc-400 hover:border-zinc-400"
                }`}
                onClick={() => {
                  setActiveTab("samples");
                  if (currentTemplate) {
                    fetchSampleData(currentTemplate);
                  }
                }}
              >
                Sample Data
              </button>
            </div>
          </div>
          
          {/* Editor Section */}
          {activeTab === "editor" ? (
            <div className="flex-1 flex flex-col overflow-hidden">
              <div className="flex justify-between items-center px-4 py-3 border-b border-zinc-800 bg-zinc-950">
                <h2 className="text-lg font-medium text-zinc-200">
                  {currentTemplate || "Select a template"}
                </h2>
                <button 
                  className="px-4 py-2 bg-zinc-700/50 text-zinc-200 text-sm font-medium rounded-md shadow-sm hover:bg-zinc-700/80 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-zinc-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  onClick={saveTemplate}
                  disabled={!currentTemplate}
                >
                  Save
                </button>
              </div>
              
              <div className="flex-1 flex overflow-hidden">
                <TemplateEditor 
                  value={templateContent} 
                  onChange={setTemplateContent}
                />
                <TemplatePreview 
                  currentTemplate={currentTemplate}
                  getPreviewContent={previewTemplate} 
                />
              </div>
            </div>
          ) : (
            <div className="flex-1 flex flex-col overflow-hidden">
              <div className="flex justify-between items-center px-4 py-3 border-b border-zinc-800 bg-zinc-950">
                <h2 className="text-lg font-medium text-zinc-200">
                  Sample Data
                </h2>
                <button 
                  className="px-4 py-2 bg-zinc-700/50 text-zinc-200 text-sm font-medium rounded-md shadow-sm hover:bg-zinc-700/80 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-zinc-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  onClick={saveSampleData}
                  disabled={!currentTemplate}
                >
                  Save
                </button>
              </div>
              <SampleDataEditor 
                value={sampleData} 
                onChange={setSampleData}
              />
            </div>
          )}
        </div>
      </main>
    </div>
  );
} 