import type { Project, GenerateResponse, Settings } from '@/types'

const BASE = '/api'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || `HTTP ${res.status}`)
  }
  return res.json()
}

export const api = {
  listProjects: () => request<Project[]>('/projects'),

  getProject: (slug: string) => request<Project>(`/projects/${slug}`),

  createProject: (project: Omit<Project, 'has_docs'>) =>
    request<Project>('/projects', {
      method: 'POST',
      body: JSON.stringify(project),
    }),

  updateProject: (slug: string, project: Omit<Project, 'has_docs'>) =>
    request<Project>(`/projects/${slug}`, {
      method: 'PUT',
      body: JSON.stringify(project),
    }),

  deleteProject: (slug: string) =>
    request<{ status: string }>(`/projects/${slug}`, { method: 'DELETE' }),

  generateDocs: (slug: string, opts?: { force_regen?: boolean; workers?: number }) =>
    request<GenerateResponse>(`/projects/${slug}/generate`, {
      method: 'POST',
      body: JSON.stringify(opts || {}),
    }),

  checkDocs: (slug: string) =>
    request<{ slug: string; exists: boolean; url: string }>(`/docs/${slug}`),

  getSettings: () => request<Settings>('/settings'),
}
