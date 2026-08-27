package config

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)


func NewLogger(env string) (*zap.Logger, error) {
	var l *zap.Logger
	var err error

	switch strings.ToLower(env) {
	case "prd":
		l, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	default:
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		l, err = cfg.Build()
		if err != nil {
			return nil, err
		}
	}


	return l, nil
}
