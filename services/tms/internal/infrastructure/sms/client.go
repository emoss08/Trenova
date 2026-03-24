package sms

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Client struct {
	l      *zap.Logger
	client *twilio.RestClient
	cfg    *config.TwilioConfig
}

func New(p Params) *Client {
	cfg := p.Config.GetTwilioConfig()

	opts := twilio.ClientParams{
		Username: cfg.AccountSID,
		Password: cfg.AuthToken,
	}

	return &Client{
		l:      p.Logger.Named("infrastructure.sms"),
		client: twilio.NewRestClientWithParams(opts),
		cfg:    cfg,
	}
}

type SendRequest struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

func (c *Client) Send(req SendRequest) error {
	log := c.l.With(
		zap.String("operation", "send_sms"),
	)

	params := &twilioApi.CreateMessageParams{
		To:   &req.To,
		From: &c.cfg.FromNumber,
		Body: &req.Body,
	}

	resp, err := c.client.Api.CreateMessage(params)
	if err != nil {
		log.Error("failed to send SMS", zap.Error(err))
		return err
	}

	log.Debug("SMS sent", zap.Any("response", resp))
	return nil
}
