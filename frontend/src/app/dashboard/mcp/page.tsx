"use client";

import { useState } from "react";
import useSWR from "swr";
import { apiClient } from "@/lib/apiClient";
import { 
  Plus, 
  RefreshCw, 
  Eye, 
  Trash2, 
  GitBranch, 
  Database, 
  Terminal,
  Loader2,
  X
} from "lucide-react";
import { clsx } from "clsx";
import type { ApiResponse, MCPServer, MCPTool } from "@/types";

const fetcher = (url: string) => apiClient.get<ApiResponse<MCPServer[]>>(url).then((res) => res.data || []);

type ServerStatus = 'active' | 'error' | 'connecting';
type TransportType = 'stdio' | 'sse';
type FilterType = 'all' | 'system' | 'private';

// Mock data for demonstration
const mockServers: MCPServer[] = [
  {
    id: "1",
    name: "git-server",
    status: "active",
    transport_type: "stdio",
    is_global: true,
    tool_count: 8,
    connection_config: {
      command: "uv",
      args: ["run", "mcp-server-git", "--repository", "."],
    },
    created_at: new Date().toISOString(),
  },
  {
    id: "2",
    name: "postgres-db",
    status: "active",
    transport_type: "sse",
    is_global: false,
    tool_count: 12,
    connection_config: {
      url: "https://api.example.com/mcp/postgres",
    },
    created_at: new Date().toISOString(),
  },
  {
    id: "3",
    name: "k8s-controller",
    status: "error",
    transport_type: "stdio",
    is_global: true,
    tool_count: 0,
    connection_config: {
      command: "kubectl",
      args: ["--server", "https://k8s.example.com"],
    },
    created_at: new Date().toISOString(),
  },
];

export default function MCPServersPage() {
  const [filter, setFilter] = useState<FilterType>("all");
  const [isRegisterOpen, setIsRegisterOpen] = useState(false);
  const [selectedServer, setSelectedServer] = useState<MCPServer | null>(null);
  const [isToolDrawerOpen, setIsToolDrawerOpen] = useState(false);
  const [syncingServerId, setSyncingServerId] = useState<string | null>(null);

  // TODO: Replace with real API call
  // const { data: servers, error, isLoading, mutate } = useSWR<MCPServer[]>("/api/mcp/servers", fetcher);
  const servers = mockServers;
  const isLoading = false;

  const filteredServers = servers?.filter((server) => {
    if (filter === "system") return server.is_global;
    if (filter === "private") return !server.is_global;
    return true;
  }) || [];

  const handleSync = async (serverId: string) => {
    setSyncingServerId(serverId);
    try {
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 2000));
      // TODO: await apiClient.post(`/api/mcp/servers/${serverId}/sync`);
      alert("同步成功！");
    } catch (error) {
      alert("同步失败");
    } finally {
      setSyncingServerId(null);
    }
  };

  const handleViewTools = (server: MCPServer) => {
    setSelectedServer(server);
    setIsToolDrawerOpen(true);
  };

  const handleDelete = async (serverId: string) => {
    if (!confirm("确定要删除这个 MCP 服务器吗？")) return;
    try {
      await apiClient.delete(`/api/mcp/servers/${serverId}`);
      alert("删除成功");
    } catch (error) {
      alert("删除失败");
    }
  };

  const getServerIcon = (name: string) => {
    const lowerName = name.toLowerCase();
    if (lowerName.includes("git")) return <GitBranch className="h-5 w-5" />;
    if (lowerName.includes("sql") || lowerName.includes("db") || lowerName.includes("postgres")) return <Database className="h-5 w-5" />;
    return <Terminal className="h-5 w-5" />;
  };

  const getStatusColor = (status: ServerStatus) => {
    switch (status) {
      case "active":
        return "bg-emerald-500";
      case "error":
        return "bg-red-500";
      case "connecting":
        return "bg-yellow-500";
      default:
        return "bg-gray-400";
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">MCP Servers</h1>
          <p className="text-sm text-gray-500 mt-1">Manage external tool connections for your Agents</p>
        </div>
        <button
          onClick={() => setIsRegisterOpen(true)}
          className="inline-flex items-center px-4 py-2 bg-emerald-600 text-white text-sm font-medium rounded-md hover:bg-emerald-500 transition-colors"
        >
          <Plus className="h-4 w-4 mr-2" />
          Register Server
        </button>
      </div>

      {/* Filter Tabs */}
      <div className="flex space-x-1 border-b border-gray-200">
        {(["all", "system", "private"] as FilterType[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setFilter(tab)}
            className={clsx(
              "px-4 py-2 text-sm font-medium border-b-2 transition-colors",
              filter === tab
                ? "border-emerald-600 text-emerald-700"
                : "border-transparent text-gray-500 hover:text-gray-700"
            )}
          >
            {tab === "all" ? "All" : tab === "system" ? "System" : "Private"}
          </button>
        ))}
      </div>

      {/* Server Grid */}
      {isLoading ? (
        <div className="text-center py-12 text-gray-500">加载中...</div>
      ) : filteredServers.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg border border-dashed border-gray-300">
          <Terminal className="h-12 w-12 text-gray-300 mx-auto mb-3" />
          <h3 className="text-sm font-medium text-gray-900">还没有 MCP 服务器</h3>
          <p className="text-sm text-gray-500 mt-1">点击右上角按钮注册一个新的服务器</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredServers.map((server) => (
            <MCPServerCard
              key={server.id}
              server={server}
              onSync={() => handleSync(server.id)}
              onViewTools={() => handleViewTools(server)}
              onDelete={() => handleDelete(server.id)}
              isSyncing={syncingServerId === server.id}
              getServerIcon={getServerIcon}
              getStatusColor={getStatusColor}
            />
          ))}
        </div>
      )}

      {/* Register Modal */}
      {isRegisterOpen && (
        <RegisterModal onClose={() => setIsRegisterOpen(false)} />
      )}

      {/* Tool Drawer */}
      {isToolDrawerOpen && selectedServer && (
        <ToolDrawer
          server={selectedServer}
          onClose={() => setIsToolDrawerOpen(false)}
        />
      )}
    </div>
  );
}

