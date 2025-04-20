import { MetaTags } from "@/components/meta-tags";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Icon } from "@/components/ui/icons";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import {
  faArrowDown,
  faArrowUp,
  faChartLine,
  faCircle,
  faSearch,
  faTrash,
} from "@fortawesome/pro-regular-svg-icons";
import { useEffect, useRef, useState } from "react";
import { ParsedLog, VirtualLogViewer } from "./_components/log-viewer";

interface LogEntry {
  level: string;
  app: string;
  hostname: string;
  message: string;
  caller: string;
  time: string;
  version: string;
  goVersion: string;
  goRoutines: number;
  cpu: number;
  service: string;
  component: string;
  client: string;
  batchSize: number;
  batchInterval: number;
  indexPrefix: string;
  indexName: string;
  listenAddress: string;
  metadata: Record<string, any>;
  method: string;
  path: string;
}

interface LogStats {
  total: number;
  byLevel: Record<string, number>;
  byService: Record<string, number>;
}

function formatLogEntry(entry: LogEntry): ParsedLog {
  return {
    timestamp: new Date(entry.time).toLocaleTimeString(),
    level: entry.level,
    service: entry.service,
    message: entry.message,
    caller: entry.caller,
    method: entry.method,
    path: entry.path,
  };
}

export function LogReader() {
  const [logs, setLogs] = useState<ParsedLog[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [levelFilter, setLevelFilter] = useState<string>("all");
  const [serviceFilter, setServiceFilter] = useState<string>("all");
  const [autoScroll, setAutoScroll] = useState(true);
  const [stats, setStats] = useState<LogStats>({
    total: 0,
    byLevel: {},
    byService: {},
  });
  const wsRef = useRef<WebSocket | null>(null);

  function disconnectFromLogs() {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }

  function clearLogs() {
    setLogs([]);
    setStats({
      total: 0,
      byLevel: {},
      byService: {},
    });
  }

  function updateStats(entry: LogEntry) {
    setStats((prev) => {
      const newStats = { ...prev };
      newStats.total++;
      newStats.byLevel[entry.level] = (newStats.byLevel[entry.level] || 0) + 1;
      if (entry.service) {
        newStats.byService[entry.service] =
          (newStats.byService[entry.service] || 0) + 1;
      }
      return newStats;
    });
  }

  function connectToLogs() {
    if (wsRef.current) {
      disconnectFromLogs();
    }

    const ws = new WebSocket("ws://localhost:3001/api/v1/logs/stream");
    wsRef.current = ws;

    ws.onopen = () => {
      console.log("Connected to log stream");
      setIsConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        const logEntry: LogEntry = JSON.parse(event.data);
        console.log("Received log entry: ", logEntry);

        // Apply filters
        if (
          (levelFilter === "all" || logEntry.level === levelFilter) &&
          (serviceFilter === "all" || logEntry.service === serviceFilter)
        ) {
          const formattedLog = formatLogEntry(logEntry);
          setLogs((prevLogs) => [...prevLogs, formattedLog]);
        }

        updateStats(logEntry);
      } catch (err) {
        console.error("Error parsing log entry:", err);
      }
    };

    ws.onclose = () => {
      console.log("Disconnected from log stream");
      setIsConnected(false);
      wsRef.current = null;
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };
  }

  useEffect(() => {
    return () => {
      disconnectFromLogs();
    };
  }, []);

  return (
    <>
      <MetaTags title="Log Reader" description="Log Reader" />
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold">System Logs</h1>
            <Badge variant="outline" className="ml-2">
              <Icon
                icon={faCircle}
                className={`size-2 mr-1 ${isConnected ? "text-green-500" : "text-red-500"}`}
              />
              {isConnected ? "Connected" : "Disconnected"}
            </Badge>
          </div>
          <div className="flex gap-2">
            <Button onClick={clearLogs} variant="outline" size="sm">
              <Icon icon={faTrash} className="size-4 mr-2" />
              Clear
            </Button>
            <Button
              onClick={isConnected ? disconnectFromLogs : connectToLogs}
              variant={isConnected ? "destructive" : "default"}
              size="sm"
            >
              {isConnected ? "Disconnect" : "Connect to Logs"}
            </Button>
          </div>
        </div>

        <div className="grid grid-cols-4 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Total Logs</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.total}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Error Rate</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {(
                  ((stats.byLevel.error || 0) / (stats.total || 1)) *
                  100
                ).toFixed(1)}
                %
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">
                Active Services
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {Object.keys(stats.byService).length}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Log Rate</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                <Icon
                  icon={faChartLine}
                  className="size-4 mr-2 text-green-500"
                />
                {Math.round(stats.total / (isConnected ? 1 : 60))} /min
              </div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Log Stream</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex gap-4 mb-4">
              <div className="flex-1">
                <Label htmlFor="search">Search Logs</Label>
                <div className="relative">
                  <Input
                    id="search"
                    placeholder="Search in logs..."
                    className="pl-8"
                    icon={<Icon icon={faSearch} className="size-3" />}
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                  />
                </div>
              </div>
              <div>
                <Label htmlFor="level">Log Level</Label>
                <Select value={levelFilter} onValueChange={setLevelFilter}>
                  <SelectTrigger id="level" className="w-[180px]">
                    <SelectValue placeholder="Select Level" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Levels</SelectItem>
                    <SelectItem value="debug">Debug</SelectItem>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warn">Warning</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="service">Service</Label>
                <Select value={serviceFilter} onValueChange={setServiceFilter}>
                  <SelectTrigger id="service" className="w-[180px]">
                    <SelectValue placeholder="Select Service" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Services</SelectItem>
                    {Object.keys(stats.byService).map((service) => (
                      <SelectItem key={service} value={service}>
                        {service}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex items-end">
                <Button
                  variant="outline"
                  onClick={() => setAutoScroll(!autoScroll)}
                  className={autoScroll ? "bg-primary/10" : ""}
                >
                  <Icon
                    icon={autoScroll ? faArrowUp : faArrowDown}
                    className="size-4 mr-2"
                  />
                  Auto-scroll
                </Button>
              </div>
            </div>

            <Separator className="my-4" />

            <div className="border rounded-lg overflow-hidden bg-muted">
              <VirtualLogViewer
                logs={logs}
                autoScroll={autoScroll}
                searchTerm={searchTerm}
              />
            </div>

            {logs.length === 0 && !isConnected && (
              <Alert className="mt-4">
                <AlertDescription>
                  No logs to display. Connect to the log stream to start
                  monitoring.
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>
      </div>
    </>
  );
}
