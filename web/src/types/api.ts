export interface ApiResponse<T = any> {
  request_id: string
  code: number
  message: string
  data: T
  meta?: PaginationMeta
}

export interface PaginationMeta {
  page: number
  page_size: number
  total: number
}

export interface PaginationParams {
  page?: number
  page_size?: number
  keyword?: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  expires_in: number
  user: {
    id: string
    username: string
    email: string
    role: string
  }
}

export interface UserItem {
  id: string
  username: string
  email: string
  role: 'user' | 'creator' | 'admin'
  avatar_url?: string
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface DashboardStats {
  workflow_count: number
  agent_count: number
  task_running_count: number
  report_count: number
  success_rate_24h: number
  recent_tasks: TaskItem[]
}

export interface WorkflowItem {
  id: string
  name: string
  description?: string
  tags: string[]
  global_params: GlobalParam[]
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  exec_config: ExecConfig
  default_model_id?: string
  version: number
  visibility: 'private' | 'public' | 'shared'
  share_code?: string
  archived: boolean
  last_run_at?: string
  run_count: number
  created_at: string
  updated_at: string
}

export interface GlobalParam {
  key: string
  label: string
  type: 'string' | 'number' | 'date' | 'select' | 'multiselect' | 'textarea'
  required: boolean
  default_value?: string
  options?: string[]
  description?: string
  sort_order: number
}

export interface WorkflowNode {
  id: string
  type: NodeType
  label: string
  position: { x: number; y: number }
  data: Record<string, any>
}

export interface WorkflowEdge {
  id: string
  source: string
  target: string
  sourceHandle?: string
  targetHandle?: string
  label?: string
}

export type NodeType =
  | 'start'
  | 'agent_run'
  | 'parallel'
  | 'debate'
  | 'cross_validate'
  | 'risk_review'
  | 'condition'
  | 'summarize'
  | 'end'
  | 'transform'

export interface ExecConfig {
  max_debate_rounds?: number
  timeout_ms?: number
  retry_count?: number
  max_concurrent?: number
  debug_mode?: boolean
}

export interface AgentItem {
  id: string
  name: string
  role_desc?: string
  tags: string[]
  icon?: string
  system_prompt: string
  llm_model_id?: string
  datasource_id?: string
  param_mappings: ParamMapping[]
  output_format: 'plaintext' | 'markdown' | 'json'
  output_lang: 'zh-CN' | 'en-US'
  max_output_chars: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface ParamMapping {
  param_key: string
  source_type: 'global_var' | 'fixed_value' | 'expression'
  source_value: string
}

export interface DataSourceItem {
  id: string
  name: string
  description?: string
  category?: string
  tags: string[]
  icon?: string
  ds_type: 'HTTP_GET' | 'HTTP_POST' | 'FILE_UPLOAD' | 'MANUAL_INPUT' | 'WEBSOCKET_STREAM'
  url_template?: string
  http_method?: string
  content_type?: string
  timeout_ms: number
  retry_count: number
  headers?: Record<string, string>
  body_template?: Record<string, any>
  auth_type: 'none' | 'bearer' | 'api_key_header' | 'custom_header'
  auth_config?: Record<string, string>
  params_schema: ParamSchema[]
  cache_policy: 'none' | 'ttl' | 'fixed'
  cache_ttl_seconds?: number
  response_jsonpath?: string
  extra_config?: Record<string, any>
  uploaded_file_id?: string
  enabled: boolean
  last_tested_at?: string
  last_test_status?: string
  created_at: string
  updated_at: string
}

export interface ParamSchema {
  name: string
  type: 'string' | 'number' | 'date' | 'array'
  required: boolean
  default_value?: string
  description?: string
  source: 'global_var' | 'fixed_value' | 'runtime_input'
}

export interface LlmModelItem {
  id: string
  name: string
  vendor: string
  endpoint: string
  model_id: string
  api_key_masked?: string
  temperature: number
  max_tokens: number
  timeout_ms: number
  retry_count: number
  stream_enabled: boolean
  enabled: boolean
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface TaskItem {
  id: string
  workflow_id: string
  workflow_name?: string
  workflow_version: number
  owner_id: string
  input_params: Record<string, any>
  mode: 'normal' | 'debug'
  status: 'pending' | 'queued' | 'running' | 'completed' | 'failed' | 'stopped' | 'paused'
  report_id?: string
  error_message?: string
  error_step_id?: string
  started_at?: string
  finished_at?: string
  duration_ms?: number
  created_at: string
  updated_at: string
}

export interface TaskStepItem {
  id: string
  task_id: string
  node_id: string
  node_type: string
  agent_id?: string
  agent_name?: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped' | 'stopped'
  debate_round?: number
  input?: Record<string, any>
  output?: Record<string, any>
  tokens_used?: number
  error_message?: string
  started_at?: string
  finished_at?: string
}

export interface ReportItem {
  id: string
  task_id: string
  workflow_id: string
  workflow_name?: string
  owner_id: string
  title: string
  content_md: string
  agent_outputs: Record<string, any>
  debate_logs: DebateLog[]
  risk_reviews: RiskReview[]
  exec_logs: ExecLog[]
  input_snapshot: Record<string, any>
  status: 'completed' | 'failed'
  archived: boolean
  total_tokens?: number
  duration_ms?: number
  created_at: string
  updated_at: string
}

export interface DebateLog {
  round: number
  agent_id: string
  agent_name: string
  output: string
  timestamp: string
}

export interface RiskReview {
  dimension: string
  level: 'low' | 'medium' | 'high' | 'critical'
  summary: string
  timestamp: string
}

export interface ExecLog {
  node_id: string
  node_type: string
  action: string
  detail: string
  timestamp: string
}

export interface AuditLogItem {
  id: string
  user_id: string
  username?: string
  action: string
  resource_type?: string
  resource_id?: string
  detail?: Record<string, any>
  ip?: string
  user_agent?: string
  created_at: string
}

export interface SystemConfigItem {
  id: string
  cfg_key: string
  cfg_value: Record<string, any>
  description?: string
}

export interface ImportMatchReport {
  matched_agents: { import_name: string; local_id: string; local_name: string }[]
  matched_datasources: { import_name: string; local_id: string; local_name: string }[]
  matched_models: { import_name: string; local_id: string; local_name: string }[]
  missing_agents: { import_name: string; import_id: string }[]
  missing_datasources: { import_name: string; import_id: string }[]
  missing_models: { import_name: string; import_id: string }[]
}

export interface ShareInfo {
  share_code: string
  share_url: string
  expires_at?: string
}

export const NODE_TYPE_CONFIG: Record<NodeType, { label: string; icon: string; color: string }> = {
  start: { label: '开始', icon: 'VideoPlay', color: '#67c23a' },
  agent_run: { label: '智能体执行', icon: 'User', color: '#409eff' },
  parallel: { label: '并行', icon: 'CopyDocument', color: '#e6a23c' },
  debate: { label: '辩论', icon: 'ChatDotRound', color: '#f56c6c' },
  cross_validate: { label: '交叉验证', icon: 'CircleCheck', color: '#9c27b0' },
  risk_review: { label: '风险评审', icon: 'Warning', color: '#ff9800' },
  condition: { label: '条件分支', icon: 'Switch', color: '#00bcd4' },
  summarize: { label: '汇总', icon: 'Document', color: '#607d8b' },
  transform: { label: '数据转换', icon: 'SetUp', color: '#795548' },
  end: { label: '结束', icon: 'CircleClose', color: '#909399' },
}
