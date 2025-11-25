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
    console.log('Theme toggle element found:', this.themeToggle);
    // 当前主题 - 优先使用存储的主题，然后是系统主题，最后是默认值
    this.currentTheme = this.getStoredTheme() || this.getSystemTheme();
    console.log('Initial theme determined:', this.currentTheme);

    // 初始化
    this.init();
  }

  /**
   * 初始化主题切换功能
   */
  init() {
    // 设置初始主题（应用到DOM and update button）
    this.applyTheme(this.currentTheme);
    console.log('Initial theme applied');

    // 绑定点击事件
    if (this.themeToggle) {
      console.log('Adding click event listener to theme toggle button');
      this.themeToggle.addEventListener('click', () => {
        this.toggleTheme();
        console.log('Theme toggle clicked');
      });
    } else {
      console.error('Theme toggle button not found in DOM!');
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
   * 应用主题（不存储到localStorage，用于初始化）
   * @param {string} theme 要应用的主题 ('light' 或 'dark')
   */
  applyTheme(theme) {
    console.log(`Applying theme: ${theme}`); 
    // 更新当前主题
    this.currentTheme = theme;

    // 更新HTML元素上的data-theme属性
    document.documentElement.setAttribute('data-theme', theme);

    // 更新按钮内容
    this.updateButtonContent();
    
    // 触发主题变化事件，用于更新代码高亮等
    this.dispatchThemeChangeEvent();
  }

  /**
   * 设置主题
   * @param {string} theme 要设置的主题 ('light' 或 'dark')
   */
  setTheme(theme) {
    console.log(`Setting theme: ${theme}`);
    // 更新当前主题
    this.currentTheme = theme;

    // 更新HTML元素上的data-theme属性
    document.documentElement.setAttribute('data-theme', theme);

    // 更新按钮内容
    this.updateButtonContent();

    // 存储主题到localStorage
    localStorage.setItem('theme', theme);
    
    // 触发主题变化事件，用于更新代码高亮等
    this.dispatchThemeChangeEvent();
  }

  /**
   * 更新按钮内容（图标和文本）
   */
  updateButtonContent() {
    console.log('Updating button content, current theme:', this.currentTheme);
    if (!this.themeToggle) {
      console.error('Theme toggle button not available in updateButtonContent');
      return;
    }

    if (this.currentTheme === 'dark') {
      // 深色主题显示太阳图标和"Light"文本
      this.themeToggle.innerHTML = `${sunIcon} Light`;
      console.log('Set button to sun icon for light theme');
    } else {
      // 浅色主题显示月亮图标和"Dark"文本
      this.themeToggle.innerHTML = `${moonIcon} Dark`;
      console.log('Set button to moon icon for dark theme');
    }
  }

  /**
   * 切换主题
   */
  toggleTheme() {
    console.log('Toggle theme function called');
    const newTheme = this.currentTheme === 'dark' ? 'light' : 'dark';
    console.log('Current theme:', this.currentTheme, 'New theme:', newTheme);
    this.setTheme(newTheme);
  }
  
  /**
   * 触发主题变化事件
   */
  dispatchThemeChangeEvent() {
    document.dispatchEvent(new CustomEvent('themeChanged', {
      detail: { theme: this.currentTheme }
    }));
  }
}

// 存储全局实例，以便在需要时引用
let themeToggleInstance;

// 页面加载完成后初始化主题切换功能
document.addEventListener('DOMContentLoaded', () => {
  console.log('DOM Content Loaded, initializing ThemeToggle');
  themeToggleInstance = new ThemeToggle();
  console.log('ThemeToggle instance created:', themeToggleInstance);
});

// 监听系统主题偏好变化
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
  // 只有在没有存储主题时才响应系统主题变化
  if (!localStorage.getItem('theme')) {
    const newTheme = e.matches ? 'dark' : 'light';
    if (themeToggleInstance) {
      themeToggleInstance.setTheme(newTheme); // Apply and store the new theme preference
    }
  }
});

// 监听主题变化事件，确保代码高亮正确更新
document.addEventListener('DOMContentLoaded', () => {
  const themeObserver = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      if (mutation.type === 'attributes' && mutation.attributeName === 'data-theme') {
        // 主题已更改，触发代码高亮更新
        document.dispatchEvent(new CustomEvent('themeChanged', {
          detail: { theme: document.documentElement.getAttribute('data-theme') }
        }));
      }
    });
  });

  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme']
  });
});