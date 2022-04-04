scrape_interval = "60s"
scrape_timeout  = "10s"

// Create list of static discovery jobs. Each of these is a component with its
// own state that can be referenced from other components.
discovery "static" "robustperception-grafana" {
  hosts  = ["demo.robustperception.io:3000"]
  labels = { app = "grafana" }
}
discovery "static" "robustperception-prometheus" {
  hosts  = ["demo.robustperception.io:9090"]
  labels = { app = "prometheus" }
}
discovery "static" "robustperception-pushgateway" {
  hosts  = ["demo.robustperception.io:9091"]
  labels = { app = "pushgateway" }
}
discovery "static" "robustperception-alertmanager" {
  hosts  = ["demo.robustperception.io:9093"]
  labels = { app = "alertmanager" }
}
discovery "static" "robustperception-node_exporter" {
  hosts  = ["demo.robustperception.io:9100"]
  labels = { app = "node_exporter" }
}

// Create a chain discovery that can be used for merging and/or filtering
// targets. Here we just merge the targets into the final set.
discovery "chain" "robustperception" {
  // Define the targets to process as the concatenation of all the other
  // targets.
  //
  // This will be re-evaluated any time the input set of targets changes.
  input = concat(
    discovery.static.robustperception-grafana.targets,
    discovery.static.robustperception-prometheus.targets,
    discovery.static.robustperception-pushgateway.targets,
    discovery.static.robustperception-alertmanager.targets,
    discovery.static.robustperception-node_exporter.targets,
  )
}

scrape "robustperception" {
  // Scrape all the targets from discovery.chain.robustperception. This could
  // be replaced by doing the concat of the individual SDs here, but we still do
  // this as an example of chaining.
  targets = discovery.chain.robustperception.targets
}

remote_write "default" {
  url = "http://localhost:9009/api/prom/push"
}
