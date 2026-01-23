<template>
  <t-layout>
    <t-header class="app-header">
      <div class="logo">
        <h1>LiteBoxd</h1>
      </div>
    </t-header>
    <t-content class="app-content">
      <router-view />
    </t-content>
    <t-footer class="app-footer">
      <a href="https://github.com/fslongjin/liteboxd" target="_blank" rel="noopener">
        GitHub
      </a>
      <span class="separator">|</span>
      <span>LiteBoxd &copy; 2025</span>
    </t-footer>
  </t-layout>
</template>

<script setup lang="ts">
</script>

<style>
body {
  margin: 0;
  padding: 0;
}

.app-header {
  display: flex;
  align-items: center;
  padding: 0 24px;
  background: var(--td-brand-color);
  color: white;
}

.logo h1 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
}

.app-content {
  min-height: calc(100vh - 64px - 40px);
  background: var(--td-bg-color-page);
}

.app-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 0 24px;
  height: 40px;
  background: var(--td-bg-color-container);
  border-top: 1px solid var(--td-component-border);
  font-size: 13px;
  color: var(--td-text-color-secondary);
}

.app-footer a {
  color: var(--td-brand-color);
  text-decoration: none;
}

.app-footer a:hover {
  text-decoration: underline;
}

.separator {
  color: var(--td-text-color-placeholder);
}
</style>
