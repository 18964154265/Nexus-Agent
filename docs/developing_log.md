### backend


### database
“在设计外键策略时，我区分了用户数据和系统资产。
消息记录（Message）属于用户隐私和内容，所以随 Session 级联删除，确保数据干净且合规。
但执行记录（Run）涉及计费核算和安全审计，属于平台的资产。即便用户删除了对话入口，我们仍需保留执行的痕迹（Trace），因此我选择 SET NULL，以防止‘毁灭证据’或‘账单对不齐’的情况发生。”

mcpserver表：其中status字段，记录这个server的健康状态。 (active / disconnected)<br>后台可以有一个巡检任务，定期 Ping 这些 Server，如果挂了就标记为不可用，防止 Agent 调用时报错。

mcpserver处理连接方式信息， mcptools相当于一个缓存，在agent的思考阶段能快速知晓可用工具列表

本项目平台的顶级数据结构是chatsession，一次会话为用户的一次任务，其中会包含多次run,一次run就是这个任务的一个执行step，每个run对应一个agent，一个agent对应若干mcpserver，每个mcpserver对应多个mcptools。
chatmessages的管理，有四个时刻要触发存储：

你的理解非常准确。对于用户来说，chat_messages 就是“聊天记录”；但对于系统来说，它还是 LLM 的“短期记忆（Context）”。
chat_messages 表不仅要存文本，还要存结构化的工具调用信息。
下面我为你详细拆解写入时机和存储内容，这通常分为 4 个关键时刻。
一、 写入时机与流程 (The 4 Moments)
在一个完整的 Agent 任务周期（Run）中，数据库会有 4 次关键写入（Insert）：
1. 用户提问时刻 (User Input)
触发时机： 当后端 API 收到用户请求（Request），在把请求发给 LLM 之前。
存什么？
role: "user"
content: "帮我检查一下 src/main.py"
run_id: 当前生成的 Run ID
作用： 立即落地用户意图，防止服务崩了导致用户输入丢失。
2. Agent 决定调用工具时刻 (Tool Call / Thought)
触发时机： LLM 思考完毕，返回了一个 JSON 说“我要调 read_file”，但工具还没真正执行时。
存什么？
role: "assistant"
content: "好的，我需要读取文件内容..." (可选的思考文本)
tool_calls: [{"id": "call_123", "name": "read_file", "args": "..."}] (关键！)
作用： 记录 Agent 的决策。这条消息对 LLM 非常重要，下次发请求必须带上，否则 LLM 会忘了自己刚才想干嘛。
3. 工具执行完毕时刻 (Tool Output)
触发时机： 你的后端（MCP Client）拿到 Git 或 文件系统的执行结果后。
存什么？
role: "tool"
tool_call_id: "call_123" (必须跟第 2 步的 ID 对应)
content: "File content: import os..." (具体的执行结果)
作用： 让 LLM 看到工具的返回结果。注意：这条消息通常在前端是隐藏的，用户不用看具体的 JSON 返回，但数据库必须存。
4. Agent 最终回复时刻 (Final Answer)
触发时机： LLM 看了工具结果，组织语言回复用户，并结束流式输出（Stream End） 后。
存什么？
role: "assistant"
content: "检查完了，代码里缺少 import os..."
tool_calls: NULL
作用： 这就是用户在界面上看到的最终气泡。

12.7:
完成所有handlers,但mcp,sessions因为还没有实际接入llm,因此留下了一些mock函数，后续补全
todo: 
    openai api文档学习 
    engine.go中的工具调用逻辑完成
to check:所有增删改查的外键检查，级联等
 

12.17:
新增trace.go模块，负责追踪调用链
为此，在架构中增加“协作式取消”方法