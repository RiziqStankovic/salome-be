package main

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

func main() {
	ctx := context.TODO()
	client := resend.NewClient("re_f9QGSeaa_NVc51ck3MTEqS832WwnrEwxv")

	params := &resend.SendEmailRequest{
		From:    "Acme <noreply@salome2.cloudfren.id>",
		To:      []string{"deltastankovic99@gmail.com"},
		Subject: "hello world",
		Html:    "<p>it works!</p>",
	}

	sent, err := client.Emails.SendWithContext(ctx, params)

	if err != nil {
		panic(err)
	}
	fmt.Println(sent.Id)
}
