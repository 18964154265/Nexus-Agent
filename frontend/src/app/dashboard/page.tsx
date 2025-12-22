export default function DashboardPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">欢迎回来</h1>
      <p className="text-gray-600">
        这是 Nexus Agent 的管理控制台。请从左侧菜单选择功能模块进行操作。
      </p>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-8">
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <h3 className="font-semibold text-lg mb-2">快速开始</h3>
          <p className="text-sm text-gray-500 mb-4">创建一个新的 Agent 并为其分配任务。</p>
          <a href="/dashboard/agents" className="text-emerald-600 text-sm font-medium hover:underline">去管理 Agents &rarr;</a>
        </div>
        
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <h3 className="font-semibold text-lg mb-2">开始对话</h3>
          <p className="text-sm text-gray-500 mb-4">与现有的 Agent 进行交互测试。</p>
          <a href="/dashboard/sessions" className="text-emerald-600 text-sm font-medium hover:underline">去聊天 &rarr;</a>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <h3 className="font-semibold text-lg mb-2">系统状态</h3>
          <p className="text-sm text-gray-500 mb-4">查看 MCP 服务器连接状态和知识库索引。</p>
          <a href="/dashboard/mcp" className="text-emerald-600 text-sm font-medium hover:underline">查看 MCP &rarr;</a>
        </div>
      </div>
    </div>
  );
}

