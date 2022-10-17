package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadConf(t *testing.T) {
	conf, err := LoadConf("config.yaml", "test")
	if err != nil {
		t.Fatalf("load conf err:%v", err)
	}
	output, _ := yaml.Marshal(conf)
	t.Logf("conf:%s", output)
	t.Logf("kafka servers:%v", conf.LogConf)
}
