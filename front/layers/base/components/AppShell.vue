<script setup lang="ts">
import { useRoute } from 'vue-router'

// Каркас ЕХД: левая сворачиваемая навигация (бренд-navy) + верхняя панель + рабочая область.
// Пока пункты статичные; позже дерево приходит из GET /api/v1/reporter/navigation.
const route = useRoute()
const collapsed = ref(false)

const sections = [
  {
    title: 'Обзор',
    items: [{ label: 'Панель мониторинга', icon: 'pi pi-th-large', to: '/' }],
  },
  {
    title: 'Reporter',
    items: [{ label: 'Витрины', icon: 'pi pi-table', to: '/reporter' }],
  },
]

const isActive = (to: string) => (to === '/' ? route.path === '/' : route.path.startsWith(to))

const userMenu = ref()
const userItems = [
  { label: 'Профиль', icon: 'pi pi-user' },
  { label: 'Настройки', icon: 'pi pi-cog' },
  { separator: true },
  { label: 'Выйти', icon: 'pi pi-sign-out' },
]
const toggleUserMenu = (e: Event) => userMenu.value.toggle(e)

const pageTitle = computed(() => {
  for (const s of sections) for (const i of s.items) if (isActive(i.to)) return i.label
  return 'ЕХД'
})
</script>

<template>
  <div class="shell" :class="{ collapsed }">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-mark">ЕХД</span>
        <span v-if="!collapsed" class="brand-sub">Единое хранилище данных</span>
      </div>

      <nav class="nav">
        <div v-for="section in sections" :key="section.title" class="nav-section">
          <p v-if="!collapsed" class="nav-section-title">{{ section.title }}</p>
          <NuxtLink
            v-for="item in section.items"
            :key="item.to"
            :to="item.to"
            class="nav-item"
            :class="{ active: isActive(item.to) }"
            v-tooltip.right="collapsed ? item.label : undefined"
          >
            <i :class="item.icon" />
            <span v-if="!collapsed">{{ item.label }}</span>
          </NuxtLink>
        </div>
      </nav>

      <NuxtLink to="/login" class="nav-item nav-foot" v-tooltip.right="collapsed ? 'Вход' : undefined">
        <i class="pi pi-sign-in" />
        <span v-if="!collapsed">Вход в систему</span>
      </NuxtLink>
    </aside>

    <div class="main">
      <header class="topbar">
        <Button
          text
          rounded
          severity="secondary"
          :icon="collapsed ? 'pi pi-angle-double-right' : 'pi pi-angle-double-left'"
          :aria-label="collapsed ? 'Развернуть меню' : 'Свернуть меню'"
          @click="collapsed = !collapsed"
        />
        <h1 class="topbar-title">{{ pageTitle }}</h1>

        <div class="topbar-right">
          <OverlayBadge value="2" severity="danger">
            <Button text rounded severity="secondary" icon="pi pi-bell" aria-label="Уведомления" />
          </OverlayBadge>

          <button class="user-chip" aria-label="Меню пользователя" @click="toggleUserMenu">
            <Avatar label="А" shape="circle" class="user-avatar" />
            <span class="user-name">Администратор</span>
            <i class="pi pi-angle-down user-caret" />
          </button>
          <Menu ref="userMenu" :model="userItems" :popup="true" />
        </div>
      </header>

      <main class="content">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.shell {
  --sidebar-w: 264px;
  display: flex;
  min-height: 100vh;
}
.shell.collapsed {
  --sidebar-w: 76px;
}

/* Sidebar */
.sidebar {
  width: var(--sidebar-w);
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--ehd-brand);
  color: #dfe6f1;
  transition: width 0.18s ease;
  position: sticky;
  top: 0;
  height: 100vh;
}
.brand {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 1.15rem 1.25rem;
  min-height: 64px;
  justify-content: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}
.brand-mark {
  font-weight: 800;
  letter-spacing: 0.08em;
  font-size: 1.15rem;
  color: #fff;
}
.brand-sub {
  font-size: 0.72rem;
  color: rgba(255, 255, 255, 0.55);
}
.nav {
  flex: 1;
  padding: 0.75rem 0.6rem;
  overflow-y: auto;
}
.nav-section + .nav-section {
  margin-top: 1rem;
}
.nav-section-title {
  margin: 0 0 0.35rem;
  padding: 0 0.65rem;
  font-size: 0.68rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: rgba(255, 255, 255, 0.4);
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.62rem 0.7rem;
  margin-bottom: 2px;
  border-radius: var(--ehd-radius-sm);
  color: rgba(255, 255, 255, 0.78);
  text-decoration: none;
  font-size: 0.9rem;
  white-space: nowrap;
  transition: background 0.12s ease, color 0.12s ease;
}
.nav-item i {
  font-size: 1rem;
  width: 1.25rem;
  text-align: center;
}
.nav-item:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}
.nav-item.active {
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
  box-shadow: inset 3px 0 0 #fff;
}
.shell.collapsed .nav-item {
  justify-content: center;
  gap: 0;
}
.nav-foot {
  margin: 0.6rem;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0;
  padding-top: 0.9rem;
}

/* Main */
.main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  height: 64px;
  padding: 0 1.25rem;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--ehd-border);
}
.topbar-title {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--ehd-ink);
}
.topbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.user-chip {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  padding: 0.3rem 0.6rem 0.3rem 0.35rem;
  border: 1px solid var(--ehd-border);
  border-radius: 999px;
  background: #fff;
  cursor: pointer;
  transition: border-color 0.12s ease, box-shadow 0.12s ease;
}
.user-chip:hover {
  border-color: var(--p-primary-300);
  box-shadow: var(--ehd-shadow-sm);
}
.user-avatar {
  background: var(--ehd-brand);
  color: #fff;
  font-weight: 700;
  width: 1.9rem;
  height: 1.9rem;
  font-size: 0.85rem;
}
.user-name {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--ehd-ink-2);
}
.user-caret {
  font-size: 0.7rem;
  color: var(--ehd-muted);
}
.content {
  flex: 1;
  padding: 1.75rem;
  max-width: 1440px;
  width: 100%;
  margin: 0 auto;
}
</style>
