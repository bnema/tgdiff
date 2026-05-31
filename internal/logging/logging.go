package logging

import (
	"io"

	"github.com/bnema/zerowrap"
)

const AppName = "ero"

type Config struct {
	Level string
	Path  string
}

func Init(cfg Config) (zerowrap.Logger, string, func(), error) {
	level := cfg.Level
	if level == "" {
		level = "info"
	}

	levelForConfig := level
	if levelForConfig == "disabled" {
		levelForConfig = "info"
	}

	fileCfg := zerowrap.FileConfig{
		Enabled:    level != "disabled",
		Path:       cfg.Path,
		AppName:    AppName,
		Name:       "ero",
		Mode:       zerowrap.FileModeSingle,
		FileFormat: zerowrap.FileFormatJSON,
		MaxSize:    25,
		MaxBackups: 5,
		MaxAge:     14,
	}
	resolvedPath, err := zerowrap.ResolveLogPath(fileCfg)
	if err != nil {
		return zerowrap.Logger{}, "", func() {}, err
	}

	log, cleanup, err := zerowrap.NewWithFile(
		zerowrap.Config{Level: levelForConfig, Format: "json", Output: io.Discard, Caller: false},
		fileCfg,
	)
	if err != nil {
		return zerowrap.Logger{}, "", func() {}, err
	}

	return log, resolvedPath, cleanup, nil
}
