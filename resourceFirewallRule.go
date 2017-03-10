package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	kapacitorClient "github.com/influxdata/kapacitor/client/v1"
)

func resourceFirewallRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallRuleCreate,
		Read:   resourceFirewallRuleRead,
		Update: resourceFirewallRuleUpdate,
		Delete: resourceFirewallRuleDelete,
		Schema: map[string]*schema.Schema{
			"cluster_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"rules": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceFirewallRuleCreate(data *schema.ResourceData, meta interface{}) error {
	client := meta.(*kapacitorClient.Client)

	createTaskOptions, err := getCreateTaskOptions(data)
	if err != nil {
		return err
	}

	task, err := client.CreateTask(createTaskOptions)
	if err != nil {
		return err
	}

	data.Set("id", task.ID)
	data.SetId(task.ID)

	return resourceFirewallRuleRead(data, meta)
}

func resourceFirewallRuleRead(data *schema.ResourceData, meta interface{}) error {
	client := meta.(*kapacitorClient.Client)

	task, err := client.Task(client.TaskLink(data.Get("id").(string)), nil)
	if err != nil {
		return err
	}

	if task.ID != "" {
		taskTypeBytes, err := task.Type.MarshalText()
		if err != nil {
			return err
		}

		taskStatusBytes, err := task.Status.MarshalText()
		if err != nil {
			return err
		}

		data.Set("type", string(taskTypeBytes))
		data.Set("database_retention_policies", serializeDatabaseRetentionPolicies(task.DBRPs))
		data.Set("tick_script", strings.Trim(task.TICKscript, "\n"))
		data.Set("status", string(taskStatusBytes))
		data.Set("id", task.ID)
	} else {
		// if we got here, then that means the script no longer exists.
		data.SetId("")
	}

	return nil
}

func resourceFirewallRuleUpdate(data *schema.ResourceData, meta interface{}) error {
	client := meta.(*kapacitorClient.Client)

	id := data.Get("id").(string)
	createTaskOptions, err := getCreateTaskOptions(data)
	if err != nil {
		return err
	}

	_, err = client.UpdateTask(client.TaskLink(id), kapacitorClient.UpdateTaskOptions{
		ID:         id,
		Type:       createTaskOptions.Type,
		TICKscript: createTaskOptions.TICKscript,
		Status:     createTaskOptions.Status,
	})
	if err != nil && !strings.Contains(err.Error(), "invalid response: code 204") {
		return err
	}

	return resourceFirewallRuleRead(data, meta)
}

func resourceFirewallRuleDelete(data *schema.ResourceData, meta interface{}) error {
	client := meta.(*kapacitorClient.Client)

	err := client.DeleteTask(client.TaskLink(data.Get("id").(string)))
	if err != nil {
		return err
	}

	data.SetId("")

	return nil
}

func getCreateTaskOptions(data *schema.ResourceData) (kapacitorClient.CreateTaskOptions, error) {
	taskTypeString := data.Get("type").(string)
	var taskType kapacitorClient.TaskType
	err := taskType.UnmarshalText([]byte(taskTypeString))

	if err != nil {
		return kapacitorClient.CreateTaskOptions{}, err
	}

	tickScript := data.Get("tick_script").(string)
	databaseRetentionPolicySliceInterface := data.Get("database_retention_policies").([]interface{})

	dbrps, err := parseDatabaseRetentionPolicies(databaseRetentionPolicySliceInterface)
	if err != nil {
		return kapacitorClient.CreateTaskOptions{}, err
	}

	taskStatusString := data.Get("status").(string)
	var taskStatus kapacitorClient.TaskStatus
	err = taskStatus.UnmarshalText([]byte(taskStatusString))
	if err != nil {
		return kapacitorClient.CreateTaskOptions{}, err
	}

	return kapacitorClient.CreateTaskOptions{
			Type:       taskType,
			DBRPs:      dbrps,
			TICKscript: tickScript,
			Status:     taskStatus,
		},
		nil
}

func serializeDatabaseRetentionPolicies(dbrps []kapacitorClient.DBRP) []string {
	databaseRetentionPolicyStrings := make([]string, len(dbrps))
	for i, dbrp := range dbrps {
		databaseRetentionPolicyStrings[i] = dbrp.String()
		// if strings.Contains(dbrp.Database, ".") || strings.Contains(dbrp.RetentionPolicy, ".") {
		// 	databaseRetentionPolicyStrings[i] = fmt.Sprintf("%q.%q", dbrp.Database, dbrp.RetentionPolicy)
		// } else {
		// 	databaseRetentionPolicyStrings[i] = fmt.Sprintf("%s.%s", dbrp.Database, dbrp.RetentionPolicy)
		// }
	}
	return databaseRetentionPolicyStrings
}

func parseDatabaseRetentionPolicies(databaseRetentionPolicySliceInterface []interface{}) ([]kapacitorClient.DBRP, error) {
	databaseRetentionPolicies := make([]kapacitorClient.DBRP, len(databaseRetentionPolicySliceInterface))
	for i, databaseRetentionPolicyInterface := range databaseRetentionPolicySliceInterface {
		databaseRetentionPolicy, err := parseDatabaseRetentionPolicy(databaseRetentionPolicyInterface.(string))
		if err != nil {
			return databaseRetentionPolicies, err
		}
		databaseRetentionPolicies[i] = databaseRetentionPolicy
	}
	return databaseRetentionPolicies, nil
}

func parseDatabaseRetentionPolicy(databaseRetentionPolicyString string) (kapacitorClient.DBRP, error) {
	split := strings.Split(databaseRetentionPolicyString, ".")
	if len(split) != 2 {
		return kapacitorClient.DBRP{},
			fmt.Errorf("error parsing databaseRetentionPolicy: %s. Expected form: \"my_db\".\"my_rp\"", databaseRetentionPolicyString)
	}
	return kapacitorClient.DBRP{
			Database:        strings.Replace(split[0], "\"", "", -1),
			RetentionPolicy: strings.Replace(split[1], "\"", "", -1),
		},
		nil
}
