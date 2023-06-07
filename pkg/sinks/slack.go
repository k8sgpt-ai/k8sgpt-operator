package sinks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

var _ ISink = (*SlackSink)(nil)

type SlackSink struct {
	Endpoint string
	Client   Client
}

type SlackMessage struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Color string `json:"color"`
	Title string `json:"title"`
}

func buildSlackMessage(kind, name, details, backend string) SlackMessage {
	return SlackMessage{
		Text: fmt.Sprintf("`Analysis from %s of the %s %s`", backend, kind, name),
		Attachments: []Attachment{
			Attachment{
				Type:  "mrkdwn",
				Text:  details,
				Color: "danger",
				Title: "Report",
			},
		},
	}
}

func (s *SlackSink) Configure(config v1alpha1.K8sGPT, c Client) {
	if config.Spec.Sink == nil {
		s.Endpoint = ""
	}
	s.Endpoint = config.Spec.Sink.Endpoint
	s.Client = c
}

func (s *SlackSink) Emit(results v1alpha1.ResultSpec) error {
	if s.Endpoint == "" {
		// emit nothing
		return nil
	}

	message := buildSlackMessage(results.Kind, results.Name, results.Details, results.Backend)
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.Endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.hclient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send report: %s", resp.Status)
	}

	return nil
}
