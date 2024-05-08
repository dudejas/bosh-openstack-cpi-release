package utils

import "os"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . EnvVar
type EnvVar interface {
	Get(key string) string
}

type envVar struct{}

func NewEnvVar() EnvVar {
	return envVar{}
}

func (envVar) Get(key string) string {
	return os.Getenv(key)
}
