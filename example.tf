provider "kapacitor" {
    url = "http://localhost:9092/"
    timeout_seconds = 20
}

resource "kapacitor_tick_script" "test" {
  tick_script = "test"
  database_retention_policies = ["\"my_db\".\"my_retention_policy\""]
}
