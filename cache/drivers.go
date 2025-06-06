package cache

import (
	"time"
	
	"github.com/gobeaver/beaver-kit/cache/driver/memory"
	"github.com/gobeaver/beaver-kit/cache/driver/redis"
)

// Driver registration functions

func memoryRegister(cfg Config) (Cache, error) {
	memCfg := memory.Config{
		MaxSize:         cfg.MaxSize,
		MaxKeys:         cfg.MaxKeys,
		DefaultTTL:      cfg.ParsedDefaultTTL(),
		CleanupInterval: cfg.ParsedCleanupInterval(),
		KeyPrefix:       cfg.KeyPrefix,
		Namespace:       cfg.Namespace,
	}
	
	return memory.New(memCfg)
}

func redisRegister(cfg Config) (Cache, error) {
	redisCfg := redis.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Password: cfg.Password,
		Database: cfg.Database,
		URL:      cfg.URL,
		
		MaxRetries:      cfg.MaxRetries,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(cfg.ConnMaxIdleTime) * time.Second,
		
		UseTLS:   cfg.UseTLS,
		CertFile: cfg.CertFile,
		KeyFile:  cfg.KeyFile,
		CAFile:   cfg.CAFile,
		
		KeyPrefix: cfg.KeyPrefix,
		Namespace: cfg.Namespace,
	}
	
	return redis.New(redisCfg)
}
