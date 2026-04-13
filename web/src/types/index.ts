export interface Project {
  name: string
  slug: string
  lang: string
  routes: string
  root: string
  title: string
  doc_lang: string
  ai_provider?: string
  ai_model?: string
  ai_base_url?: string
  has_docs?: boolean
}

export interface GenerateResult {
  total_endpoints: number
  from_cache: number
  newly_documented: number
  failed: number
  output_path: string
}

export interface GenerateResponse {
  result: GenerateResult
  log: string
}

export interface AIProviderOption {
  id: string
  label: string
}

export interface Settings {
  parsers: string[]
  doc_langs: string[]
  ai_providers: AIProviderOption[]
}
