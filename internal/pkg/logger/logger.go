package logger

import "go.uber.org/zap"

var L *zap.Logger

func Init(mode string) error {
	var err error
	if mode == "release" {
		L, err = zap.NewProduction()
	} else {
		L, err = zap.NewDevelopment()
	}
	return err
}

func Sync() { _ = L.Sync() }
