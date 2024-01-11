package sinks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

var _ ISink = (*MattermostSink)(nil)

type MattermostSink struct {
	Endpoint string
	K8sGPT   string
	Client   Client
	Channel  string
	UserName string
	IconURL  string
}

type MattermostMessage struct {
	Text        string       `json:"text"`
	Channel     string       `json:"channel,omitempty"`
	UserName    string       `json:"username,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	Attachments []attachment `json:"attachments"`
}

type attachment struct {
	Text  string `json:"text"`
	Color string `json:"color"`
	Title string `json:"title"`
}

func buildMattermostMessage(kind, name, details, k8sgptCR, channel, username, iconURL string) MattermostMessage {
	return MattermostMessage{
		Text:     fmt.Sprintf(">*[%s] K8sGPT analysis of the %s %s*", k8sgptCR, kind, name),
		Channel:  channel,
		UserName: username,
		IconURL:  iconURL,
		Attachments: []attachment{
			attachment{
				Text:  details,
				Color: "danger",
				Title: "Report",
			},
		},
	}
}

func (s *MattermostSink) Configure(config v1alpha1.K8sGPT, c Client, secret string) {
	_ = secret
	s.Endpoint = config.Spec.Sink.Endpoint
	// If no value is given, the default value of the webhook is used
	if config.Spec.Sink.Channel != "" {
		s.Channel = config.Spec.Sink.Channel
	}
	// If no value is given, the default value of the webhook is used
	if config.Spec.Sink.UserName != "" {
		s.UserName = config.Spec.Sink.UserName
	}
	// If no value is given, the default value of the webhook is used
	if config.Spec.Sink.IconURL != "" {
		s.IconURL = config.Spec.Sink.IconURL
	}
	s.Client = c
	// take the name of the K8sGPT Custom Resource
	s.K8sGPT = config.Name
}

func (s *MattermostSink) Emit(results v1alpha1.ResultSpec) error {
	details := ""
	// If AI is set to False, Details will not have a value, so if it is empty, use the Error text.
	if results.Details == "" && len(results.Error) > 0 {
		for i, v := range results.Error {
			details += fmt.Sprintf("%d. %s\n", i+1, v.Text)
		}
	} else {
		details = results.Details
	}
	message := buildMattermostMessage(
		results.Kind, results.Name, details, s.K8sGPT,
		s.Channel, s.UserName, s.IconURL,
	)
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
