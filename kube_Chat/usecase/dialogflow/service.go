package dialogflow

import (
	dg "cloud.google.com/go/dialogflow/apiv2"
	"context"
	"fmt"
	"github.com/leopardslab/kubebot/kube_Chat/entity"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
	"log"
)

type CommandDetect struct {
	Message   *entity.Message
	Client    *dg.SessionsClient
	ProjectID string
}

func NewFLowAgent(credentialJSON, projectID string, msg *entity.Message) (*CommandDetect, error) {
	client, err := dg.NewSessionsClient(context.Background(), option.WithCredentialsFile(credentialJSON))
	if err != nil {
		return nil, err
	}
	return &CommandDetect{
		Message:   msg,
		Client:    client,
		ProjectID: projectID,
	}, nil
}

func (c *CommandDetect) DetectCommand() (*entity.CommandContext, error) {

	resp, err := c.Client.DetectIntent(context.Background(), &dialogflow.DetectIntentRequest{
		Session: fmt.Sprintf("projects/%s/agent/sessions/%s", c.ProjectID, c.Message.Sender),
		QueryInput: &dialogflow.QueryInput{
			Input: &dialogflow.QueryInput_Text{Text: &dialogflow.TextInput{
				Text:         c.Message.Content,
				LanguageCode: "en",
			}},
		},
	})
	if err != nil {
		return nil, err
	}
	result := resp.GetQueryResult().FulfillmentText

	log.Printf("DialogFlow Response : %s", result)

	//TODO add additional logic to call Command builder and check for requirements of the command

	return nil, nil // Change the return.
}
