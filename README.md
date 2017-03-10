# terraform-provider-kapacitor

A Terraform plugin for managing TICK scripts in Kapacitor 

This is a simple Terraform plugin written following this guide: https://www.hashicorp.com/blog/terraform-custom-providers.html

Based on the Kapacitor HTTP API: https://docs.influxdata.com/kapacitor/v1.0//api/api/

#### features left unimplemented:

 * better integration with influxdb provider
  * influxdb provider needs ability to create users and retention policies
  * specify DB and retention policy as two strings rather than one wierdly formatted one
  * kapacitor server resource? not sure.
 * Kapacitor template support
 * UDFs
 * Other Kapacitor things that arent "tasks".
