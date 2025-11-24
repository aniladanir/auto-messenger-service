package main

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	HttpPort           int           `json:"http_port"`
	DbConnString       string        `json:"db_conn_string"`
	RedisAddr          string        `json:"redis_addr"`
	WebHookUrl         string        `json:"webhook_url"`
	MsgBatchSize       int           `json:"msg_batch_size"`
	MsgSendIntervalStr string        `json:"msg_send_interval"`
	MsgSendInterval    time.Duration `json:"-"`
	MsgMaxRetry        int           `json:"msg_max_retry"`
}

// ReadConfigJson reads json formatted configuration from the given file
func ReadConfigJson(configFile string) (*Config, error) {
	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	cfg := new(Config)

	if err = json.Unmarshal(content, cfg); err != nil {
		return nil, err
	}

	cfg.MsgSendInterval, err = time.ParseDuration(cfg.MsgSendIntervalStr)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
