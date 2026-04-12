<script setup lang="ts">
import Navbar from '@/components/Navbar.vue'
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <Navbar />
    <main class="flex-1 mx-auto w-full max-w-5xl px-6 py-10">
      <router-view />
    </main>
  </div>
</template>
