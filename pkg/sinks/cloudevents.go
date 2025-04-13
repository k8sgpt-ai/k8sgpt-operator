package sinks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

var _ ISink = (*CloudEventsSink)(nil)

type CloudEventsSink struct {
	Endpoint string
	K8sGPT   string
	Client   Client
}

type CloudEventsData struct {
	Text        string                  `json:"text"`
	Attachments []CloudEventsAttachment `json:"attachments"`
}

type CloudEventsAttachment struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Color string `json:"color"`
	Title string `json:"title"`
}

func buildCloudEventsStructuredContent(kind, name, details, k8sgptCR string) cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetSource("https://github.com/k8sgpt-ai/k8sgpt-operator")
	event.SetType("com.github.k8sgpt-ai.k8sgpt-operator.sinks.cloudevents")
	event.SetTime(time.Now().UTC())
	event.SetID(uuid.NewString())
	event.SetData(cloudevents.ApplicationJSON, CloudEventsData{
		Text: fmt.Sprintf(">*[%s] K8sGPT analysis of the %s %s*", k8sgptCR, kind, name),
		Attachments: []CloudEventsAttachment{
			{
				Type:  "mrkdwn",
				Text:  details,
				Color: "danger",
				Title: "Report",
			},
		},
	})
	return event
}

func (s *CloudEventsSink) Configure(config v1alpha1.K8sGPT, c Client, sinkSecretValue string) {
	s.Endpoint = sinkSecretValue
	// check if the webhook url is passed as a sinkSecretValue, if not use spec.sink.webhook
	if s.Endpoint == "" {
		s.Endpoint = config.Spec.Sink.Endpoint
	}
	s.Client = c
	// take the name of the K8sGPT Custom ResourceRef
	s.K8sGPT = config.Name
}

func (s *CloudEventsSink) Emit(results v1alpha1.ResultSpec) error {
	event := buildCloudEventsStructuredContent(results.Kind, results.Name, results.Details, s.K8sGPT)
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.Endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/cloudevents+json")
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
