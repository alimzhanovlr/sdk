package i18n

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Config holds i18n configuration
type Config struct {
	DefaultLanguage string
	SupportedLangs  []string
	Path            string
}

// I18n manages internationalization
type I18n struct {
	bundle          *i18n.Bundle
	defaultLanguage string
	supportedLangs  map[string]bool
}

// New creates a new i18n instance
func New(cfg Config) (*I18n, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	// Load language files
	for _, lang := range cfg.SupportedLangs {
		filename := filepath.Join(cfg.Path, fmt.Sprintf("%s.yaml", lang))
		if _, err := bundle.LoadMessageFile(filename); err != nil {
			// If file doesn't exist, continue (not all languages may be ready)
			continue
		}
	}

	supportedLangs := make(map[string]bool)
	for _, lang := range cfg.SupportedLangs {
		supportedLangs[lang] = true
	}

	return &I18n{
		bundle:          bundle,
		defaultLanguage: cfg.DefaultLanguage,
		supportedLangs:  supportedLangs,
	}, nil
}

// NewFromEmbed creates i18n from embedded files
func NewFromEmbed(cfg Config, fs embed.FS) (*I18n, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	for _, lang := range cfg.SupportedLangs {
		filename := filepath.Join(cfg.Path, fmt.Sprintf("%s.yaml", lang))
		data, err := fs.ReadFile(filename)
		if err != nil {
			continue
		}
		if _, err := bundle.ParseMessageFileBytes(data, filename); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
		}
	}

	supportedLangs := make(map[string]bool)
	for _, lang := range cfg.SupportedLangs {
		supportedLangs[lang] = true
	}

	return &I18n{
		bundle:          bundle,
		defaultLanguage: cfg.DefaultLanguage,
		supportedLangs:  supportedLangs,
	}, nil
}

// Localizer creates a localizer for a specific language
func (i *I18n) Localizer(lang string) *i18n.Localizer {
	if !i.supportedLangs[lang] {
		lang = i.defaultLanguage
	}
	return i18n.NewLocalizer(i.bundle, lang, i.defaultLanguage)
}

// T translates a message
func (i *I18n) T(lang, messageID string, templateData map[string]interface{}) string {
	localizer := i.Localizer(lang)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		return messageID
	}

	return msg
}

// GetSupportedLanguages returns list of supported languages
func (i *I18n) GetSupportedLanguages() []string {
	langs := make([]string, 0, len(i.supportedLangs))
	for lang := range i.supportedLangs {
		langs = append(langs, lang)
	}
	return langs
}

// IsSupported checks if language is supported
func (i *I18n) IsSupported(lang string) bool {
	return i.supportedLangs[lang]
}
