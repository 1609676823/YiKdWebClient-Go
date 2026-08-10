package yikdwebclient

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type xmlConfiguration struct {
	Entries []xmlConfigEntry `xml:"appSettings>add"`
}

type xmlConfigEntry struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// DefaultConfigPath returns YiKdWebCfg/appsettings.xml below the current directory.
func DefaultConfigPath() string {
	return filepath.Join("YiKdWebCfg", "appsettings.xml")
}

// GetAllCfgDic reads a .NET-style appSettings XML file. A blank path uses
// YiKdWebCfg/appsettings.xml below the current working directory.
func GetAllCfgDic(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configuration %q: %w", path, err)
	}
	var cfg xmlConfiguration
	if err := xml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse configuration %q: %w", path, err)
	}
	values := make(map[string]string, len(cfg.Entries))
	for _, entry := range cfg.Entries {
		if entry.Key == "" {
			continue
		}
		if _, exists := values[entry.Key]; !exists {
			values[entry.Key] = entry.Value
		}
	}
	return values, nil
}

// LoadAppSettings reads an appSettings XML file into AppSettingsModel.
func LoadAppSettings(path string) (*AppSettingsModel, error) {
	values, err := GetAllCfgDic(path)
	if err != nil {
		return nil, err
	}
	settings := &AppSettingsModel{
		XKDApiAcctID:    values["X-KDApi-AcctID"],
		XKDApiAppID:     values["X-KDApi-AppID"],
		XKDApiAppSec:    values["X-KDApi-AppSec"],
		XKDApiUserName:  values["X-KDApi-UserName"],
		XKDApiLCID:      values["X-KDApi-LCID"],
		XKDApiServerUrl: values["X-KDApi-ServerUrl"],
		XKDApiOrgNum:    values["X-KDApi-OrgNum"],
	}
	settings.Normalize()
	return settings, nil
}

// LoadAppSettingsIfExists returns empty settings when the configuration file
// does not exist, matching the original constructor's behavior.
func LoadAppSettingsIfExists(path string) (*AppSettingsModel, error) {
	settings, err := LoadAppSettings(path)
	if err == nil {
		return settings, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return &AppSettingsModel{}, nil
	}
	return nil, err
}
