const apps = [];

apps.push({
  name: "cmc_global_metrics",
  script: "cmc_global_metrics",
  exec_interpreter: "none",
  exec_mode: "fork",
  instances: 1,
  restart_delay: 5000, // 5 seconds
});

apps.push({
  name: "cmc_price_crawler",
  script: "cmc_price_crawler",
  exec_interpreter: "none",
  exec_mode: "fork",
  instances: 1,
  restart_delay: 5000, // 5 seconds
});

apps.push({
  name: "crawler_block_header",
  script: "crawler_block_header",
  exec_interpreter: "none",
  exec_mode: "fork",
  instances: 1,
  restart_delay: 5000, // 5 seconds
});

apps.push({
  name: "crawler_gas_price",
  script: "crawler_gas_price",
  exec_interpreter: "none",
  exec_mode: "fork",
  instances: 1,
  restart_delay: 5000, // 5 seconds
});

apps.push({
  name: "mark_price",
  script: "mark_price",
  exec_interpreter: "none",
  exec_mode: "fork",
  instances: 1,
  restart_delay: 5000, // 5 seconds
});

apps.push({
  name: "ftx_spot_price",
  script: "ftx_spot_price",
  exec_interpreter: "none",
  exec_mode: "fork",
  instances: 1,
  restart_delay: 5000, // 5 seconds
});

apps.push({
  name: "upload",
  script: "/usr/local/bin/upload.sh",
  exec_interpreter: "bash",
  exec_mode: "fork_mode",
  instances: 1,
  restart_delay: 5000, // 5 seconds
});

module.exports = {
  apps,
};
