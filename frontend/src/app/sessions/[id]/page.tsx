"use client";

import { useState, useRef, useEffect, use } from "react";
import useSWR from "swr";
import { apiClient } from "@/lib/apiClient";
import { fetcher } from "@/lib/fetcher";
import { 
  Send, 
  Brain, 
  Wrench, 
  Bot, 
  CheckCircle2, 
  XCircle, 
  Loader2,
  ChevronDown,
  ChevronRight,
  Copy,
  Check,
  X,
  Terminal,
  Clock
} from "lucide-react";
import { clsx } from "clsx";
import ReactMarkdown from "react-markdown";
import type { ApiResponse, ChatMessage, ChatSession, Run, RunStep, Agent } from "@/types";

// Message interface for UI
interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string; // Markdown text
  run_id?: string; // If role is assistant, this links to the trace
  meta?: {
    status: 'success' | 'failed' | 'running';
    tool_count: number;
    latency_ms: number;
  };
}

// Trace Node interface
interface TraceNode {
  id: string;
  type: 'thought' | 'tool' | 'run';
  name: string;
  status: 'success' | 'failed' | 'running';
  duration: string;
  input?: string; // JSON string
  output?: string; // JSON string or logs
  children?: TraceNode[];
}


interface ChatPageProps {
  params: Promise<{
    id: string;
  }>;
}

