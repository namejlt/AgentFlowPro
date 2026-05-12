package app

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/namejlt/AgentFlowPro/internal/model"
)

// Seed initializes the database with high-quality built-in data.
// It creates admin user, LLM models, data sources, agents, workflows, and system configs.
// All UUIDs are fixed for predictable relationships between entities.
func (a *App) Seed(ctx context.Context) error {
	var count int64
	if err := a.DB.WithContext(ctx).Model(&model.User{}).Where("email = ?", "admin@agentflow.local").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	adminID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u := model.User{
		ID:           adminID,
		Username:     "admin",
		Email:        "admin@agentflow.local",
		PasswordHash: string(hash),
		Role:         "admin",
	}
	if err := a.DB.WithContext(ctx).Create(&u).Error; err != nil {
		return err
	}

	// Seed LLM Models (placeholder API keys, user must configure real ones)
	if err := seedLLMModels(ctx, a.DB, adminID); err != nil {
		return err
	}

	// Seed DataSources (real production-ready endpoints)
	if err := seedDataSources(ctx, a.DB, adminID); err != nil {
		return err
	}

	// Seed Agents (linked to models and datasources)
	if err := seedAgents(ctx, a.DB, adminID); err != nil {
		return err
	}

	// Seed Workflows (linked to agents and default model)
	if err := seedWorkflows(ctx, a.DB, adminID); err != nil {
		return err
	}

	// Seed System Config
	if err := seedSystemConfig(ctx, a.DB); err != nil {
		return err
	}

	return nil
}

