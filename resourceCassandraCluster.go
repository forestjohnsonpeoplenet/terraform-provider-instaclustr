package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCassandraCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceCassandraClusterCreate,
		Read:   resourceCassandraClusterRead,
		Update: resourceCassandraClusterUpdate,
		Delete: resourceCassandraClusterDelete,
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"provider": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "AWS_VPC",
				Optional: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"data_center": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"client_encryption": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "false",
				Optional: true,
			},
			"authn_authz": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "false",
				Optional: true,
			},
			"use_private_broadcast_rpc_address": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "true",
				Optional: true,
			},
			"default_network": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "true",
				Optional: true,
			},
			"rack_allocation": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
			},
			"firewall_rules": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     resourceFirewallRule(),
			},
			"nodes": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     resourceClusterNode(),
			},
			"vpc_peering_connections": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceVpcPeeringConnection(),
			},
		},
	}
}

func resourceCassandraClusterCreate(data *schema.ResourceData, meta interface{}) error {
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

	return resourceCassandraClusterRead(data, meta)
}

func resourceCassandraClusterRead(data *schema.ResourceData, meta interface{}) error {
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

func resourceCassandraClusterUpdate(data *schema.ResourceData, meta interface{}) error {
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

	return resourceCassandraClusterRead(data, meta)
}

func resourceCassandraClusterDelete(data *schema.ResourceData, meta interface{}) error {
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
