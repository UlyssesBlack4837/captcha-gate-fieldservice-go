package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	fieldservice "captcha-gate-fieldservice"
)

type request struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	Name           string `json:"name"`
	CaptchaToken   string `json:"captcha_token"`
	WidgetRecordID string `json:"widget_record_id"`
	IP             string `json:"ip"`
}

func main() {
	var in request
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		log.Fatal(err)
	}
	client, err := fieldservice.NewCaptchaClient()
	if err != nil {
		log.Fatal(err)
	}
	service := fieldservice.SignupService{Captcha: client}
	result, err := service.Signup(context.Background(), fieldservice.SignupRequest{Email: in.Email, Password: in.Password, Name: in.Name, CaptchaToken: in.CaptchaToken, WidgetRecordID: in.WidgetRecordID, IP: in.IP})
	if err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		log.Fatal(err)
	}
}