export default function ChatPage({ params }: ChatPageProps) {
  const { id } = use(params);
  const sessionId = id;
  const [input, setInput] = useState("");
  const [isSending, setIsSending] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [isTraceDrawerOpen, setIsTraceDrawerOpen] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  
  // 流式消息状态
  const [streamingContent, setStreamingContent] = useState<string>("");
  const [streamingStatus, setStreamingStatus] = useState<string>(""); // "thinking", "tool:xxx", ""
  
  // 乐观更新：立刻显示用户发送的消息
  const [pendingUserMessage, setPendingUserMessage] = useState<string | null>(null);

  // Fetch messages
  const { data: messages, error, isLoading, mutate } = useSWR<ChatMessage[]>(
    sessionId ? `/api/sessions/${sessionId}/messages` : null,
    fetcher
  );

  // Fetch session info
  const { data: session } = useSWR<ChatSession>(
    sessionId ? `/api/sessions/${sessionId}` : null,
    fetcher
  );

  // Fetch current agent info
  const { data: currentAgent } = useSWR<Agent>(
    session?.agent_id ? `/api/agents/${session.agent_id}` : null,
    fetcher
  );

  // Fetch all runs for the current user
  const { data: allRuns } = useSWR<Run[]>(
    `/api/runs`,
    fetcher
  );

  // Create a map of run_id -> Run for quick lookup
  const runsMap = new Map<string, Run>();
  if (allRuns && Array.isArray(allRuns)) {
    allRuns.forEach(run => {
      runsMap.set(run.id, run);
    });
  }

  // Get unique run IDs from messages
  const runIds = Array.from(new Set(
    (messages || [])
      .filter(msg => msg.run_id)
      .map(msg => msg.run_id!)
  ));

  // Fetch run steps for tool count calculation
  const { data: allRunSteps } = useSWR<Record<string, RunStep[]>>(
    runIds.length > 0 
      ? Promise.all(
          runIds.map(async (runId) => {
            try {
              const steps = await fetcher<RunStep[]>(`/api/runs/${runId}/trace`);
              return { runId, steps };
            } catch {
              return { runId, steps: [] };
            }
          })
        ).then(results => {
          const map: Record<string, RunStep[]> = {};
          results.forEach(({ runId, steps }) => {
            map[runId] = steps;
          });
          return map;
        })
      : null
  );

  // Convert ChatMessage to Message format (filter out tool messages)
  const uiMessages: Message[] = (messages || [])
    .filter((msg) => msg.role === 'user' || msg.role === 'assistant')
    .map((msg) => {
      const contentText = typeof msg.content === 'string' 
        ? msg.content 
        : msg.content?.text || JSON.stringify(msg.content);
      
      const run = msg.run_id ? runsMap.get(msg.run_id) : undefined;
      const runSteps = msg.run_id && allRunSteps ? allRunSteps[msg.run_id] || [] : [];
      
      // Calculate latency from Run
      let latency_ms = 0;
      if (run && run.finished_at && run.created_at) {
        const start = new Date(run.created_at).getTime();
        const end = new Date(run.finished_at).getTime();
        latency_ms = end - start;
      }
      
      // Count tools from RunSteps
      const tool_count = runSteps.filter(step => step.step_type === 'tool').length;
      
      // Map Run status to Message meta status
      let metaStatus: 'success' | 'failed' | 'running' = 'success';
      if (run) {
        if (run.status === 'running') metaStatus = 'running';
        else if (run.status === 'failed' || run.status === 'cancelled') metaStatus = 'failed';
        else metaStatus = 'success';
      }
      
      return {
        id: msg.id,
        role: msg.role as 'user' | 'assistant',
        content: contentText,
        run_id: msg.run_id,
        meta: msg.run_id ? {
          status: metaStatus,
          tool_count,
          latency_ms,
        } : undefined,
      };
    });

  // Auto scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [uiMessages.length, streamingContent, pendingUserMessage]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isSending) return;

    const userMessage = input.trim();
    setInput("");
    setIsSending(true);
    setPendingUserMessage(userMessage); // 立刻显示用户消息
    setStreamingContent("");
    setStreamingStatus("thinking");

    try {
      const token = localStorage.getItem("token");
      const apiBaseUrl = process.env.NEXT_PUBLIC_API_URL || "http://127.0.0.1:8888";
      const response = await fetch(`${apiBaseUrl}/api/sessions/${sessionId}/chat/stream`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({ content: userMessage }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error("No reader available");
      }

      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || ""; // 保留不完整的行

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const jsonStr = line.slice(6).trim();
            if (!jsonStr) continue;

            try {
              const event = JSON.parse(jsonStr);
              
              switch (event.type) {
                case "content":
                  setStreamingContent(prev => prev + event.content);
                  setStreamingStatus("");
                  break;
                case "tool_start":
                  setStreamingStatus(`executing: ${event.tool}`);
                  break;
                case "tool_end":
                  setStreamingStatus("");
                  break;
                case "handoff":
                  setStreamingStatus(`transferring to: ${event.agent_id}`);
                  break;
                case "error":
                  console.error("Stream error:", event.content);
                  setStreamingStatus("");
                  break;
                case "done":
                  setStreamingStatus("");
                  break;
              }
            } catch (parseError) {
              console.error("Parse SSE error:", parseError, jsonStr);
            }
          }
        }
      }

      // 流结束，刷新消息列表
      mutate();
      setPendingUserMessage(null); // 清除临时用户消息
      setStreamingContent("");
      setStreamingStatus("");
    } catch (error) {
      console.error("Send message error:", error);
      alert("发送失败，请重试");
      setPendingUserMessage(null); // 出错也清除
    } finally {
      setIsSending(false);
      setStreamingContent("");
      setStreamingStatus("");
    }
  };

  const handleOpenTrace = (runId: string) => {
    setSelectedRunId(runId);
    setIsTraceDrawerOpen(true);
  };

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen bg-gray-50">
        <Loader2 className="h-8 w-8 animate-spin text-emerald-600" />
      </div>
    );
  }

  return (
    <div className="h-screen flex bg-gray-50 overflow-hidden">
      {/* Main Chat Area */}
      <div className={clsx(
        "h-full flex flex-col transition-all duration-300",
        isTraceDrawerOpen ? "w-[calc(100%-600px)] sm:w-[calc(100%-600px)]" : "w-full"
      )}>
        {/* Header */}
        <div className="h-16 bg-white border-b border-gray-200 px-6 flex items-center justify-between flex-none">
          <div className="flex items-center space-x-3">
            <div className="h-10 w-10 bg-emerald-100 rounded-full flex items-center justify-center text-emerald-600 shrink-0">
              <Bot className="h-6 w-6" />
            </div>
            <div>
              <h1 className="text-lg font-semibold text-gray-900">
                {session?.title || "Chat Session"}
              </h1>
              <p className="text-xs text-gray-500">
                {session?.agent_id ? `Agent: ${session.agent_id.slice(0, 8)}...` : "No agent"}
              </p>
            </div>
          </div>
          
          {/* Current Agent Info */}
          <div className="flex items-center gap-2 px-3 py-1.5 bg-emerald-50 border border-emerald-200 rounded-lg">
            <div className="h-2 w-2 bg-emerald-500 rounded-full animate-pulse" />
            <span className="text-sm font-medium text-emerald-700">
              {currentAgent?.name || "Loading..."}
            </span>
          </div>
        </div>

        {/* Messages List - Scrollable */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-6">
          {uiMessages.map((msg) => (
            <div
              key={msg.id}
              className={clsx(
                "flex items-start gap-3",
                msg.role === "user" ? "flex-row-reverse" : "flex-row"
              )}
            >
              {/* Avatar */}
              <div className={clsx(
                "h-8 w-8 rounded-full flex items-center justify-center shrink-0",
                msg.role === "user" 
                  ? "bg-gray-200" 
                  : "bg-gray-200"
              )}>
                {msg.role === "user" ? (
                  <span className="text-gray-900 text-xs font-medium">U</span>
                ) : (
                  <Bot className="h-4 w-4 text-gray-600" />
                )}
              </div>

              {/* Message Bubble */}
              <div className={clsx(
                "flex flex-col max-w-[75%] min-w-0",
                msg.role === "user" ? "items-end" : "items-start"
              )}>
                <div className={clsx(
                  "rounded-2xl px-4 py-3 shadow-sm overflow-hidden w-full",
                  msg.role === "user"
                    ? "bg-white border border-gray-200 text-gray-900"
                    : "bg-white border border-gray-200 text-gray-900"
                )}>
                  {msg.role === "user" ? (
                    <p className="text-sm whitespace-pre-wrap text-gray-900 break-words">{msg.content}</p>
                  ) : (
                    <>
                      <div className="prose prose-sm max-w-none prose-headings:mt-0 prose-headings:mb-2 prose-headings:text-gray-900 prose-p:my-1 prose-p:text-gray-900 prose-ul:my-1 prose-ol:my-1 prose-li:text-gray-900 prose-code:text-sm prose-code:text-gray-900 prose-pre:bg-gray-100 prose-pre:border prose-pre:border-gray-200 prose-pre:overflow-x-auto prose-strong:text-gray-900 prose-a:text-emerald-600 break-words overflow-hidden">
                        <ReactMarkdown>{msg.content}</ReactMarkdown>
                      </div>
                      
                      {/* Trace Footer - Professional Design */}
                      {msg.run_id && msg.meta && (
                        <button
                          onClick={() => handleOpenTrace(msg.run_id!)}
                          className="mt-3 pt-3 border-t border-gray-200 w-full flex items-center justify-between text-xs hover:bg-gray-50 -mx-4 -mb-3 px-4 pb-3 rounded-b-2xl transition-colors group"
                        >
                          <div className="flex items-center gap-2 text-gray-600">
                            <div className={clsx(
                              "h-1.5 w-1.5 rounded-full",
                              msg.meta.status === "success" ? "bg-emerald-500" : 
                              msg.meta.status === "failed" ? "bg-red-500" : 
                              "bg-yellow-500 animate-pulse"
                            )} />
                            <span className="font-medium">
                              {msg.meta.tool_count > 0 ? `Used ${msg.meta.tool_count} Tool${msg.meta.tool_count > 1 ? 's' : ''}` : 'No tools'}
                            </span>
                            <span className="text-gray-400">•</span>
                            <span className="text-gray-500">{formatDuration(msg.meta.latency_ms)}</span>
                          </div>
                          <span className="text-emerald-600 opacity-0 group-hover:opacity-100 transition-opacity font-medium">
                            View Trace →
                          </span>
                        </button>
                      )}
                    </>
                  )}
                </div>
              </div>
            </div>
          ))}

          {/* 乐观更新：立刻显示用户发送的消息 */}
          {pendingUserMessage && (
            <div className="flex items-start gap-3 flex-row-reverse">
              <div className="h-8 w-8 rounded-full flex items-center justify-center shrink-0 bg-gray-200">
                <span className="text-gray-900 text-xs font-medium">U</span>
              </div>
              <div className="flex flex-col max-w-[75%] min-w-0 items-end">
                <div className="rounded-2xl px-4 py-3 shadow-sm overflow-hidden w-full bg-white border border-gray-200 text-gray-900">
                  <p className="text-sm whitespace-pre-wrap text-gray-900 break-words">{pendingUserMessage}</p>
                </div>
              </div>
            </div>
          )}
          
          {/* 流式消息显示 */}
          {(streamingContent || streamingStatus) && (
            <div className="flex items-start gap-3">
              {/* Avatar */}
              <div className="h-8 w-8 rounded-full flex items-center justify-center shrink-0 bg-gray-200">
                <Bot className="h-4 w-4 text-gray-600" />
              </div>

              {/* Message Bubble */}
              <div className="flex flex-col max-w-[75%] min-w-0 items-start">
                <div className="rounded-2xl px-4 py-3 shadow-sm overflow-hidden w-full bg-white border border-gray-200 text-gray-900">
                  {streamingStatus && (
                    <div className="flex items-center gap-2 text-sm text-gray-500 mb-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span>{streamingStatus}</span>
                    </div>
                  )}
                  {streamingContent && (
                    <div className="prose prose-sm max-w-none prose-p:my-1 prose-p:text-gray-900">
                      <ReactMarkdown>{streamingContent}</ReactMarkdown>
                    </div>
                  )}
                  {!streamingContent && !streamingStatus && isSending && (
                    <div className="flex items-center gap-2 text-sm text-gray-500">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span>Thinking...</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
          
          <div ref={messagesEndRef} />
        </div>

        {/* Input Area */}
        <div className="h-20 bg-white border-t border-gray-200 px-6 py-4 flex-none">
          <form onSubmit={handleSend} className="flex items-end gap-3">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  handleSend(e);
                }
              }}
              placeholder="Type your message..."
              rows={1}
              className="flex-1 resize-none px-4 py-2.5 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 text-sm text-gray-900 placeholder:text-gray-400"
              style={{ 
                minHeight: "44px", 
                maxHeight: "120px",
                lineHeight: "1.5"
              }}
            />
            <button
              type="submit"
              disabled={!input.trim() || isSending}
              className={clsx(
                "h-11 w-11 rounded-lg flex items-center justify-center transition-colors shrink-0",
                input.trim() && !isSending
                  ? "bg-emerald-600 text-white hover:bg-emerald-500"
                  : "bg-gray-200 text-gray-400 cursor-not-allowed"
              )}
            >
              {isSending ? (
                <Loader2 className="h-5 w-5 animate-spin" />
              ) : (
                <Send className="h-5 w-5" />
              )}
            </button>
          </form>
        </div>
      </div>

      {/* Trace Detail Sidebar - Sheet/Drawer */}
      {isTraceDrawerOpen && selectedRunId && (
        <TraceDrawer
          runId={selectedRunId}
          onClose={() => {
            setIsTraceDrawerOpen(false);
            setSelectedRunId(null);
          }}
        />
      )}
    </div>
  );
}

// Trace Drawer Component (Sheet)
interface TraceDrawerProps {
  runId: string;
  onClose: () => void;
}

function TraceDrawer({ runId, onClose }: TraceDrawerProps) {
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [copiedId, setCopiedId] = useState<string | null>(null);

  // Fetch trace data
  const { data: traceData, isLoading } = useSWR<RunStep[]>(
    runId ? `/api/runs/${runId}/trace` : null,
    fetcher
  );

  // Convert RunSteps to TraceNode tree (支持 children)
  const convertToTraceNode = (step: any): TraceNode => {
    const node: TraceNode = {
      id: step.id,
      type: (step.step_type === 'thought' || step.step_type === 'tool' || step.step_type === 'run') 
        ? step.step_type 
        : 'thought',
      name: step.name,
      status: (step.status === 'success' || step.status === 'failed' || step.status === 'running')
        ? step.status
        : 'success',
      duration: `${step.latency_ms || 0}ms`,
      input: step.input_payload ? JSON.stringify(step.input_payload, null, 2) : undefined,
      output: step.output_payload ? JSON.stringify(step.output_payload, null, 2) : undefined,
    };
    
    // 递归处理 children
    if (step.children && Array.isArray(step.children) && step.children.length > 0) {
      node.children = step.children.map(convertToTraceNode);
    }
    
    return node;
  };
  
  const traceNodes: TraceNode[] = (traceData || []).map(convertToTraceNode);

  const toggleNode = (id: string) => {
    const newExpanded = new Set(expandedNodes);
    if (newExpanded.has(id)) {
      newExpanded.delete(id);
    } else {
      newExpanded.add(id);
    }
    setExpandedNodes(newExpanded);
  };

  const handleCopy = async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
    } catch (error) {
      alert("复制失败");
    }
  };

  const getIcon = (type: string) => {
    switch (type) {
      case 'thought':
        return <Brain className="h-4 w-4" />;
      case 'tool':
        return <Wrench className="h-4 w-4" />;
      case 'run':
        return <Bot className="h-4 w-4" />;
      default:
        return null;
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle2 className="h-4 w-4 text-emerald-500" />;
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />;
      case 'running':
        return <Loader2 className="h-4 w-4 text-yellow-500 animate-spin" />;
      default:
        return null;
    }
  };

  return (
    <div className="fixed right-0 top-0 bottom-0 w-[50%] sm:w-[600px] bg-white border-l border-gray-200 shadow-2xl flex flex-col z-30">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 shrink-0">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">Execution Trace</h2>
            <p className="text-xs text-gray-500 mt-1 font-mono">Run ID: {runId.slice(0, 16)}...</p>
          </div>
          <button 
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content - Scrollable */}
        <div className="flex-1 overflow-y-auto p-6 space-y-2">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-emerald-600" />
            </div>
          ) : traceNodes.length === 0 ? (
            <div className="text-center py-12 text-gray-500">No trace data available</div>
          ) : (
            traceNodes.map((node) => (
              <TraceNodeItem
                key={node.id}
                node={node}
                isExpanded={expandedNodes.has(node.id)}
                onToggle={() => toggleNode(node.id)}
                onCopy={handleCopy}
                copiedId={copiedId}
                getIcon={getIcon}
                getStatusIcon={getStatusIcon}
                expandedNodes={expandedNodes}
                setExpandedNodes={setExpandedNodes}
              />
            ))
          )}
        </div>
      </div>
  );
}