// MCPServerCard Component
interface MCPServerCardProps {
  server: MCPServer;
  onSync: () => void;
  onViewTools: () => void;
  onDelete: () => void;
  isSyncing: boolean;
  getServerIcon: (name: string) => React.ReactNode;
  getStatusColor: (status: ServerStatus) => string;
}

function MCPServerCard({
  server,
  onSync,
  onViewTools,
  onDelete,
  isSyncing,
  getServerIcon,
  getStatusColor,
}: MCPServerCardProps) {
  const configDisplay = server.connection_config?.url 
    ? server.connection_config.url
    : `${server.connection_config?.command || ""} ${(server.connection_config?.args || []).join(" ")}`;

  return (
    <div className="group relative bg-white border border-gray-200 rounded-xl p-6 hover:shadow-lg transition-all">
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center space-x-3">
          <div className="h-10 w-10 bg-emerald-100 rounded-lg flex items-center justify-center text-emerald-600">
            {getServerIcon(server.name)}
          </div>
          <div>
            <h3 className="text-base font-semibold text-gray-900">{server.name}</h3>
            <div className="flex items-center space-x-2 mt-1">
              <span className={clsx(
                "inline-flex items-center px-2 py-0.5 rounded text-xs font-medium",
                server.is_global 
                  ? "bg-blue-100 text-blue-800" 
                  : "bg-gray-100 text-gray-800"
              )}>
                {server.is_global ? "System" : "Private"}
              </span>
              <div className="flex items-center space-x-1">
                <div className={clsx("h-2 w-2 rounded-full", getStatusColor(server.status as ServerStatus))} />
                <span className="text-xs text-gray-500 capitalize">{server.status}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Body */}
      <div className="space-y-3 mb-4">
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-500">Transport</span>
          <span className="font-mono text-xs text-gray-700 uppercase">{server.transport_type}</span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-500">Tools</span>
          <span className="font-medium text-gray-900">{server.tool_count} Tools</span>
        </div>
        <div className="pt-2 border-t border-gray-100">
          <p className="text-xs text-gray-500 mb-1">Connection</p>
          <code className="block text-xs font-mono bg-gray-50 p-2 rounded border border-gray-200 text-gray-700 truncate">
            {configDisplay}
          </code>
        </div>
      </div>

      {/* Footer Actions */}
      <div className="flex items-center justify-end space-x-2 pt-4 border-t border-gray-100">
        <button
          onClick={onSync}
          disabled={isSyncing}
          className="p-2 text-gray-400 hover:text-emerald-600 hover:bg-emerald-50 rounded-md transition-colors disabled:opacity-50"
          title="同步工具"
        >
          {isSyncing ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
        </button>
        <button
          onClick={onViewTools}
          className="p-2 text-gray-400 hover:text-emerald-600 hover:bg-emerald-50 rounded-md transition-colors"
          title="查看工具"
        >
          <Eye className="h-4 w-4" />
        </button>
        {!server.is_global && (
          <button
            onClick={onDelete}
            className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
            title="删除"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}

// Register Modal Component
interface RegisterModalProps {
  onClose: () => void;
}

function RegisterModal({ onClose }: RegisterModalProps) {
  const [transportType, setTransportType] = useState<TransportType>("stdio");
  const [formData, setFormData] = useState({
    name: "",
    command: "",
    args: "",
    url: "",
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload: any = {
        name: formData.name,
        transport_type: transportType,
        connection_config: {},
      };

      if (transportType === "stdio") {
        payload.connection_config.command = formData.command;
        payload.connection_config.args = formData.args.split(" ").filter(Boolean);
      } else {
        payload.connection_config.url = formData.url;
      }

      await apiClient.post<ApiResponse<MCPServer>>("/api/mcp/servers", payload);
      onClose();
      setFormData({ name: "", command: "", args: "", url: "" });
      setTransportType("stdio");
      alert("注册成功！");
    } catch (error) {
      alert("注册失败，请重试");
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50" onClick={onClose}>
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md mx-4" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Register MCP Server</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Server Name</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="e.g., git-server"
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Transport Type</label>
            <div className="flex space-x-4">
              <label className="flex items-center space-x-2 cursor-pointer">
                <input
                  type="radio"
                  value="stdio"
                  checked={transportType === "stdio"}
                  onChange={(e) => setTransportType(e.target.value as TransportType)}
                  className="text-emerald-600 focus:ring-emerald-500"
                />
                <span className="text-sm">STDIO</span>
              </label>
              <label className="flex items-center space-x-2 cursor-pointer">
                <input
                  type="radio"
                  value="sse"
                  checked={transportType === "sse"}
                  onChange={(e) => setTransportType(e.target.value as TransportType)}
                  className="text-emerald-600 focus:ring-emerald-500"
                />
                <span className="text-sm">SSE</span>
              </label>
            </div>
          </div>

          {transportType === "stdio" ? (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Command</label>
                <input
                  type="text"
                  value={formData.command}
                  onChange={(e) => setFormData({ ...formData, command: e.target.value })}
                  placeholder="e.g., uv"
                  required
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Arguments (space-separated)</label>
                <input
                  type="text"
                  value={formData.args}
                  onChange={(e) => setFormData({ ...formData, args: e.target.value })}
                  placeholder="e.g., run mcp-server-git --repository ."
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
                />
              </div>
            </>
          ) : (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">URL</label>
              <input
                type="url"
                value={formData.url}
                onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                placeholder="https://api.example.com/mcp"
                required
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
              />
            </div>
          )}

          <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-4 py-2 text-sm font-medium text-white bg-emerald-600 rounded-md hover:bg-emerald-500"
            >
              Save
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// Tool Drawer Component
interface ToolDrawerProps {
  server: MCPServer;
  onClose: () => void;
}

function ToolDrawer({ server, onClose }: ToolDrawerProps) {
  const [expandedTool, setExpandedTool] = useState<string | null>(null);
  
  // Mock tools data - in real app, fetch from API
  const mockTools: MCPTool[] = [
    { 
      id: "1",
      server_id: server.id,
      name: "git_status", 
      description: "Get the status of the git repository", 
      input_schema: { type: "object", properties: {} } 
    },
    { 
      id: "2",
      server_id: server.id,
      name: "git_commit", 
      description: "Create a new commit", 
      input_schema: { type: "object", properties: { message: { type: "string" } } } 
    },
  ];

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black bg-opacity-50" onClick={onClose}>
      <div 
        className="bg-white w-full sm:max-w-lg h-full shadow-xl overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-6 border-b border-gray-200 sticky top-0 bg-white">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">{server.name} Tools</h2>
            <p className="text-sm text-gray-500 mt-1">{server.tool_count || 0} tools available</p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="p-6 space-y-2">
          {mockTools.length === 0 ? (
            <p className="text-sm text-gray-500 text-center py-8">暂无工具</p>
          ) : (
            mockTools.map((tool) => (
              <div
                key={tool.id}
                className="border border-gray-200 rounded-lg p-4 cursor-pointer hover:bg-gray-50 transition-colors"
                onClick={() => setExpandedTool(expandedTool === tool.name ? null : tool.name)}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <h4 className="font-medium text-sm text-gray-900">{tool.name}</h4>
                    <p className="text-xs text-gray-500 mt-1">{tool.description}</p>
                  </div>
                </div>
                {expandedTool === tool.name && (
                  <div className="mt-3 pt-3 border-t border-gray-200">
                    <p className="text-xs text-gray-500 mb-2">Input Schema:</p>
                    <pre className="text-xs bg-gray-900 text-gray-100 p-3 rounded overflow-auto">
                      {JSON.stringify(tool.input_schema, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
