package log

import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/sns"
	"github.com/Loopring/relay/config"
)
import (
	"fmt"
	"time"
)

type SnsClient struct {
	innerClient *sns.SNS
	topicArn string
	valid bool
}

const region = "ap-northeast-1"

func NewSnsClient(options config.AwsServiceOption) *SnsClient {
	if len(options.SNSTopicArn) == 0 {
		Errorf("Sns TopicArn not set, will not init sns client")
		return &SnsClient{nil, nil, false}
	}
	//NOTE: use default config ~/.asw/credentials
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewSharedCredentials("", ""),
	})
	if err != nil {
		Errorf("new aws session failed \n", err.Error())
		return &SnsClient{nil, options.SNSTopicArn, false}
	} else {
		return &SnsClient{sns.New(sess), options.SNSTopicArn,true}
	}
}

func (client *SnsClient) PublishSns(subject string, message string) {
	if !client.valid {
		Error("SnsClient invalid, will not send message")
		return
	} else {
		input := &sns.PublishInput{}
		input.SetTopicArn(client.topicArn)
		input.SetSubject(subject)
		input.SetMessage(fmt.Sprintf("%s|%s",time.Now().Format("15:04:05"), message))
		_, err := client.innerClient.Publish(input)
		if err != nil {
			Errorf("Failed send sns message with error : %s\nSubject: %s\n, Message %s\n", err.Error(), subject, message)
		}
	}
}