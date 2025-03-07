package i18n

import (
	"fmt"
	"strings"
	"sync"
)

// 支持的语言
const (
	ZH_CN = "zh-CN"
	EN_US = "en-US"
)

// 默认语言
const DefaultLanguage = EN_US

// I18n 国际化接口
type I18n interface {
	// Translate 翻译文本
	Translate(key string, lang string) string
	// TranslateWithParams 带参数的翻译
	TranslateWithParams(key string, lang string, params map[string]interface{}) string
}

// Manager 国际化管理器
type Manager struct {
	translations map[string]map[string]string
	mutex        sync.RWMutex
}

// NewManager 创建国际化管理器
func NewManager() *Manager {
	return &Manager{
		translations: make(map[string]map[string]string),
	}
}

// LoadTranslations 加载翻译
func (m *Manager) LoadTranslations(lang string, translations map[string]string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.translations[lang] == nil {
		m.translations[lang] = make(map[string]string)
	}
	
	for key, value := range translations {
		m.translations[lang][key] = value
	}
}

// Translate 翻译文本
func (m *Manager) Translate(key string, lang string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// 如果语言不存在，使用默认语言
	if _, ok := m.translations[lang]; !ok {
		lang = DefaultLanguage
	}
	
	// 如果翻译不存在，返回键名
	if translation, ok := m.translations[lang][key]; ok {
		return translation
	}
	
	// 如果在指定语言中找不到翻译，尝试在默认语言中查找
	if lang != DefaultLanguage {
		if translation, ok := m.translations[DefaultLanguage][key]; ok {
			return translation
		}
	}
	
	return key
}

// TranslateWithParams 带参数的翻译
func (m *Manager) TranslateWithParams(key string, lang string, params map[string]interface{}) string {
	text := m.Translate(key, lang)
	
	// 简单的参数替换实现
	for k, v := range params {
		placeholder := "{" + k + "}"
		text = replaceAll(text, placeholder, toString(v))
	}
	
	return text
}

// 全局国际化管理器实例
var globalManager *Manager
var once sync.Once

// GetManager 获取全局国际化管理器实例
func GetManager() *Manager {
	once.Do(func() {
		globalManager = NewManager()
		// 加载默认翻译
		loadDefaultTranslations(globalManager)
	})
	return globalManager
}

// T 翻译快捷方法
func T(key string, lang string) string {
	return GetManager().Translate(key, lang)
}

// TP 带参数的翻译快捷方法
func TP(key string, lang string, params map[string]interface{}) string {
	return GetManager().TranslateWithParams(key, lang, params)
}

// 辅助函数：将任意类型转换为字符串
func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	
	switch v := value.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// 辅助函数：替换字符串中的所有匹配项
func replaceAll(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

// 加载默认翻译
func loadDefaultTranslations(manager *Manager) {
	// 加载英文翻译
	manager.LoadTranslations(EN_US, enTranslations)
	
	// 加载中文翻译
	manager.LoadTranslations(ZH_CN, zhTranslations)
} 