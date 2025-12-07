package handler

import (
	"context"
	"fmt"

	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/store"
	"example.com/agent-server/pkg/response" // 引入统一响应包
	"github.com/cloudwego/hertz/pkg/app"
)

// ==========================================
// DTOs
// ==========================================

type CreateKBReq struct {
	Name        string                 `json:"name" vd:"required"`
	Description string                 `json:"description"`
	IsPublic    bool                   `json:"is_public"`
	MetaInfo    map[string]interface{} `json:"meta_info"` // e.g. {"domain": "finance"}
}

type KBResp struct {
	*store.KnowledgeBase
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// 额外展示的信息
	DocumentCount int `json:"document_count"` // 文档数量
}

// UploadDocResp 上传后的返回结果
type UploadDocResp struct {
	Filename      string `json:"filename"`
	Size          int64  `json:"size"`
	ChunksCreated int    `json:"chunks_created"` // 切片数量
	Status        string `json:"status"`
}

// ==========================================
// Helpers
// ==========================================

func toKBResp(kb *store.KnowledgeBase) *KBResp {
	// 从 MetaInfo 里尝试提取文档数量 (Mock逻辑)
	docCount := 0
	if val, ok := kb.MetaInfo["doc_count"]; ok {
		if c, ok := val.(int); ok {
			docCount = c
		} else if c, ok := val.(float64); ok {
			docCount = int(c)
		}
	}

	return &KBResp{
		KnowledgeBase: kb,
		CreatedAt:     kb.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     kb.UpdatedAt.Format("2006-01-02 15:04:05"),
		DocumentCount: docCount,
	}
}

// ==========================================
// Handlers
// ==========================================

// ListKnowledgeBases 列出知识库
func (h *Handler) ListKnowledgeBases(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	kbs := h.Store.ListKnowledgeBasesByUser(userID)

	res := make([]*KBResp, 0, len(kbs))
	for _, k := range kbs {
		res = append(res, toKBResp(k))
	}

	response.Success(ctx, res)
}

// CreateKnowledgeBase 创建知识库
func (h *Handler) CreateKnowledgeBase(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	var req CreateKBReq
	if err := ctx.BindAndValidate(&req); err != nil {
		response.BadRequest(ctx, "参数错误: "+err.Error())
		return
	}

	// 默认 MetaInfo
	if req.MetaInfo == nil {
		req.MetaInfo = make(map[string]interface{})
	}
	req.MetaInfo["doc_count"] = 0 // 初始化文档计数

	kb := &store.KnowledgeBase{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		MetaInfo:    req.MetaInfo,
		// ID, CreatedAt, UpdatedAt 由 Store 处理
	}

	createdKB := h.Store.CreateKnowledgeBase(kb)

	response.Created(ctx, toKBResp(createdKB))
}

// UploadDocument 上传文档并进行 Embedding (核心 RAG 入口)
// POST /api/knowledge/:id/documents
// Content-Type: multipart/form-data; boundary=...
func (h *Handler) UploadDocument(c context.Context, ctx *app.RequestContext) {
	// 1. 鉴权与参数校验
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	kbID := ctx.Param("id")
	kb := h.Store.GetKnowledgeBase(kbID)
	if kb == nil {
		response.Error(ctx, 404, 40400, "知识库不存在")
		return
	}
	// 权限检查
	if kb.UserID != userID {
		response.Error(ctx, 403, 40300, "无权操作此知识库")
		return
	}

	// 2. 获取上传的文件
	fileHeader, err := ctx.FormFile("file") // 前端 FormData 的 key 必须是 "file"
	if err != nil {
		response.BadRequest(ctx, "请上传文件: "+err.Error())
		return
	}

	// 打开文件流 (真实场景需要读取内容)
	file, err := fileHeader.Open()
	if err != nil {
		response.ServerError(ctx, err)
		return
	}
	defer file.Close()

	// =======================================================
	// 3. Mock RAG Pipeline (模拟处理过程)
	// =======================================================
	// 真实逻辑：
	// a. text := parseFile(file) (PDF/Word转文本)
	// b. chunks := splitText(text) (LangChain切片)
	// c. embeddings := openAI.Embed(chunks)
	// d. chromaClient.Add(embeddings)

	// 模拟耗时 (让前端感觉在处理)
	// time.Sleep(500 * time.Millisecond)

	// 模拟计算出的切片数量 (假设 1KB = 1 Chunk)
	mockChunks := int(fileHeader.Size / 1024)
	if mockChunks < 1 {
		mockChunks = 1
	}

	logMsg := fmt.Sprintf("Mock RAG: Processed file %s (%d bytes) into %d chunks.",
		fileHeader.Filename, fileHeader.Size, mockChunks)
	fmt.Println(logMsg) // 打印到控制台，方便你调试

	// =======================================================

	// 4. 更新知识库元数据 (文档数 +1)
	h.Store.UpdateKnowledgeBase(kbID, func(k *store.KnowledgeBase) {
		if k.MetaInfo == nil {
			k.MetaInfo = make(map[string]interface{})
		}
		// 简单的计数器
		currentCount := 0
		if val, ok := k.MetaInfo["doc_count"]; ok {
			if c, ok := val.(int); ok {
				currentCount = c
			}
		}
		k.MetaInfo["doc_count"] = currentCount + 1

		// 记录最后上传的文件名
		k.MetaInfo["last_file"] = fileHeader.Filename
	})

	// 5. 返回结果
	response.Success(ctx, UploadDocResp{
		Filename:      fileHeader.Filename,
		Size:          fileHeader.Size,
		ChunksCreated: mockChunks,
		Status:        "indexed", // 已索引
	})
}
