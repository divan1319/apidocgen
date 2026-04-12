<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Project, Settings } from '@/types'
import { api } from '@/api/client'

const route = useRoute()
const router = useRouter()

const isEdit = computed(() => !!route.params.slug)
const slug = computed(() => route.params.slug as string)

const form = ref<Omit<Project, 'has_docs'>>({
  name: '',
  slug: '',
  lang: 'laravel',
  routes: '',
  root: '',
  title: '',
  doc_lang: 'es',
})

const settings = ref<Settings>({ parsers: [], doc_langs: [] })
const loading = ref(false)
const saving = ref(false)
const error = ref('')

function slugify(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}

watch(() => form.value.name, (name) => {
  if (!isEdit.value) {
    form.value.slug = slugify(name)
  }
})

async function loadData() {
  loading.value = true
  try {
    settings.value = await api.getSettings()
    if (isEdit.value) {
      const p = await api.getProject(slug.value)
      form.value = {
        name: p.name,
        slug: p.slug,
        lang: p.lang,
        routes: p.routes,
        root: p.root,
        title: p.title,
        doc_lang: p.doc_lang,
      }
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleSubmit() {
  error.value = ''
  saving.value = true
  try {
    if (isEdit.value) {
      await api.updateProject(slug.value, form.value)
    } else {
      await api.createProject(form.value)
    }
    router.push('/')
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="max-w-xl mx-auto">
    <div class="mb-8">
      <router-link to="/" class="text-text-muted text-xs hover:text-text transition-colors">&larr; Volver a proyectos</router-link>
      <h1 class="text-2xl font-bold text-text mt-2">
        {{ isEdit ? 'Editar proyecto' : 'Nuevo proyecto' }}
      </h1>
    </div>

    <div v-if="error" class="bg-danger/10 border border-danger/30 text-danger text-sm rounded-lg px-4 py-3 mb-6">
      {{ error }}
    </div>

    <div v-if="loading" class="text-center py-20 text-text-muted text-sm">Cargando...</div>

    <form v-else @submit.prevent="handleSubmit" class="space-y-5">
      <!-- Name -->
      <div>
        <label class="block text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">Nombre</label>
        <input
          v-model="form.name"
          required
          class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-sm text-text font-mono placeholder:text-text-muted focus:outline-none focus:border-accent transition-colors"
          placeholder="Mi API Laravel"
        />
      </div>

      <!-- Slug -->
      <div>
        <label class="block text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">Slug</label>
        <input
          v-model="form.slug"
          :readonly="isEdit"
          class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-sm text-text-muted font-mono placeholder:text-text-muted focus:outline-none focus:border-accent transition-colors read-only:opacity-50"
          placeholder="mi-api-laravel"
        />
      </div>

      <!-- Lang -->
      <div>
        <label class="block text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">Framework / Lenguaje</label>
        <select
          v-model="form.lang"
          class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-sm text-text font-mono focus:outline-none focus:border-accent transition-colors"
        >
          <option v-for="p in settings.parsers" :key="p" :value="p">{{ p }}</option>
          <option v-if="!settings.parsers.length" value="laravel">laravel</option>
        </select>
      </div>

      <!-- Routes -->
      <div>
        <label class="block text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">Archivos de rutas</label>
        <input
          v-model="form.routes"
          required
          class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-sm text-text font-mono placeholder:text-text-muted focus:outline-none focus:border-accent transition-colors"
          placeholder="routes/api.php"
        />
        <p class="text-text-muted text-[10px] mt-1">Separar multiples archivos con coma</p>
      </div>

      <!-- Root -->
      <div>
        <label class="block text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">Directorio raiz del proyecto</label>
        <input
          v-model="form.root"
          required
          class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-sm text-text font-mono placeholder:text-text-muted focus:outline-none focus:border-accent transition-colors"
          placeholder="/path/to/project"
        />
      </div>

      <!-- Title -->
      <div>
        <label class="block text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">Titulo de la documentacion</label>
        <input
          v-model="form.title"
          class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-sm text-text font-mono placeholder:text-text-muted focus:outline-none focus:border-accent transition-colors"
          placeholder="API Documentation"
        />
      </div>

      <!-- Doc lang -->
      <div>
        <label class="block text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">Idioma de documentacion</label>
        <select
          v-model="form.doc_lang"
          class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-sm text-text font-mono focus:outline-none focus:border-accent transition-colors"
        >
          <option value="en">English</option>
          <option value="es">Español</option>
        </select>
      </div>

      <!-- Actions -->
      <div class="flex items-center gap-3 pt-4">
        <button
          type="submit"
          :disabled="saving"
          class="bg-accent text-bg text-sm font-semibold px-6 py-2.5 rounded-md hover:opacity-90 transition-opacity disabled:opacity-50 cursor-pointer"
        >
          {{ saving ? 'Guardando...' : (isEdit ? 'Guardar cambios' : 'Crear proyecto') }}
        </button>
        <router-link to="/" class="text-text-muted text-sm hover:text-text transition-colors">
          Cancelar
        </router-link>
      </div>
    </form>
  </div>
</template>