// Trace Node Item Component (Accordion-like)
interface TraceNodeItemProps {
  node: TraceNode;
  isExpanded: boolean;
  onToggle: () => void;
  onCopy: (text: string, id: string) => void;
  copiedId: string | null;
  getIcon: (type: string) => React.ReactNode;
  getStatusIcon: (status: string) => React.ReactNode;
  expandedNodes: Set<string>;
  setExpandedNodes: (nodes: Set<string>) => void;
}

function TraceNodeItem({
  node,
  isExpanded,
  onToggle,
  onCopy,
  copiedId,
  getIcon,
  getStatusIcon,
  expandedNodes,
  setExpandedNodes,
}: TraceNodeItemProps) {
  const [activeTab, setActiveTab] = useState<'input' | 'output'>('input');

  return (
    <div className={clsx(
      "border rounded-lg overflow-hidden transition-all",
      node.type === 'run' 
        ? "border-blue-200 bg-blue-50/50" 
        : node.type === 'tool'
        ? "border-orange-200 bg-orange-50/30"
        : "border-gray-200 bg-white"
    )}>
      {/* Header - Accordion Trigger */}
      <button
        onClick={onToggle}
        className="w-full px-4 py-3 flex items-center justify-between hover:bg-gray-50/50 transition-colors text-left"
      >
        <div className="flex items-center gap-3 flex-1 min-w-0">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-gray-400 shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-gray-400 shrink-0" />
          )}
          
          <div className={clsx(
            "h-8 w-8 rounded-lg flex items-center justify-center shrink-0",
            node.type === 'thought' ? "bg-purple-100 text-purple-600" :
            node.type === 'tool' ? "bg-orange-100 text-orange-600" :
            "bg-blue-100 text-blue-600"
          )}>
            {getIcon(node.type)}
          </div>
          
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium text-sm text-gray-900 truncate">{node.name}</span>
              {getStatusIcon(node.status)}
            </div>
            <div className="flex items-center gap-2 mt-0.5">
              <Clock className="h-3 w-3 text-gray-400" />
              <span className="text-xs text-gray-500">{node.duration}</span>
            </div>
          </div>
        </div>
      </button>

      {/* Content - Accordion Content */}
      {isExpanded && (
        <div className="border-t border-gray-200 bg-gray-50/50">
          {(node.input || node.output) && (
            <div className="p-4">
              {/* Tabs */}
              <div className="flex gap-1 mb-3 border-b border-gray-200">
                {node.input && (
                  <button
                    onClick={() => setActiveTab('input')}
                    className={clsx(
                      "px-3 py-1.5 text-xs font-medium border-b-2 transition-colors",
                      activeTab === 'input'
                        ? "border-emerald-600 text-emerald-700"
                        : "border-transparent text-gray-500 hover:text-gray-700"
                    )}
                  >
                    Input
                  </button>
                )}
                {node.output && (
                  <button
                    onClick={() => setActiveTab('output')}
                    className={clsx(
                      "px-3 py-1.5 text-xs font-medium border-b-2 transition-colors",
                      activeTab === 'output'
                        ? "border-emerald-600 text-emerald-700"
                        : "border-transparent text-gray-500 hover:text-gray-700"
                    )}
                  >
                    Output
                  </button>
                )}
              </div>

              {/* Content */}
              <div className="relative">
                <pre className="text-xs bg-gray-900 text-gray-100 p-3 rounded overflow-auto max-h-64 font-mono">
                  {activeTab === 'input' ? node.input : node.output}
                </pre>
                <button
                  onClick={() => onCopy(activeTab === 'input' ? node.input! : node.output!, node.id)}
                  className="absolute top-2 right-2 p-1.5 bg-gray-700 hover:bg-gray-600 rounded text-gray-300 transition-colors"
                  title="Copy"
                >
                  {copiedId === node.id ? (
                    <Check className="h-3 w-3" />
                  ) : (
                    <Copy className="h-3 w-3" />
                  )}
                </button>
              </div>
            </div>
          )}

          {/* Children (Nested) */}
          {node.children && node.children.length > 0 && (
            <div className="px-4 pb-4 space-y-2">
              {node.children.map((child) => {
                const toggleChild = () => {
                  const newExpanded = new Set(expandedNodes);
                  if (newExpanded.has(child.id)) {
                    newExpanded.delete(child.id);
                  } else {
                    newExpanded.add(child.id);
                  }
                  setExpandedNodes(newExpanded);
                };
                return (
                  <TraceNodeItem
                    key={child.id}
                    node={child}
                    isExpanded={expandedNodes.has(child.id)}
                    onToggle={toggleChild}
                    onCopy={onCopy}
                    copiedId={copiedId}
                    getIcon={getIcon}
                    getStatusIcon={getStatusIcon}
                    expandedNodes={expandedNodes}
                    setExpandedNodes={setExpandedNodes}
                  />
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
