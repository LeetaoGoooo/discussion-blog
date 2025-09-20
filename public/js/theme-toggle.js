/**
 * 主题切换功能
 * 处理主题切换按钮的SVG图标更新
 */

// SVG图标定义
const sunIcon = `
  <svg class="theme-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path d="M12 16C14.2091 16 16 14.2091 16 12C16 9.79086 14.2091 8 12 8C9.79086 8 8 9.79086 8 12C8 14.2091 9.79086 16 12 16Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M12 2V4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M12 20V22" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M4.93 4.93L6.34 6.34" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M17.66 17.66L19.07 19.07" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M2 12H4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M20 12H22" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M6.34 17.66L4.93 19.07" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M19.07 4.93L17.66 6.34" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
  </svg>
`;

const moonIcon = `
  <svg class="theme-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
  </svg>
`;

class ThemeToggle {
  constructor() {
    // 获取主题切换按钮
    this.themeToggle = document.getElementById('theme-toggle');
    // 当前主题
    this.currentTheme = this.getStoredTheme() || this.getSystemTheme();
    
    // 初始化
    this.init();
  }

  /**
   * 初始化主题切换功能
   */
  init() {
    // 设置初始主题
    this.setTheme(this.currentTheme);
    
    // 绑定点击事件
    if (this.themeToggle) {
      this.themeToggle.addEventListener('click', () => {
        this.toggleTheme();
      });
    }
  }

  /**
   * 获取存储的主题
   * @returns {string|null} 存储的主题
   */
  getStoredTheme() {
    return localStorage.getItem('theme');
  }

  /**
   * 获取系统主题偏好
   * @returns {string} 系统主题 ('light' 或 'dark')
   */
  getSystemTheme() {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  /**
   * 设置主题
   * @param {string} theme 要设置的主题 ('light' 或 'dark')
   */
  setTheme(theme) {
    // 更新当前主题
    this.currentTheme = theme;
    
    // 更新HTML元素上的data-theme属性
    document.documentElement.setAttribute('data-theme', theme);
    
    // 更新按钮内容
    this.updateButtonContent();
    
    // 存储主题到localStorage
    localStorage.setItem('theme', theme);
  }

  /**
   * 更新按钮内容（图标和文本）
   */
  updateButtonContent() {
    if (!this.themeToggle) return;
    
    if (this.currentTheme === 'dark') {
      // 深色主题显示太阳图标和"Light"文本
      this.themeToggle.innerHTML = `${sunIcon} Light`;
    } else {
      // 浅色主题显示月亮图标和"Dark"文本
      this.themeToggle.innerHTML = `${moonIcon} Dark`;
    }
  }

  /**
   * 切换主题
   */
  toggleTheme() {
    const newTheme = this.currentTheme === 'dark' ? 'light' : 'dark';
    this.setTheme(newTheme);
  }
}

// 页面加载完成后初始化主题切换功能
document.addEventListener('DOMContentLoaded', () => {
  new ThemeToggle();
});

// 监听系统主题偏好变化
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
  // 只有在没有存储主题时才响应系统主题变化
  if (!localStorage.getItem('theme')) {
    const newTheme = e.matches ? 'dark' : 'light';
    new ThemeToggle().setTheme(newTheme);
  }
});