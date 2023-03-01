package utils

import (
	"crypto/sha256"
	"crypto/sha512"
	"github.com/Shopify/sarama"
	"github.com/xdg-go/scram"
)

type Algorithm string

type Config func(*sarama.Config)

const (
	SHA_256 Algorithm = "sha256"
	SHA_512 Algorithm = "sha512"
)

var (
	sHA256 scram.HashGeneratorFcn = sha256.New
	sHA512 scram.HashGeneratorFcn = sha512.New
)

type xDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func (x *xDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

func (x *xDGSCRAMClient) Step(challenge string) (response string, err error) {
	response, err = x.ClientConversation.Step(challenge)
	return
}

func (x *xDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}

func WithSASLAuth(user string, pwd string, algo string) Config {
	algorithm := Algorithm(algo)
	return func(conf *sarama.Config) {
		if user != "" && pwd != "" {
			conf.Net.SASL.Enable = true
			conf.Net.SASL.User = user
			conf.Net.SASL.Password = pwd
			conf.Net.SASL.Handshake = true
			if algorithm == SHA_256 {
				conf.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &xDGSCRAMClient{HashGeneratorFcn: sHA512} }
				conf.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
			} else if algorithm == SHA_512 {
				conf.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &xDGSCRAMClient{HashGeneratorFcn: sHA256} }
				conf.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			}
		}
	}
}