// seedLLMModels creates built-in LLM model configurations.
// All models use placeholder API keys that must be replaced by the user.
func seedLLMModels(ctx context.Context, db *gorm.DB, ownerID uuid.UUID) error {
	models := []model.LLMModel{
		{
			ID:              uuid.MustParse("00000000-0000-0000-0000-000000000101"),
			Name:            "OpenAI GPT-4o",
			Vendor:          "OpenAI",
			Endpoint:        "https://api.openai.com/v1/chat/completions",
			ModelID:         "gpt-4o",
			APIKeyEncrypted: "[PLEASE_SET_REAL_OPENAI_API_KEY]",
			Temperature:     0.7,
			MaxTokens:       4096,
			TimeoutMS:       60000,
			RetryCount:      3,
			StreamEnabled:   true,
			Enabled:         false,
			IsDefault:       false,
			CreatedBy:       &ownerID,
		},
		{
			ID:              uuid.MustParse("00000000-0000-0000-0000-000000000102"),
			Name:            "OpenAI GPT-4o-mini",
			Vendor:          "OpenAI",
			Endpoint:        "https://api.openai.com/v1/chat/completions",
			ModelID:         "gpt-4o-mini",
			APIKeyEncrypted: "[PLEASE_SET_REAL_OPENAI_API_KEY]",
			Temperature:     0.7,
			MaxTokens:       4096,
			TimeoutMS:       60000,
			RetryCount:      3,
			StreamEnabled:   true,
			Enabled:         false,
			IsDefault:       false,
			CreatedBy:       &ownerID,
		},
		{
			ID:              uuid.MustParse("00000000-0000-0000-0000-000000000103"),
			Name:            "DeepSeek-V3",
			Vendor:          "DeepSeek",
			Endpoint:        "https://api.deepseek.com/v1/chat/completions",
			ModelID:         "deepseek-chat",
			APIKeyEncrypted: "[PLEASE_SET_REAL_DEEPSEEK_API_KEY]",
			Temperature:     0.7,
			MaxTokens:       4096,
			TimeoutMS:       60000,
			RetryCount:      3,
			StreamEnabled:   true,
			Enabled:         false,
			IsDefault:       false,
			CreatedBy:       &ownerID,
		},
		{
			ID:              uuid.MustParse("00000000-0000-0000-0000-000000000104"),
			Name:            "Claude 3.5 Sonnet",
			Vendor:          "Anthropic",
			Endpoint:        "https://api.anthropic.com/v1/messages",
			ModelID:         "claude-3-5-sonnet-20241022",
			APIKeyEncrypted: "[PLEASE_SET_REAL_ANTHROPIC_API_KEY]",
			Temperature:     0.7,
			MaxTokens:       4096,
			TimeoutMS:       60000,
			RetryCount:      3,
			StreamEnabled:   true,
			Enabled:         false,
			IsDefault:       false,
			CreatedBy:       &ownerID,
		},
		{
			ID:              uuid.MustParse("00000000-0000-0000-0000-000000000105"),
			Name:            "通义千问 Qwen-Max",
			Vendor:          "Alibaba",
			Endpoint:        "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
			ModelID:         "qwen-max",
			APIKeyEncrypted: "[PLEASE_SET_REAL_DASHSCOPE_API_KEY]",
			Temperature:     0.7,
			MaxTokens:       4096,
			TimeoutMS:       60000,
			RetryCount:      3,
			StreamEnabled:   true,
			Enabled:         false,
			IsDefault:       false,
			CreatedBy:       &ownerID,
		},
	}
	for _, m := range models {
		if err := db.WithContext(ctx).Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedDataSources creates built-in data sources with real production-ready endpoints.
// HTTP_GET sources use publicly available APIs, MANUAL_INPUT sources allow user-provided data.
func seedDataSources(ctx context.Context, db *gorm.DB, ownerID uuid.UUID) error {
	sources := []model.DataSource{
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000201"),
			OwnerID:     ownerID,
			Name:        "股票实时行情",
			Description: strPtr("获取指定股票代码的实时行情数据，支持 A 股、港股、美股。数据源：腾讯财经"),
			Category:    strPtr("finance"),
			Tags:        mustJSON([]string{"finance", "stock", "realtime"}),
			Icon:        strPtr("TrendCharts"),
			DSType:      "HTTP_GET",
			URLTemplate: strPtr("https://qt.gtimg.cn/q={{stock_code}}"),
			HTTPMethod:  strPtr("GET"),
			ContentType: strPtr("text/plain"),
			TimeoutMS:   10000,
			RetryCount:  2,
			AuthType:    "none",
			ParamsSchema: mustJSON([]map[string]interface{}{
				{"name": "stock_code", "type": "string", "required": true, "default_value": "sh600519", "description": "股票代码，如 sh600519、sz000001、hk00700", "source": "global_var"},
			}),
			CachePolicy:      "ttl",
			CacheTTLSeconds:  intPtr(60),
			ResponseJSONPath: strPtr("$"),
			ExtraConfig:      mustJSON(map[string]interface{}{}),
			Enabled:          true,
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000202"),
			OwnerID:     ownerID,
			Name:        "股票历史 K 线",
			Description: strPtr("获取股票历史 K 线数据，支持日K、周K、月K。数据源：新浪财经"),
			Category:    strPtr("finance"),
			Tags:        mustJSON([]string{"finance", "stock", "history"}),
			Icon:        strPtr("Histogram"),
			DSType:      "HTTP_GET",
			URLTemplate: strPtr("https://quotes.sina.cn/cn/api/quotes.php?symbol={{stock_code}}&scale={{scale}}&datalen={{count}}"),
			HTTPMethod:  strPtr("GET"),
			ContentType: strPtr("application/json"),
			TimeoutMS:   15000,
			RetryCount:  2,
			AuthType:    "none",
			ParamsSchema: mustJSON([]map[string]interface{}{
				{"name": "stock_code", "type": "string", "required": true, "default_value": "sh600519", "description": "股票代码，如 sh600519", "source": "global_var"},
				{"name": "scale", "type": "string", "required": true, "default_value": "240", "description": "时间间隔(分钟): 240=日K 1200=周K", "source": "fixed_value"},
				{"name": "count", "type": "number", "required": true, "default_value": "30", "description": "返回条数", "source": "fixed_value"},
			}),
			CachePolicy:      "ttl",
			CacheTTLSeconds:  intPtr(300),
			ResponseJSONPath: strPtr("$"),
			ExtraConfig:      mustJSON(map[string]interface{}{}),
			Enabled:          true,
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000203"),
			OwnerID:     ownerID,
			Name:        "高考分数线查询",
			Description: strPtr("查询各省高考录取分数线，支持历年数据。需手动输入或对接教育考试院 API"),
			Category:    strPtr("education"),
			Tags:        mustJSON([]string{"education", "gaokao", "score"}),
			Icon:        strPtr("DocumentChecked"),
			DSType:      "MANUAL_INPUT",
			TimeoutMS:   5000,
			RetryCount:  0,
			AuthType:    "none",
			ParamsSchema: mustJSON([]map[string]interface{}{
				{"name": "province", "type": "string", "required": true, "default_value": "北京", "description": "省份名称", "source": "global_var"},
				{"name": "year", "type": "number", "required": true, "default_value": "2024", "description": "年份", "source": "global_var"},
				{"name": "batch", "type": "string", "required": true, "default_value": "本科一批", "description": "批次：本科一批/本科二批/专科批", "source": "global_var"},
			}),
			CachePolicy:      "none",
			ResponseJSONPath: strPtr("$"),
			ExtraConfig:      mustJSON(map[string]interface{}{}),
			Enabled:          true,
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000204"),
			OwnerID:     ownerID,
			Name:        "高校专业信息",
			Description: strPtr("查询高校及专业详细信息，包括招生计划、就业方向等。需手动输入或对接教育部阳光高考平台"),
			Category:    strPtr("education"),
			Tags:        mustJSON([]string{"education", "university", "major"}),
			Icon:        strPtr("School"),
			DSType:      "MANUAL_INPUT",
			TimeoutMS:   5000,
			RetryCount:  0,
			AuthType:    "none",
			ParamsSchema: mustJSON([]map[string]interface{}{
				{"name": "university_name", "type": "string", "required": true, "default_value": "清华大学", "description": "高校名称", "source": "global_var"},
				{"name": "major_name", "type": "string", "required": false, "default_value": "", "description": "专业名称（可选）", "source": "global_var"},
			}),
			CachePolicy:      "none",
			ResponseJSONPath: strPtr("$"),
			ExtraConfig:      mustJSON(map[string]interface{}{}),
			Enabled:          true,
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000205"),
			OwnerID:     ownerID,
			Name:        "手动输入数据",
			Description: strPtr("允许用户手动输入自定义数据，适用于静态分析场景"),
			Category:    strPtr("general"),
			Tags:        mustJSON([]string{"general", "manual", "input"}),
			Icon:        strPtr("EditPen"),
			DSType:      "MANUAL_INPUT",
			TimeoutMS:   5000,
			RetryCount:  0,
			AuthType:    "none",
			ParamsSchema: mustJSON([]map[string]interface{}{
				{"name": "content", "type": "string", "required": true, "default_value": "", "description": "手动输入的分析内容", "source": "runtime_input"},
			}),
			CachePolicy:      "none",
			ResponseJSONPath: strPtr("$"),
			ExtraConfig:      mustJSON(map[string]interface{}{}),
			Enabled:          true,
		},
		{
			ID:                  uuid.MustParse("00000000-0000-0000-0000-000000000206"),
			OwnerID:             ownerID,
			Name:                "新闻资讯聚合",
			Description:         strPtr("获取指定关键词的最新新闻资讯。需配置 NewsAPI Key"),
			Category:            strPtr("general"),
			Tags:                mustJSON([]string{"news", "general", "api"}),
			Icon:                strPtr("News"),
			DSType:              "HTTP_GET",
			URLTemplate:         strPtr("https://newsapi.org/v2/everything?q={{keyword}}&sortBy=publishedAt&pageSize={{limit}}&language=zh"),
			HTTPMethod:          strPtr("GET"),
			ContentType:         strPtr("application/json"),
			TimeoutMS:           10000,
			RetryCount:          2,
			AuthType:            "api_key_header",
			AuthConfigEncrypted: strPtr(`{"header_name":"X-Api-Key","api_key":"[PLEASE_SET_NEWSAPI_KEY]"}`),
			ParamsSchema: mustJSON([]map[string]interface{}{
				{"name": "keyword", "type": "string", "required": true, "default_value": "人工智能", "description": "搜索关键词", "source": "global_var"},
				{"name": "limit", "type": "number", "required": true, "default_value": "10", "description": "返回条数(最大100)", "source": "fixed_value"},
			}),
			CachePolicy:      "ttl",
			CacheTTLSeconds:  intPtr(300),
			ResponseJSONPath: strPtr("$.articles"),
			ExtraConfig:      mustJSON(map[string]interface{}{}),
			Enabled:          true,
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000207"),
			OwnerID:     ownerID,
			Name:        "汇率查询",
			Description: strPtr("获取实时汇率数据，支持多种货币对。数据源： exchangerate-api.com"),
			Category:    strPtr("finance"),
			Tags:        mustJSON([]string{"finance", "exchange", "rate"}),
			Icon:        strPtr("Money"),
			DSType:      "HTTP_GET",
			URLTemplate: strPtr("https://api.exchangerate-api.com/v4/latest/{{base_currency}}"),
			HTTPMethod:  strPtr("GET"),
			ContentType: strPtr("application/json"),
			TimeoutMS:   10000,
			RetryCount:  2,
			AuthType:    "none",
			ParamsSchema: mustJSON([]map[string]interface{}{
				{"name": "base_currency", "type": "string", "required": true, "default_value": "CNY", "description": "基础货币代码，如 CNY、USD、EUR", "source": "global_var"},
			}),
			CachePolicy:      "ttl",
			CacheTTLSeconds:  intPtr(3600),
			ResponseJSONPath: strPtr("$.rates"),
			ExtraConfig:      mustJSON(map[string]interface{}{}),
			Enabled:          true,
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000208"),
			OwnerID:     ownerID,
			Name:        "天气查询",
			Description: strPtr("获取指定城市的实时天气和未来预报。数据源：Open-Meteo（无需API Key）"),
			Category:    strPtr("general"),
			Tags:        mustJSON([]string{"weather", "general", "free"}),
			Icon:        strPtr("Sunny"),
			DSType:      "HTTP_GET",
			URLTemplate: strPtr("https://api.open-meteo.com/v1/forecast?latitude={{lat}}&longitude={{lon}}&current=temperature_2m,relative_humidity_2m,weather_code&daily=temperature_2m_max,temperature_2m_min&timezone=auto"),
			HTTPMethod:  strPtr("GET"),
			ContentType: strPtr("application/json"),
			TimeoutMS:   10000,
			RetryCount:  2,
			AuthType:    "none",
			ParamsSchema: mustJSON([]map[string]interface{}{
				{"name": "lat", "type": "number", "required": true, "default_value": "39.9042", "description": "纬度，如北京 39.9042", "source": "global_var"},
				{"name": "lon", "type": "number", "required": true, "default_value": "116.4074", "description": "经度，如北京 116.4074", "source": "global_var"},
			}),
			CachePolicy:      "ttl",
			CacheTTLSeconds:  intPtr(600),
			ResponseJSONPath: strPtr("$"),
			ExtraConfig:      mustJSON(map[string]interface{}{}),
			Enabled:          true,
		},
	}
	for _, s := range sources {
		if err := db.WithContext(ctx).Create(&s).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedAgents creates built-in agents with proper model and data source associations.
// Each agent is linked to a specific LLM model and optionally a data source.
func seedAgents(ctx context.Context, db *gorm.DB, ownerID uuid.UUID) error {
	// Use DeepSeek-V3 as default model for all agents (ID: 00000000-0000-0000-0000-000000000103)
	defaultModelID := uuid.MustParse("00000000-0000-0000-0000-000000000103")

	agents := []model.Agent{
		{
			ID:       uuid.MustParse("00000000-0000-0000-0000-000000000301"),
			OwnerID:  ownerID,
			Name:     "股票分析师",
			RoleDesc: strPtr("专业股票分析师，擅长技术分析和基本面分析"),
			Tags:     mustJSON([]string{"finance", "stock", "analysis"}),
			Icon:     strPtr("TrendCharts"),
			SystemPrompt: `你是一位资深股票分析师，拥有 CFA 和 FRM 双证。请基于提供的数据，从以下维度进行分析：
1. 技术面分析：K线形态、均线系统、成交量、MACD/KDJ指标
2. 基本面分析：PE/PB估值、营收增长、净利润、ROE
3. 市场情绪：资金流向、龙虎榜、融资融券
4. 风险提示：波动率、黑天鹅事件、政策风险
5. 投资建议：短期/中期/长期策略

请用 Markdown 格式输出专业分析报告，包含数据表格和关键指标。`,
			LLMModelID:     &defaultModelID,
			DataSourceID:   uuidPtr(uuid.MustParse("00000000-0000-0000-0000-000000000201")),
			ParamMappings:  mustJSON([]map[string]interface{}{}),
			OutputFormat:   "markdown",
			OutputLang:     "zh-CN",
			MaxOutputChars: 8000,
			Enabled:        true,
		},
		{
			ID:       uuid.MustParse("00000000-0000-0000-0000-000000000302"),
			OwnerID:  ownerID,
			Name:     "行业研究员",
			RoleDesc: strPtr("行业研究专家，擅长产业链分析和竞争格局研判"),
			Tags:     mustJSON([]string{"finance", "industry", "research"}),
			Icon:     strPtr("OfficeBuilding"),
			SystemPrompt: `你是一位行业研究总监，专注于产业链深度分析。请基于提供的数据，完成以下分析：
1. 行业概况：市场规模、增长率、生命周期阶段
2. 竞争格局：CR5集中度、护城河分析、波特五力
3. 产业链：上游供应商议价能力、下游客户结构
4. 政策环境：监管政策、补贴政策、国际贸易影响
5. 未来趋势：技术变革、替代品威胁、新进入者

输出格式为 Markdown，要求逻辑清晰、数据支撑充分。`,
			LLMModelID:     &defaultModelID,
			DataSourceID:   uuidPtr(uuid.MustParse("00000000-0000-0000-0000-000000000202")),
			ParamMappings:  mustJSON([]map[string]interface{}{}),
			OutputFormat:   "markdown",
			OutputLang:     "zh-CN",
			MaxOutputChars: 8000,
			Enabled:        true,
		},
		{
			ID:       uuid.MustParse("00000000-0000-0000-0000-000000000303"),
			OwnerID:  ownerID,
			Name:     "高考志愿规划师",
			RoleDesc: strPtr("高考志愿填报专家，熟悉各省录取政策和高校信息"),
			Tags:     mustJSON([]string{"education", "gaokao", "advisor"}),
			Icon:     strPtr("School"),
			SystemPrompt: `你是一位资深高考志愿填报规划师，拥有 15 年经验。请基于考生信息和高校数据，提供以下分析：
1. 分数定位：根据省排名定位可报考院校层次
2. 院校推荐：冲-稳-保三档院校列表及录取概率
3. 专业分析：就业前景、薪资水平、深造率
4. 风险评估：滑档风险、调剂风险、退档风险
5. 填报策略：平行志愿排序建议、梯度设计

请用 Markdown 格式输出，包含表格和关键数据。`,
			LLMModelID:     &defaultModelID,
			DataSourceID:   uuidPtr(uuid.MustParse("00000000-0000-0000-0000-000000000203")),
			ParamMappings:  mustJSON([]map[string]interface{}{}),
			OutputFormat:   "markdown",
			OutputLang:     "zh-CN",
			MaxOutputChars: 8000,
			Enabled:        true,
		},
		{
			ID:       uuid.MustParse("00000000-0000-0000-0000-000000000304"),
			OwnerID:  ownerID,
			Name:     "教育专家",
			RoleDesc: strPtr("教育政策研究专家，擅长教育趋势分析和政策解读"),
			Tags:     mustJSON([]string{"education", "policy", "research"}),
			Icon:     strPtr("Reading"),
			SystemPrompt: `你是一位教育政策研究专家，专注于中国高等教育发展研究。请基于提供的数据，完成以下分析：
1. 政策解读：最新高考改革政策、强基计划、综合评价
2. 院校对比：985/211/双一流对比、学科评估结果
3. 就业分析：各专业就业率、平均薪资、行业分布
4. 深造路径：保研率、考研难度、出国留学趋势
5. 个性化建议：根据考生兴趣和能力匹配最优路径

输出格式为 Markdown，语言通俗易懂。`,
			LLMModelID:     &defaultModelID,
			DataSourceID:   uuidPtr(uuid.MustParse("00000000-0000-0000-0000-000000000204")),
			ParamMappings:  mustJSON([]map[string]interface{}{}),
			OutputFormat:   "markdown",
			OutputLang:     "zh-CN",
			MaxOutputChars: 8000,
			Enabled:        true,
		},
		{
			ID:       uuid.MustParse("00000000-0000-0000-0000-000000000305"),
			OwnerID:  ownerID,
			Name:     "风险评审员",
			RoleDesc: strPtr("风险评审专家，擅长合规性审查和风险评估"),
			Tags:     mustJSON([]string{"risk", "compliance", "review"}),
			Icon:     strPtr("Warning"),
			SystemPrompt: `你是一位资深风险评审专家。请对提供的分析结果进行独立的风险评审，重点检查：
1. 合规性：是否符合相关法律法规和监管要求
2. 准确性：数据来源是否可靠、结论是否有充分依据
3. 完整性：是否遗漏重要风险因素
4. 偏见：是否存在确认偏误、幸存者偏差等认知偏差
5. 建议质量：建议是否可操作、是否有明确的风险提示

请用 Markdown 格式输出评审报告，对每个维度给出评分（低/中/高/严重）和具体说明。`,
			LLMModelID:     &defaultModelID,
			DataSourceID:   uuidPtr(uuid.MustParse("00000000-0000-0000-0000-000000000205")),
			ParamMappings:  mustJSON([]map[string]interface{}{}),
			OutputFormat:   "markdown",
			OutputLang:     "zh-CN",
			MaxOutputChars: 8000,
			Enabled:        true,
		},
		{
			ID:       uuid.MustParse("00000000-0000-0000-0000-000000000306"),
			OwnerID:  ownerID,
			Name:     "报告撰写员",
			RoleDesc: strPtr("专业报告撰写员，擅长将分析结果整理为结构化报告"),
			Tags:     mustJSON([]string{"report", "writing", "summary"}),
			Icon:     strPtr("Document"),
			SystemPrompt: `你是一位专业报告撰写员。请将各智能体的分析结果汇总整理为一份结构完整、逻辑清晰的综合报告：
1. 执行摘要：核心结论和建议（200字以内）
2. 背景介绍：分析目标和数据来源
3. 详细分析：各维度分析结果的整合与对比
4. 风险提示：关键风险点和不确定性
5. 行动建议：具体可执行的建议和优先级排序

要求语言专业、格式规范，使用 Markdown 格式，包含必要的表格和列表。`,
			LLMModelID:     &defaultModelID,
			DataSourceID:   uuidPtr(uuid.MustParse("00000000-0000-0000-0000-000000000205")),
			ParamMappings:  mustJSON([]map[string]interface{}{}),
			OutputFormat:   "markdown",
			OutputLang:     "zh-CN",
			MaxOutputChars: 12000,
			Enabled:        true,
		},
	}
	for _, ag := range agents {
		if err := db.WithContext(ctx).Create(&ag).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedWorkflows creates built-in workflows with proper agent and model associations.
// Each workflow node references a valid agent ID and the workflow has a default model.
func seedWorkflows(ctx context.Context, db *gorm.DB, ownerID uuid.UUID) error {
	// Default model for workflows (DeepSeek-V3)
	defaultModelID := uuid.MustParse("00000000-0000-0000-0000-000000000103")

	workflows := []struct {
		wf     model.Workflow
		params []map[string]interface{}
		nodes  []map[string]interface{}
		edges  []map[string]interface{}
	}{
		{
			wf: model.Workflow{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000401"),
				OwnerID:     ownerID,
				Name:        "股票深度分析",
				Description: strPtr("对指定股票进行技术面、基本面、行业深度分析，并生成专业投资报告"),
				Tags:        mustJSON([]string{"finance", "stock", "analysis"}),
				Visibility:  "public",
				Version:     1,
				ExecConfig:  mustJSON(map[string]interface{}{"max_debate_rounds": 2, "timeout_ms": 300000, "retry_count": 2, "max_concurrent": 4, "debug_mode": false}),
				DefaultModelID: &defaultModelID,
				Archived:    false,
			},
			params: []map[string]interface{}{
				{"key": "stock_code", "label": "股票代码", "type": "string", "required": true, "default_value": "sh600519", "description": "如 sh600519、sz000001、hk00700", "sort_order": 0},
				{"key": "analysis_period", "label": "分析周期", "type": "select", "required": true, "default_value": "30", "options": []string{"7", "30", "90", "180"}, "description": "分析天数", "sort_order": 1},
				{"key": "investment_style", "label": "投资风格", "type": "select", "required": true, "default_value": "value", "options": []string{"value", "growth", "balanced"}, "description": "价值投资/成长投资/均衡", "sort_order": 2},
			},
			nodes: []map[string]interface{}{
				{"id": "start", "type": "start", "label": "开始", "position": map[string]int{"x": 100, "y": 300}, "data": map[string]interface{}{}},
				{"id": "agent_tech", "type": "agent_run", "label": "技术面分析", "position": map[string]int{"x": 300, "y": 200}, "data": map[string]interface{}{"agent_id": "00000000-0000-0000-0000-000000000301", "timeout_ms": 120000}},
				{"id": "agent_fund", "type": "agent_run", "label": "基本面分析", "position": map[string]int{"x": 300, "y": 400}, "data": map[string]interface{}{"agent_id": "00000000-0000-0000-0000-000000000301", "timeout_ms": 120000}},
				{"id": "agent_industry", "type": "agent_run", "label": "行业研究", "position": map[string]int{"x": 500, "y": 300}, "data": map[string]interface{}{"agent_id": "00000000-0000-0000-0000-000000000302", "timeout_ms": 120000}},
				{"id": "debate", "type": "debate", "label": "多维度辩论", "position": map[string]int{"x": 700, "y": 300}, "data": map[string]interface{}{"max_rounds": 2, "stop_condition": "max_rounds", "agent_ids": []string{"00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000302"}}},
				{"id": "risk", "type": "risk_review", "label": "风险评审", "position": map[string]int{"x": 900, "y": 200}, "data": map[string]interface{}{"risk_dimensions": "合规性,准确性,完整性", "risk_threshold": "medium"}},
				{"id": "summarize", "type": "summarize", "label": "汇总报告", "position": map[string]int{"x": 1100, "y": 300}, "data": map[string]interface{}{"summary_prompt": "请将技术面分析、基本面分析和行业研究的结果汇总为一份完整的股票投资分析报告", "output_format": "markdown"}},
				{"id": "end", "type": "end", "label": "结束", "position": map[string]int{"x": 1300, "y": 300}, "data": map[string]interface{}{"report_title_template": "{{stock_code}} 股票深度分析报告"}},
			},
			edges: []map[string]interface{}{
				{"id": "e1", "source": "start", "target": "agent_tech"},
				{"id": "e2", "source": "start", "target": "agent_fund"},
				{"id": "e3", "source": "agent_tech", "target": "agent_industry"},
				{"id": "e4", "source": "agent_fund", "target": "agent_industry"},
				{"id": "e5", "source": "agent_industry", "target": "debate"},
				{"id": "e6", "source": "debate", "target": "risk"},
				{"id": "e7", "source": "risk", "target": "summarize"},
				{"id": "e8", "source": "summarize", "target": "end"},
			},
		},
		{
			wf: model.Workflow{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000402"),
				OwnerID:     ownerID,
				Name:        "高考志愿填报分析",
				Description: strPtr("基于考生分数和意向，提供个性化高考志愿填报建议和专业分析报告"),
				Tags:        mustJSON([]string{"education", "gaokao", "advisor"}),
				Visibility:  "public",
				Version:     1,
				ExecConfig:  mustJSON(map[string]interface{}{"max_debate_rounds": 2, "timeout_ms": 300000, "retry_count": 2, "max_concurrent": 4, "debug_mode": false}),
				DefaultModelID: &defaultModelID,
				Archived:    false,
			},
			params: []map[string]interface{}{
				{"key": "province", "label": "省份", "type": "select", "required": true, "default_value": "北京", "options": []string{"北京", "上海", "广东", "江苏", "浙江", "四川", "湖北", "河南", "山东", "安徽"}, "description": "考生所在省份", "sort_order": 0},
				{"key": "score", "label": "高考分数", "type": "number", "required": true, "default_value": "600", "description": "高考总分", "sort_order": 1},
				{"key": "rank", "label": "省排名", "type": "number", "required": true, "default_value": "5000", "description": "全省排名", "sort_order": 2},
				{"key": "subject", "label": "选科", "type": "select", "required": true, "default_value": "物理+化学+生物", "options": []string{"物理+化学+生物", "物理+化学+地理", "物理+生物+政治", "历史+政治+地理", "历史+化学+生物"}, "description": "选考科目组合", "sort_order": 3},
				{"key": "interests", "label": "兴趣方向", "type": "textarea", "required": false, "default_value": "计算机、人工智能、金融", "description": "感兴趣的专业或方向", "sort_order": 4},
			},
			nodes: []map[string]interface{}{
				{"id": "start", "type": "start", "label": "开始", "position": map[string]int{"x": 100, "y": 300}, "data": map[string]interface{}{}},
				{"id": "agent_score", "type": "agent_run", "label": "分数定位分析", "position": map[string]int{"x": 300, "y": 200}, "data": map[string]interface{}{"agent_id": "00000000-0000-0000-0000-000000000303", "timeout_ms": 120000}},
				{"id": "agent_school", "type": "agent_run", "label": "院校推荐", "position": map[string]int{"x": 300, "y": 400}, "data": map[string]interface{}{"agent_id": "00000000-0000-0000-0000-000000000304", "timeout_ms": 120000}},
				{"id": "agent_policy", "type": "agent_run", "label": "政策解读", "position": map[string]int{"x": 500, "y": 300}, "data": map[string]interface{}{"agent_id": "00000000-0000-0000-0000-000000000304", "timeout_ms": 120000}},
				{"id": "debate", "type": "debate", "label": "方案辩论", "position": map[string]int{"x": 700, "y": 300}, "data": map[string]interface{}{"max_rounds": 2, "stop_condition": "max_rounds", "agent_ids": []string{"00000000-0000-0000-0000-000000000303", "00000000-0000-0000-0000-000000000304"}}},
				{"id": "risk", "type": "risk_review", "label": "风险评审", "position": map[string]int{"x": 900, "y": 200}, "data": map[string]interface{}{"risk_dimensions": "滑档风险,调剂风险,退档风险", "risk_threshold": "medium"}},
				{"id": "summarize", "type": "summarize", "label": "汇总报告", "position": map[string]int{"x": 1100, "y": 300}, "data": map[string]interface{}{"summary_prompt": "请将分数定位、院校推荐和政策解读的结果汇总为一份完整的高考志愿填报分析报告", "output_format": "markdown"}},
				{"id": "end", "type": "end", "label": "结束", "position": map[string]int{"x": 1300, "y": 300}, "data": map[string]interface{}{"report_title_template": "{{province}} {{score}}分 高考志愿填报分析报告"}},
			},
			edges: []map[string]interface{}{
				{"id": "e1", "source": "start", "target": "agent_score"},
				{"id": "e2", "source": "start", "target": "agent_school"},
				{"id": "e3", "source": "agent_score", "target": "agent_policy"},
				{"id": "e4", "source": "agent_school", "target": "agent_policy"},
				{"id": "e5", "source": "agent_policy", "target": "debate"},
				{"id": "e6", "source": "debate", "target": "risk"},
				{"id": "e7", "source": "risk", "target": "summarize"},
				{"id": "e8", "source": "summarize", "target": "end"},
			},
		},
		{
			wf: model.Workflow{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000403"),
				OwnerID:     ownerID,
				Name:        "通用多智能体分析",
				Description: strPtr("通用分析模板，支持自定义主题的多智能体协作分析"),
				Tags:        mustJSON([]string{"general", "template", "multi-agent"}),
				Visibility:  "public",
				Version:     1,
				ExecConfig:  mustJSON(map[string]interface{}{"max_debate_rounds": 3, "timeout_ms": 300000, "retry_count": 2, "max_concurrent": 4, "debug_mode": false}),
				DefaultModelID: &defaultModelID,
				Archived:    false,
			},
			params: []map[string]interface{}{
				{"key": "topic", "label": "分析主题", "type": "string", "required": true, "default_value": "新能源汽车行业分析", "description": "输入要分析的主题", "sort_order": 0},
				{"key": "context", "label": "背景信息", "type": "textarea", "required": false, "default_value": "", "description": "补充背景信息（可选）", "sort_order": 1},
			},
			nodes: []map[string]interface{}{
				{"id": "start", "type": "start", "label": "开始", "position": map[string]int{"x": 100, "y": 300}, "data": map[string]interface{}{}},
				{"id": "agent_a", "type": "agent_run", "label": "分析师A", "position": map[string]int{"x": 300, "y": 200}, "data": map[string]interface{}{"agent_id": "00000000-0000-0000-0000-000000000301", "timeout_ms": 120000}},
				{"id": "agent_b", "type": "agent_run", "label": "分析师B", "position": map[string]int{"x": 300, "y": 400}, "data": map[string]interface{}{"agent_id": "00000000-0000-0000-0000-000000000302", "timeout_ms": 120000}},
				{"id": "debate", "type": "debate", "label": "辩论", "position": map[string]int{"x": 500, "y": 300}, "data": map[string]interface{}{"max_rounds": 3, "stop_condition": "max_rounds", "agent_ids": []string{"00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000302"}}},
				{"id": "validate", "type": "cross_validate", "label": "交叉验证", "position": map[string]int{"x": 700, "y": 300}, "data": map[string]interface{}{"agent_ids": []string{"00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000302"}, "validate_dimensions": "准确性,完整性,一致性"}},
				{"id": "risk", "type": "risk_review", "label": "风险评审", "position": map[string]int{"x": 900, "y": 200}, "data": map[string]interface{}{"risk_dimensions": "合规性,准确性,偏见", "risk_threshold": "medium"}},
				{"id": "summarize", "type": "summarize", "label": "汇总", "position": map[string]int{"x": 1100, "y": 300}, "data": map[string]interface{}{"summary_prompt": "请将多位分析师的分析结果和辩论结论汇总为一份综合报告", "output_format": "markdown"}},
				{"id": "end", "type": "end", "label": "结束", "position": map[string]int{"x": 1300, "y": 300}, "data": map[string]interface{}{"report_title_template": "{{topic}} 分析报告"}},
			},
			edges: []map[string]interface{}{
				{"id": "e1", "source": "start", "target": "agent_a"},
				{"id": "e2", "source": "start", "target": "agent_b"},
				{"id": "e3", "source": "agent_a", "target": "debate"},
				{"id": "e4", "source": "agent_b", "target": "debate"},
				{"id": "e5", "source": "debate", "target": "validate"},
				{"id": "e6", "source": "validate", "target": "risk"},
				{"id": "e7", "source": "risk", "target": "summarize"},
				{"id": "e8", "source": "summarize", "target": "end"},
			},
		},
	}

	for _, item := range workflows {
		item.wf.GlobalParams = mustJSON(item.params)
		item.wf.Nodes = mustJSON(item.nodes)
		item.wf.Edges = mustJSON(item.edges)
		if err := db.WithContext(ctx).Create(&item.wf).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedSystemConfig creates default system configuration entries.
func seedSystemConfig(ctx context.Context, db *gorm.DB) error {
	configs := []model.SystemConfig{
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000501"),
			Key:         "app_name",
			Value:       mustJSON("AgentFlow Pro"),
			Description: strPtr("应用名称"),
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000502"),
			Key:         "max_upload_size_mb",
			Value:       mustJSON(50),
			Description: strPtr("最大上传文件大小(MB)"),
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000503"),
			Key:         "default_debate_rounds",
			Value:       mustJSON(3),
			Description: strPtr("默认辩论轮次"),
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000504"),
			Key:         "max_workflow_nodes",
			Value:       mustJSON(50),
			Description: strPtr("工作流最大节点数"),
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000505"),
			Key:         "enable_registration",
			Value:       mustJSON(true),
			Description: strPtr("是否开放用户注册"),
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000506"),
			Key:         "audit_log_retention_days",
			Value:       mustJSON(90),
			Description: strPtr("审计日志保留天数"),
		},
	}
	for _, cfg := range configs {
		if err := db.WithContext(ctx).Create(&cfg).Error; err != nil {
			return err
		}
	}
	return nil
}

// Helper functions for creating pointers and JSON data

func intPtr(i int) *int {
	return &i
}

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
