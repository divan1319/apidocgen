<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <header class="border-b border-border bg-surface sticky top-0 z-50">
    <div class="mx-auto max-w-5xl px-6 h-14 flex items-center justify-between">
      <router-link to="/" class="flex items-center gap-2 text-text font-bold text-sm tracking-wide hover:text-accent transition-colors">
        <span class="text-accent font-mono text-base">{'>'}</span>
        apidocgen
      </router-link>

      <nav class="flex items-center gap-4">
        <router-link
          to="/"
          class="text-text-muted text-xs font-medium hover:text-text transition-colors"
          active-class="!text-accent"
        >
          Proyectos
        </router-link>
        <button
          @click="router.push('/projects/new')"
          class="bg-accent text-bg text-xs font-semibold px-3 py-1.5 rounded-md hover:opacity-90 transition-opacity cursor-pointer"
        >
          + Nuevo
        </button>
      </nav>
    </div>
  </header>
</template>
