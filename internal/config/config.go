package config

import (
    "github.com/spf13/viper"
    "log"
)

type Config struct {
    DBUrl string `mapstructure:"DB_URL"`
}

func LoadConfig() Config {
    viper.SetConfigFile(".env")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        log.Println("No .env file found, using env variables only")
    }

    var c Config
    if err := viper.Unmarshal(&c); err != nil {
        log.Fatal("config unmarshal error:", err)
    }

    log.Printf("DEBUG: Using DB_URL: %s", c.DBUrl)
    return c
}