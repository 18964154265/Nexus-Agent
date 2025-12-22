"use client";

import { useState, useEffect } from "react";
import { Bot, Save, RefreshCw, Send, Plus, X } from "lucide-react";
import { useForm, Controller } from "react-hook-form";
import Link from "next/link";
import { clsx } from "clsx";
import useSWR from "swr";
import { apiClient } from "@/lib/apiClient";
import { useRouter } from "next/navigation";
import Editor from "react-simple-code-editor";
import { highlight, languages } from "prismjs";
import "prismjs/components/prism-json";
import "prismjs/themes/prism.css";
import type { ApiResponse, MCPServer, KnowledgeBase, Agent } from "@/types";

// 定义 JSON 编辑器默认模板
const DEFAULT_SYSTEM_PROMPT_JSON = JSON.stringify({
  role_description: "你是一个智能助手。",
  responsibilities: ["回答用户问题", "提供建议"],
  rules: ["保持礼貌", "实事求是"]
}, null, 2);

const fetcher = (url: string) => apiClient.get<ApiResponse<any[]>>(url).then((res) => res.data || []);

export default function CreateAgentPage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState("basic");
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // 预览区状态
  const [chatMessages, setChatMessages] = useState([
    { role: "assistant", content: "你好！我是当前配置下的预览 Agent。你可以随时发送消息来测试我的回复效果。" }
  ]);
  const [previewInput, setPreviewInput] = useState("");

  // 获取可选数据
  const { data: mcpServers } = useSWR<MCPServer[]>("/api/mcp/servers", fetcher);
  const { data: knowledgeBases } = useSWR<KnowledgeBase[]>("/api/knowledge", fetcher);

  const { register, control, handleSubmit, watch, setValue, formState: { errors } } = useForm({
    defaultValues: {
      name: "",
      description: "",
      tags: [] as string[],
      model_name: "gpt-4o",
      system_prompt: DEFAULT_SYSTEM_PROMPT_JSON,
      temperature: 0.7,
      // MCP & KB
      selected_mcp_servers: [] as string[], // 前端临时字段，用于关联查找 Tools
      knowledge_base_ids: [] as string[],
    }
  });

  const formData = watch();

  // 处理标签输入
  const [tagInput, setTagInput] = useState("");
  const handleAddTag = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && tagInput.trim()) {
      e.preventDefault();
      if (!formData.tags.includes(tagInput.trim())) {
        setValue("tags", [...formData.tags, tagInput.trim()]);
      }
      setTagInput("");
    }
  };
  const removeTag = (tagToRemove: string) => {
    setValue("tags", formData.tags.filter(t => t !== tagToRemove));
  };

  const onSubmit = async (data: any) => {
    setIsSubmitting(true);
    try {
      // 构造最终 payload
      // 注意：这里需要根据 selected_mcp_servers 去后端查找对应的 capabilities (Tools)
      // 但根据后端设计，Agent 创建时可能还没法直接绑定 MCP Server 关系（需要看后端接口定义）
      // 假设我们这里只传基本信息，后续再关联，或者后端支持直接传 extra_config
      
      const payload = {
        name: data.name,
        description: data.description,
        model_name: data.model_name,
        system_prompt: data.system_prompt, // 保持 JSON 字符串格式，或者解析后传给后端（视后端要求而定，通常 string 即可）
        temperature: parseFloat(data.temperature),
        tags: data.tags,
        knowledge_base_ids: data.knowledge_base_ids,
        // 这里假设我们把选中的 MCP Server ID 存到 meta 或 extra_config 里，或者后端有专门字段
        // 暂时先忽略 MCP 的具体绑定逻辑，等待后端确认
        extra_config: {
           mcp_server_ids: data.selected_mcp_servers
        }
      };

      await apiClient.post<ApiResponse<Agent>>("/api/agents", payload);
      router.push("/dashboard/agents");
    } catch (error) {
      console.error(error);
      alert("创建失败，请重试");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handlePreviewSend = () => {
     if (!previewInput.trim()) return;
     const userMsg = { role: "user", content: previewInput };
     setChatMessages(prev => [...prev, userMsg]);
     setPreviewInput("");
     
     // 模拟回复
     setTimeout(() => {
         setChatMessages(prev => [...prev, { role: "assistant", content: `[模拟回复] 我收到了你的消息：${userMsg.content}\n当前模型：${formData.model_name}\n温度：${formData.temperature}` }]);
     }, 600);
  };

  const tabs = [
    { id: "basic", label: "基础信息" },
    { id: "model", label: "模型设定" },
    { id: "capabilities", label: "能力装配 (MCP)" },
    { id: "knowledge", label: "知识库 (RAG)" },
  ];

  return (
    <div className="h-[calc(100vh-8rem)] flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between mb-4 flex-shrink-0">
        <h1 className="text-xl font-bold text-gray-900">新建 Agent</h1>
        <div className="flex space-x-3">
            <Link href="/dashboard/agents" className="px-4 py-2 text-sm text-gray-600 hover:text-gray-900">取消</Link>
            <button 
                onClick={handleSubmit(onSubmit)}
                disabled={isSubmitting}
                className="inline-flex items-center px-4 py-2 bg-emerald-600 text-white text-sm font-medium rounded-md hover:bg-emerald-500 disabled:opacity-50"
            >
                <Save className="h-4 w-4 mr-2" />
                {isSubmitting ? "保存中..." : "保存 Agent"}
            </button>
        </div>
      </div>

      <div className="flex-1 flex gap-6 overflow-hidden">
        {/* Left: Configuration Form (60%) */}
        <div className="w-[60%] flex flex-col bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm">
           {/* Config Tabs */}
           <div className="flex border-b border-gray-200 bg-gray-50 px-2">
              {tabs.map(tab => (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={clsx(
                        "px-4 py-3 text-sm font-medium border-b-2 transition-colors",
                        activeTab === tab.id 
                            ? "border-emerald-600 text-emerald-700 bg-white" 
                            : "border-transparent text-gray-500 hover:text-gray-700 hover:bg-gray-100"
                    )}
                  >
                      {tab.label}
                  </button>
              ))}
           </div>

           {/* Config Content */}
           <div className="flex-1 overflow-y-auto p-6">
                {/* A. 基础信息 */}
                {activeTab === "basic" && (
                    <div className="space-y-6 max-w-lg">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">Agent 名称</label>
                            <input {...register("name", { required: true })} className="block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm p-2 border" placeholder="例如：DevOps 助手" />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">描述</label>
                            <textarea {...register("description")} rows={3} className="block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm p-2 border" placeholder="简要描述这个 Agent 的用途..." />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">标签 (Tags)</label>
                            <div className="flex flex-wrap gap-2 mb-2 p-2 border border-gray-300 rounded-md min-h-[42px] focus-within:ring-1 focus-within:ring-emerald-500 focus-within:border-emerald-500">
                                {formData.tags.map(tag => (
                                    <span key={tag} className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-100 text-emerald-800">
                                        {tag}
                                        <button type="button" onClick={() => removeTag(tag)} className="ml-1 text-emerald-600 hover:text-emerald-900 focus:outline-none">
                                            <X className="h-3 w-3" />
                                        </button>
                                    </span>
                                ))}
                                <input 
                                    type="text" 
                                    className="flex-1 border-none outline-none focus:ring-0 text-sm min-w-[120px]"
                                    placeholder="输入标签并回车..."
                                    value={tagInput}
                                    onChange={(e) => setTagInput(e.target.value)}
                                    onKeyDown={handleAddTag}
                                />
                            </div>
                            <p className="text-xs text-gray-500">输入标签名称后按回车添加。</p>
                        </div>
                    </div>
                )}

                {/* B. 模型设定 */}
                {activeTab === "model" && (
                    <div className="space-y-6">
                        <div className="grid grid-cols-2 gap-6">
                             <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">模型 (Model)</label>
                                <select {...register("model_name")} className="block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm p-2 border">
                                    <option value="gpt-4o">GPT-4o</option>
                                    <option value="claude-3-5-sonnet">Claude 3.5 Sonnet</option>
                                    <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                                </select>
                             </div>
                             <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">温度 (Temperature)</label>
                                <div className="flex items-center space-x-4">
                                    <input {...register("temperature")} type="range" min="0" max="2" step="0.1" className="flex-1 h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer accent-emerald-600" />
                                    <span className="text-sm font-mono text-gray-700 w-8">{formData.temperature}</span>
                                </div>
                             </div>
                        </div>

                        <div className="h-full flex flex-col">
                            <label className="block text-sm font-medium text-gray-700 mb-1">系统提示词 (JSON Config)</label>
                            <p className="text-xs text-gray-500 mb-2">请按照 JSON 格式配置角色详情。</p>
                            <div className="relative flex-1 min-h-[400px] border border-gray-300 rounded-md overflow-hidden bg-white focus-within:ring-1 focus-within:ring-emerald-500 focus-within:border-emerald-500">
                                <Controller
                                    name="system_prompt"
                                    control={control}
                                    render={({ field }) => (
                                        <Editor
                                            value={field.value}
                                            onValueChange={field.onChange}
                                            highlight={(code) => highlight(code, languages.json, "json")}
                                            padding={16}
                                            style={{
                                                fontFamily: '"SF Mono", "Monaco", "Inconsolata", "Roboto Mono", "Consolas", "Courier New", monospace',
                                                fontSize: 14,
                                                lineHeight: 1.5,
                                                backgroundColor: "#ffffff",
                                                color: "#24292f",
                                                minHeight: "400px",
                                                width: "100%",
                                                outline: "none",
                                            }}
                                            textareaClassName="editor-textarea"
                                            preClassName="editor-pre"
                                        />
                                    )}
                                />
                            </div>
                            <style jsx global>{`
                                .editor-textarea {
                                    outline: none !important;
                                    border: none !important;
                                    background: transparent !important;
                                    color: #24292f !important;
                                    caret-color: #0969da !important;
                                }
                                .editor-pre {
                                    background: transparent !important;
                                    margin: 0 !important;
                                    padding: 0 !important;
                                }
                                .editor-pre code {
                                    background: transparent !important;
                                    color: inherit !important;
                                }
                            `}</style>
                        </div>
                    </div>
                )}

                {/* C. 能力装配 (MCP) */}
                {activeTab === "capabilities" && (
                    <div className="space-y-6">
                        <div>
                            <label className="block text-base font-medium text-gray-900 mb-4">选择 MCP 服务器 (Toolbox)</label>
                            {!mcpServers ? (
                                <div className="text-sm text-gray-500">加载中...</div>
                            ) : mcpServers.length === 0 ? (
                                <div className="text-sm text-gray-500">暂无可用 MCP 服务器，请先在 "MCP 服务" 页面注册。</div>
                            ) : (
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    <Controller
                                        name="selected_mcp_servers"
                                        control={control}
                                        render={({ field }) => (
                                            <>
                                                {mcpServers.map((server) => (
                                                    <label key={server.id} className={clsx(
                                                        "relative flex items-start p-4 border rounded-lg cursor-pointer transition-all hover:shadow-md",
                                                        field.value.includes(server.id) ? "border-emerald-500 bg-emerald-50" : "border-gray-200 bg-white"
                                                    )}>
                                                        <div className="flex h-5 items-center">
                                                            <input
                                                                type="checkbox"
                                                                className="h-4 w-4 rounded border-gray-300 text-emerald-600 focus:ring-emerald-600"
                                                                checked={field.value.includes(server.id)}
                                                                onChange={(e) => {
                                                                    if (e.target.checked) {
                                                                        field.onChange([...field.value, server.id]);
                                                                    } else {
                                                                        field.onChange(field.value.filter((id: string) => id !== server.id));
                                                                    }
                                                                }}
                                                            />
                                                        </div>
                                                        <div className="ml-3 text-sm">
                                                            <span className="font-medium text-gray-900">{server.name}</span>
                                                            <p className="text-gray-500 text-xs mt-1">
                                                                {server.is_global ? "全局共享" : "私有服务"} · {server.status}
                                                            </p>
                                                        </div>
                                                    </label>
                                                ))}
                                            </>
                                        )}
                                    />
                                </div>
                            )}
                        </div>

                        <div className="mt-8 border-t border-gray-200 pt-6">
                            <h4 className="text-sm font-medium text-gray-900 mb-2">包含的工具 (Tools) - 只读预览</h4>
                            <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
                                <p className="text-sm text-gray-500 italic">选中服务器后，其包含的工具列表将显示在这里（需后端接口支持）。</p>
                            </div>
                        </div>
                    </div>
                )}

                {/* D. 知识库 (RAG) */}
                {activeTab === "knowledge" && (
                     <div className="space-y-6">
                        <label className="block text-base font-medium text-gray-900 mb-4">关联知识库</label>
                         {!knowledgeBases ? (
                                <div className="text-sm text-gray-500">加载中...</div>
                            ) : knowledgeBases.length === 0 ? (
                                <div className="text-sm text-gray-500">暂无知识库，请先在 "知识库" 页面创建。</div>
                            ) : (
                                <div className="space-y-2">
                                    <Controller
                                        name="knowledge_base_ids"
                                        control={control}
                                        render={({ field }) => (
                                            <>
                                                {knowledgeBases.map((kb) => (
                                                     <label key={kb.id} className="flex items-center p-3 hover:bg-gray-50 rounded-md cursor-pointer">
                                                        <input
                                                            type="checkbox"
                                                            className="h-4 w-4 rounded border-gray-300 text-emerald-600 focus:ring-emerald-600"
                                                            checked={field.value.includes(kb.id)}
                                                            onChange={(e) => {
                                                                if (e.target.checked) {
                                                                    field.onChange([...field.value, kb.id]);
                                                                } else {
                                                                    field.onChange(field.value.filter((id: string) => id !== kb.id));
                                                                }
                                                            }}
                                                        />
                                                        <div className="ml-3">
                                                            <span className="text-sm font-medium text-gray-900">{kb.name}</span>
                                                            <span className="ml-2 text-xs text-gray-500">{kb.description || "无描述"}</span>
                                                        </div>
                                                     </label>
                                                ))}
                                            </>
                                        )}
                                    />
                                </div>
                            )}
                    </div>
                )}
           </div>
        </div>

        {/* Right: Preview Chat (40%) */}
        <div className="w-[40%] flex flex-col bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
             <div className="p-3 border-b border-gray-200 bg-gray-50 flex justify-between items-center">
                 <h3 className="text-sm font-medium text-gray-700 flex items-center">
                     <Bot className="h-4 w-4 mr-2 text-emerald-600"/>
                     调试预览
                 </h3>
                 <button className="text-xs text-gray-500 hover:text-emerald-600 flex items-center">
                     <RefreshCw className="h-3 w-3 mr-1"/> 重置
                 </button>
             </div>
             
             {/* Chat Area */}
             <div className="flex-1 bg-gray-50 p-4 overflow-y-auto space-y-4">
                {chatMessages.map((msg, idx) => (
                    <div key={idx} className={clsx("flex", msg.role === "user" ? "justify-end" : "justify-start")}>
                         <div className={clsx(
                             "max-w-[85%] rounded-lg px-4 py-2 text-sm whitespace-pre-wrap",
                             msg.role === "user" 
                                ? "bg-emerald-600 text-white rounded-br-none" 
                                : "bg-white border border-gray-200 text-gray-800 rounded-bl-none shadow-sm"
                         )}>
                             {msg.content}
                         </div>
                    </div>
                ))}
             </div>

             {/* Input Area */}
             <div className="p-3 border-t border-gray-200 bg-white">
                 <div className="relative">
                     <input 
                        type="text" 
                        value={previewInput}
                        onChange={(e) => setPreviewInput(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handlePreviewSend()}
                        placeholder="在此输入消息进行测试..."
                        className="w-full pl-4 pr-10 py-2 text-sm border border-gray-300 rounded-full focus:outline-none focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                     />
                     <button 
                        onClick={handlePreviewSend}
                        className="absolute right-1.5 top-1.5 p-1 text-emerald-600 hover:bg-emerald-50 rounded-full transition-colors"
                     >
                         <Send className="h-4 w-4" />
                     </button>
                 </div>
                 <p className="text-[10px] text-gray-400 text-center mt-2">预览模式不消耗真实 Token (Mock)，且不会保存历史。</p>
             </div>
        </div>
      </div>
    </div>
  );
}
