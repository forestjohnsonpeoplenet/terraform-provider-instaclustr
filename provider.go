package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	kapacitorClient "github.com/influxdata/kapacitor/client/v1"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"kapacitor_tick_script": resourceTickScript(),
		},
		Schema: map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "https://api.instaclustr.com/provisioning/v1/",
				Optional: true,
			},
			"timeout_seconds": &schema.Schema{
				Type:     schema.TypeInt,
				Default:  15,
				Optional: true,
			},
			"auth_username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"auth_token": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
		ConfigureFunc: configure,
	}
}

func configure(data *schema.ResourceData) (interface{}, error) {

	config := kapacitorClient.Config{
		URL:                data.Get("url").(string),
		Timeout:            (time.Second * time.Duration(data.Get("timeout_seconds").(int))),
		InsecureSkipVerify: data.Get("insecure_skip_verify").(bool),
	}

	if data.Get("auth_username").(string) != "" {
		credentials := kapacitorClient.Credentials{
			Method:   parseAuthenticationMethod(data.Get("auth_method").(string)),
			Username: data.Get("auth_username").(string),
			Password: data.Get("auth_password").(string),
			Token:    data.Get("auth_token").(string),
		}

		err := credentials.Validate()

		if err != nil {
			return nil, fmt.Errorf("error validating credentials: %s", err)
		}

		config.Credentials = &credentials
	}

	client, err := kapacitorClient.New(config)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %s", err)
	}

	_, _, err = client.Ping()
	if err != nil {
		return nil, fmt.Errorf("error pinging server: %s", err)
	}

	return client, err
}

func parseAuthenticationMethod(methodString string) kapacitorClient.AuthenticationMethod {
	switch methodString {
	case "BearerAuthentication":
		return kapacitorClient.BearerAuthentication
	}
	return kapacitorClient.UserAuthentication
}
