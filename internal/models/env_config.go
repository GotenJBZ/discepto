package models

import (
	"fmt"
	"os"
	"strconv"
)

type EnvConfig struct {
	GoogleClientID  string
	DatabaseURL     string
	PostsPerSeconds int
	Port            string
	SessionKey      []byte
	Debug           bool
}

func ReadEnvConfig() EnvConfig {
	debug := os.Getenv("DISCEPTO_DEBUG") == "true"
	port := os.Getenv("DISCEPTO_PORT")
	if port == "" {
		port = "23495"
	}
	postsPerMinute, err := strconv.Atoi(os.Getenv("DISCEPTO_POSTS_PER_MINUTE"))
	sessionKey := os.Getenv("DISCEPTO_SESSION_KEY")
	if err != nil {
		fmt.Println("Using default value for DISCEPTO_POSTS_PER_MINUTE")
		postsPerMinute = 2
	}
	return EnvConfig{
		GoogleClientID:  os.Getenv("DISCEPTO_GOOGLE_CLIENT_ID"),
		DatabaseURL:     os.Getenv("DISCEPTO_DATABASE_URL"),
		PostsPerSeconds: postsPerMinute,
		Port:            port,
		SessionKey:      []byte(sessionKey),
		Debug:           debug,
	}
}
